// Command bacnet-multi-comm-test checks the availability [bacnet] devices and objects read from [appconf] files.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/sc-bos/pkg/app/appconf"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

var (
	dir             = flag.String("dir", ".", "directory to scan for config")
	configFile      = flag.String("config-file", "area-controller.local.json", "file name to look for and load")
	resultsFile     = flag.String("results-file", "results", "file name to save results to, timestamp will be appended")
	devicesOnly     = flag.Bool("devices-only", false, "only check devices, not objects")
	discoverObjects = flag.Bool("discover-objects", false, "discover objects for devices that don't have them configured, overrides similar setting in config")
	timeout         = flag.Duration("timeout", time.Second*4, "timeout for requests")
	localPort       = flag.Int("local-port", 0, "local port to use for bacnet requests, overrides any found in config")
)

const concurrency = 10

var objectProperties = []property.ID{
	property.ObjectName,
	property.Description,
	// property.PresentValue,
	// property.NotificationClass,
	// property.EventEnable,
}
var deviceProperties = objectProperties[:2] // must be a strict prefix for the CSV to work
// Populated during scanning, includes properties that come from the configuration file.
var additionalProperties []property.ID

func main() {
	flag.Parse()
	start := time.Now()
	err := run()
	if err != nil {
		log.Printf("uhoh: %s", err)
	} else {
		elapsed := time.Since(start)
		log.Printf("done in %s", elapsed)
	}
}

// run will recursively scan dir for any files named configFile, and load any bacnet driver config from them.
// each bacnet device will be sent a request, and if successful will be marked "responding", and all configured
// objects will then be read
// results are saved to a CSV
func run() error {
	bacnetConfigs, err := loadConfigs()
	if err != nil {
		return err
	}
	results := make(map[string][]*result, len(bacnetConfigs))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	for key, cfg := range bacnetConfigs {
		select {
		case <-ctx.Done():
			continue
		default:
		}
		log.Printf("config: %s, devices: %d", key, len(cfg.Devices))

		port := int(cfg.LocalPort)
		if *localPort > 0 {
			port = *localPort
		}
		client, err := gobacnet.NewClient(cfg.LocalInterface, port, gobacnet.WithLogLevel(logrus.InfoLevel))
		if err != nil {
			return err
		}
		client.Log.Out = writer{client.Log.Out} // else client.Close() will close os.Stderr
		address, err := client.LocalUDPAddress()
		if err == nil {
			log.Printf("bacnet client configured: local=%s, localInterface=%s, localPort:%d", address, cfg.LocalInterface, cfg.LocalPort)
		}

		for _, device := range cfg.Devices {
			select {
			case <-ctx.Done():
				continue
			default:
			}
			res := &result{
				name:     device.Name,
				objectId: fmt.Sprintf("Device:%d", device.ID),
			}
			if comm := device.Comm; comm != nil {
				res.address = comm.IP.String()
				if des := comm.Destination; des != nil {
					res.network = des.Network
					res.mac = string(des.Address)
				}
			}

			switch {
			case res.mac != "":
				log.Printf("check device: ip:%s mac:%s net:%v id:%d", res.address, res.mac, res.network, device.ID)
			case res.address != "":
				log.Printf("check device: ip:%s id:%d", res.address, device.ID)
			default:
				log.Printf("check device: id:%d", device.ID)
			}

			var dev bactypes.Device
			{
				ctx, cancel := context.WithTimeout(ctx, *timeout)
				dev, err = bacnet.FindDevice(ctx, client, device)

				if err != nil {
					log.Printf("ERR: unable to find device %v : %v", device.ID, err)
				}
				if err == nil {
					res.responding = true
					readId := fmt.Sprintf("Device:%d", dev.ID.Instance)
					if readId != res.objectId {
						log.Printf("configured device %v is really %v", res.objectId, readId)
						res.objectId = readId
					}
					readObjectProps(ctx, client, dev, config.ObjectID(dev.ID), res, deviceProperties...)
				}
				cancel()
			}
			results[key] = append(results[key], res)
			if !res.responding || *devicesOnly {
				continue
			}

			cfgObjects := device.Objects
			// Discover device objects if we're asked to
			if shouldDiscoverObjects(cfg, device) {
				ctx, cancel := context.WithTimeout(ctx, (*timeout)*10)
				objects, err := client.Objects(ctx, dev)
				cancel()
				if err != nil {
					log.Printf("ERR: object discovery error for %v : %v", device.ID, err)
				}

				uniqueObjIds := make(map[string]bool, len(cfgObjects))
				for _, object := range cfgObjects {
					uniqueObjIds[object.ID.String()] = true
				}
				for _, obj := range objects.Objects {
					for _, object := range obj {
						id := object.ID
						if id == dev.ID {
							continue // don't duplicate the device
						}
						if !uniqueObjIds[id.String()] {
							uniqueObjIds[id.String()] = true
							// log.Printf("discovered device object %v:%v (name=%v)", id.Type, id.Instance, object.Name)
							cfgObjects = append(cfgObjects, config.Object{ID: config.ObjectID(id), Name: object.Name})
						}
					}
				}
			}
			// Fetch the object properties for the objects and record them in the results
			readAllObjectProps(ctx, client, dev, res, key, cfgObjects, results)
		}
		client.Close()
	}

	fileName := *resultsFile + "_" + time.Now().Format("2006-01-02T15_04") + ".csv"
	return writeResults(fileName, results)
}

