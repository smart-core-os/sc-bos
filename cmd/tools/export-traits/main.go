// Command export-traits reads various trait information and writes it to a CSV file.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/util/client"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

var clientConfig client.Config
var subsystems stringList
var traitNames stringList
var outFile string

type stringList []string

func (t *stringList) String() string {
	if t == nil {
		return ""
	}
	return strings.Join(*t, ",")
}

func (t *stringList) Set(s string) error {
	*t = append(*t, strings.Split(s, ",")...)
	return nil
}

func init() {
	flag.StringVar(&clientConfig.Endpoint, "endpoint", "localhost:23557", "smart core endpoint")
	flag.StringVar(&clientConfig.Name, "name", "", "smart core name we interrogate for list of devices")
	flag.BoolVar(&clientConfig.TLS.InsecureNoClientCert, "insecure-no-client-cert", false, "")
	flag.BoolVar(&clientConfig.TLS.InsecureSkipVerify, "insecure-skip-verify", false, "")
	flag.Var(&subsystems, "subsystem", "subsystems to export, supports multiple flags or comma separated values")
	flag.Var(&traitNames, "trait", "traits to export, supports multiple flags or comma separated values")
	flag.StringVar(&outFile, "out", "", "output file")
}

var getter = map[trait.Name]func(context.Context, grpc.ClientConnInterface, *devicespb.Device, *report) error{
	meter.TraitName: func(ctx context.Context, conn grpc.ClientConnInterface, device *devicespb.Device, r *report) error {
		apiClient := meterpb.NewMeterApiClient(conn)
		res, err := apiClient.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: device.Name})
		if err != nil {
			return err
		}
		r.addPoint(device.Name, "meter.usage", res.Usage)
		return nil
	},
	trait.Electric: func(ctx context.Context, conn grpc.ClientConnInterface, device *devicespb.Device, r *report) error {
		apiClient := traits.NewElectricApiClient(conn)
		res, err := apiClient.GetDemand(ctx, &traits.GetDemandRequest{Name: device.Name})
		if err != nil {
			return err
		}
		r.addPoint(device.Name, "electric.realPower", res.GetRealPower())
		r.addPoint(device.Name, "electric.apparentPower", res.GetApparentPower())
		r.addPoint(device.Name, "electric.reactivePower", res.GetReactivePower())
		r.addPoint(device.Name, "electric.powerFactor", res.GetPowerFactor())
		return nil
	},
}

func main() {
	flag.Parse()
	if outFile == "" {
		outFile = fmt.Sprintf("export-%s.csv", time.Now().Format("2006-01-02T15-04-05"))
	}
	if outFile == "console" {
		outFile = ""
	}

	ctx, cleanup := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cleanup()

	conn, err := client.NewConnection(clientConfig)
	if err != nil {
		panic(err)
	}

	devices, err := listDevices(ctx, conn)
	if err != nil {
		panic(err)
	}

	results := &report{}

	const concurrency = 100
	jobs := make(chan *devicespb.Device, concurrency)
	var jobsComplete sync.WaitGroup
	jobsComplete.Add(concurrency)
	for range concurrency {
		go func() {
			defer jobsComplete.Done()
			worker(ctx, conn, jobs, results)
		}()
	}

	for _, device := range devices {
		jobs <- device
	}
	close(jobs)
	jobsComplete.Wait()

	out := os.Stdout
	if outFile != "" {
		log.Println("Writing results to", outFile)
		out, err = os.Create(outFile)
		if err != nil {
			panic(err)
		}
		defer out.Close()
	}

	err = results.printResults(out)
	if err != nil {
		panic(err)
	}
}

func worker(ctx context.Context, conn grpc.ClientConnInterface, jobs <-chan *devicespb.Device, results *report) {
	for {
		select {
		case <-ctx.Done():
			return
		case device, ok := <-jobs:
			if !ok {
				return
			}
			for _, traitName := range traitNames {
				if !deviceHasTrait(device, traitName) {
					continue
				}
				if task, ok := getter[trait.Name(traitName)]; ok {
					if err := task(ctx, conn, device, results); err != nil {
						log.Printf("failed to get %s for %s: %v", traitName, device.Name, err)
					}
				}
			}
		}
	}
}

func deviceHasTrait(device *devicespb.Device, traitName string) bool {
	for _, tm := range device.GetMetadata().GetTraits() {
		if tm.Name == traitName {
			return true
		}
	}
	return false
}

func listDevices(ctx context.Context, conn *grpc.ClientConn) ([]*devicespb.Device, error) {
	apiClient := devicespb.NewDevicesApiClient(conn)
	req := &devicespb.ListDevicesRequest{
		PageSize: 1000,
		Query:    &devicespb.Device_Query{},
	}
	for _, subsystem := range subsystems {
		req.Query.Conditions = append(req.Query.Conditions, &devicespb.Device_Query_Condition{
			Field: "metadata.membership.subsystem",
			Value: &devicespb.Device_Query_Condition_StringEqualFold{StringEqualFold: subsystem},
		})
	}

	var allDevices []*devicespb.Device
	for {
		res, err := apiClient.ListDevices(ctx, req)
		if err != nil {
			return nil, err
		}
		allDevices = append(allDevices, res.Devices...)

		req.PageToken = res.NextPageToken
		if req.PageToken == "" {
			break
		}
	}

	return allDevices, nil
}

type report struct {
	mu      sync.Mutex
	headers map[string]int
	rows    map[string][]string
}

func (r *report) addPoint(name, key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.headers == nil {
		r.headers = make(map[string]int)
	}
	if r.rows == nil {
		r.rows = make(map[string][]string)
	}
	if _, ok := r.headers[key]; !ok {
		l := len(r.headers)
		r.headers[key] = l
		for k, vs := range r.rows {
			r.rows[k] = append(vs, "")
		}
	}
	if _, ok := r.rows[name]; !ok {
		r.rows[name] = make([]string, len(r.headers))
	}
	var s string
	switch v := value.(type) {
	case float64:
		s = fmt.Sprintf("%f", v)
	case float32:
		s = fmt.Sprintf("%f", v)
	default:
		s = fmt.Sprint(v)
	}
	r.rows[name][r.headers[key]] = s
}

func (r *report) printResults(out io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	headers := make([]string, 0, len(r.headers))
	for k := range r.headers {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	if _, err := fmt.Fprintln(out, "name,"+strings.Join(headers, ",")); err != nil {
		return err
	}
	sortedRows := make([][]string, 0, len(r.rows))
	for name, row := range r.rows {
		sortedRow := make([]string, len(headers)+1) // +1 for name
		sortedRow[0] = name
		for i, h := range headers {
			sortedRow[i+1] = row[r.headers[h]]
		}
		sortedRows = append(sortedRows, sortedRow)
	}
	sort.Slice(sortedRows, func(i, j int) bool {
		return sortedRows[i][0] < sortedRows[j][0]
	})
	for _, row := range sortedRows {
		if _, err := fmt.Fprintln(out, strings.Join(row, ",")); err != nil {
			return err
		}
	}
	return nil
}
