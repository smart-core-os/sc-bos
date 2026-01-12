// Command export-alerts reads alerts from [alertpb.AlertApiClient] and writes them to a CSV file.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/util/client"
)

var clientConfig client.Config
var outFile string

func init() {
	flag.StringVar(&clientConfig.Endpoint, "endpoint", "localhost:23557", "smart core endpoint")
	flag.StringVar(&clientConfig.Name, "name", "", "smart core name we interrogate for list of devices")
	flag.BoolVar(&clientConfig.TLS.InsecureNoClientCert, "insecure-no-client-cert", false, "")
	flag.BoolVar(&clientConfig.TLS.InsecureSkipVerify, "insecure-skip-verify", false, "")
	flag.StringVar(&outFile, "out", "", "output file")

}

func main() {
	flag.Parse()
	if outFile == "" {
		outFile = fmt.Sprintf("alerts-%s.csv", time.Now().Format("2006-01-02T15-04-05"))
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

	apiClient := alertpb.NewAlertApiClient(conn)
	req := &alertpb.ListAlertsRequest{
		Name:     clientConfig.Name,
		PageSize: 1000,
		Query: &alertpb.Alert_Query{
			// Resolved: &falseVal,
			Subsystem: "lighting",
		},
	}

	var alerts []*alertpb.Alert
	for {
		res, err := apiClient.ListAlerts(ctx, req)
		if err != nil {
			log.Println("Error reading page:", err)
			break
		}
		alerts = append(alerts, res.Alerts...)
		req.PageToken = res.NextPageToken
		if req.PageToken == "" {
			break
		}
	}

	out := os.Stdout
	if outFile != "" {
		log.Println("Writing results to", outFile)
		out, err = os.Create(outFile)
		if err != nil {
			panic(err)
		}
		defer out.Close()
	}

	fmt.Fprintln(out, strings.Join([]string{
		"Source",
		"Severity",
		"Create Time",
		"Resolve Time",
		"Message",
		"Floor",
		"Zone",
		"Subsystem",
		"Federation",
	}, ","))
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Source < alerts[j].Source
	})
	for _, alert := range alerts {
		var resolveTimeStr string
		if alert.ResolveTime != nil {
			resolveTimeStr = alert.ResolveTime.AsTime().Format(time.RFC3339)
		}
		fmt.Fprintln(out, strings.Join([]string{
			csvEscape(alert.Source),
			csvEscape(alert.Severity.String()),
			csvEscape(alert.CreateTime.AsTime().Format(time.RFC3339)),
			csvEscape(resolveTimeStr),
			csvEscape(alert.Description),
			csvEscape(alert.Floor),
			csvEscape(alert.Zone),
			csvEscape(alert.Subsystem),
			csvEscape(alert.Federation),
		}, ","))
	}
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

var (
	falseVal = false
)