func readAllObjectProps(ctx context.Context, client *gobacnet.Client, dev bactypes.Device, baseResult *result, key string, cfgObjects []config.Object, results map[string][]*result) {
	worker := func(jobs <-chan config.Object, results chan<- *result) {
		for obj := range jobs {
			// read both the hard coded props (in objectProperties) and any props defined in the config
			allProps := make([]property.ID, 0, len(objectProperties)+len(obj.Properties))
			if len(obj.Properties) == 0 {
				allProps = append(allProps, objectProperties...)
				allProps = append(allProps, property.PresentValue)
			}
			for _, prop := range obj.Properties {
				allProps = append(allProps, property.ID(prop.ID))
			}

			objRes := baseResult.child(obj.Name, obj.ID.String())
			ctx, cancel := context.WithTimeout(ctx, *timeout)
			readObjectProps(ctx, client, dev, obj.ID, objRes, allProps...)
			cancel()
			results <- objRes
		}
	}

	jobs := make(chan config.Object, concurrency)
	resultsChan := make(chan *result, len(cfgObjects))
	var jobsComplete sync.WaitGroup
	jobsComplete.Add(concurrency)
	for range concurrency {
		go func() {
			defer jobsComplete.Done()
			worker(jobs, resultsChan)
		}()
	}

	for _, device := range cfgObjects {
		jobs <- device
	}
	close(jobs)
	jobsComplete.Wait()
	close(resultsChan) // safe, nothing is sending here anymore
	for result := range resultsChan {
		results[key] = append(results[key], result)
	}
}

func readObjectProps(ctx context.Context, client *gobacnet.Client, dev bactypes.Device, objId config.ObjectID, res *result, props ...property.ID) {
	values, err := readProps(ctx, client, dev, objId, props...)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		log.Printf("ERR: unable to read properties %v %v : %v", objId, props, err)
	}
	if err == nil {
		res.responding = true
		res.properties = make(map[property.ID]any, len(props))
		for i, key := range props {
			res.properties[key] = values[i]
		}
	}
}

func loadConfigs() (map[string]config.Root, error) {
	var configPaths []string
	fileSystem := os.DirFS(*dir)
	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, *configFile) {
			configPaths = append(configPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Printf("configPaths: %s", strings.Join(configPaths, ";"))

	bacnetConfigs := make(map[string]config.Root, len(configPaths))

	for _, configPath := range configPaths {
		var localConfig appconf.Config
		rawLocalConfig, err := os.ReadFile(filepath.Join(*dir, configPath))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(rawLocalConfig, &localConfig)
		if err != nil {
			return nil, fmt.Errorf("%s JSON unmarshal: %w", configPath, err)
		}
		for _, driver := range localConfig.Drivers {
			if driver.Type == bacnet.DriverName {
				bacnetConfig, err := config.ReadBytes(driver.Raw)
				if err != nil {
					return nil, err
				}
				bacnetConfigs[configPath+":"+driver.Name] = bacnetConfig
			}
		}
	}
	return bacnetConfigs, nil
}

func readProps(ctx context.Context, client *gobacnet.Client, device bactypes.Device, objId config.ObjectID, props ...property.ID) ([]any, error) {
	objReq := bactypes.Object{
		ID: bactypes.ObjectID(objId),
	}
	for _, id := range props {
		objReq.Properties = append(objReq.Properties, bactypes.Property{ID: id, ArrayIndex: bactypes.ArrayAll})
	}
	req := bactypes.ReadMultipleProperty{
		Objects: []bactypes.Object{objReq},
	}
	res, err := client.ReadProperties(ctx, device, req)
	if err != nil {
		return nil, err
	}
	if len(res.Objects) == 0 {
		return nil, errors.New("zero length objects")
	}
	objRes := res.Objects[0]
	if len(objRes.Properties) == 0 {
		// Shouldn't happen, but has on occasion. I guess it depends how the device responds to our request
		return nil, errors.New("zero length object properties")
	}
	vals := make([]any, len(props))
	for i, prop := range objRes.Properties {
		if prop.ID != props[i] {
			return nil, fmt.Errorf("unexpected property: %v", prop.ID)
		}
		value := prop.Data
		if prop.ID == property.PresentValue && strings.HasPrefix(objId.String(), "Binary") {
			value = value == 1
		}
		vals[i] = value
	}
	return vals, nil
}

func writeResults(fileName string, results map[string][]*result) error {
	var rows [][]string
	header := []string{"Name", "Address", "Network", "MAC", "BACnet ID", "Responding"}
	seen := make(map[property.ID]struct{})
	for _, objectProperty := range objectProperties {
		header = append(header, objectProperty.String())
		seen[objectProperty] = struct{}{}
	}

	// calculate additional properties
	for _, keyedResults := range results {
		for _, result := range keyedResults {
			for id := range result.properties {
				if _, ok := seen[id]; !ok {
					seen[id] = struct{}{}
					additionalProperties = append(additionalProperties, id)
				}
			}
		}
	}
	slices.Sort(additionalProperties)
	for _, additionalProperty := range additionalProperties {
		header = append(header, additionalProperty.String())
	}

	rows = append(rows, header)
	for _, res := range results {
		for _, re := range res {
			rows = append(rows, re.toRow())
		}
	}
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	// clear the file
	err = f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	err = w.WriteAll(rows)
	if err != nil {
		return err
	}
	w.Flush()
	return nil
}

type result struct {
	name       string
	address    string
	network    uint16
	mac        string
	objectId   string
	responding bool

	// additional properties we've been asked for
	properties map[property.ID]any
}

func (r *result) child(name, id string) *result {
	return &result{
		name:       path.Join(r.name, name),
		address:    r.address,
		network:    r.network,
		mac:        r.mac,
		objectId:   id,
		responding: r.responding,
	}
}

func (r *result) toRow() []string {
	row := []string{r.name, r.address, netString(r.network), r.mac, r.objectId, boolYesNo(r.responding)}
	for _, key := range objectProperties {
		row = append(row, anyToString(r.properties[key]))
	}
	for _, key := range additionalProperties {
		row = append(row, anyToString(r.properties[key]))
	}
	return row
}

func anyToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case bool:
		return fmt.Sprintf("%t", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	case uint32:
		return fmt.Sprintf("%d", v)
	}
	return fmt.Sprintf("%v", value)
}

func netString(n uint16) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d", n)
}

func boolYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// writer wraps an io.Writer to remove any non-Writer methods.
type writer struct {
	io.Writer
}
