package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/notificationsemail"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/alert"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
)

var sampleNow = time.Date(2024, 01, 19, 0, 0, 0, 0, time.Local)

func addDummyAlerts(m *alert.Model, t *time.Time) {

	sevs := []alertpb.Alert_Severity{
		alertpb.Alert_SEVERITY_UNSPECIFIED,
		alertpb.Alert_INFO,
		alertpb.Alert_WARNING,
		alertpb.Alert_SEVERE,
		alertpb.Alert_LIFE_SAFETY,
	}

	zones := []string{"East", "West"}

	for i := range 100 {
		m.AddAlert(&alertpb.Alert{
			Id:          fmt.Sprintf("alert Id : %d", i),
			Severity:    sevs[i%5],
			Description: fmt.Sprintf("test: %ds", i),
			Zone:        zones[i%2],
			Floor:       fmt.Sprintf("%d", i/4),
			Source:      "manual",
			CreateTime:  timestamppb.New(t.Add(-24 * 30 * time.Hour)),
			ResolveTime: nil,
		})
	}

	m.AddAlert(&alertpb.Alert{
		Id:          fmt.Sprintf("alert Id : %d", 555),
		Severity:    sevs[0],
		Description: fmt.Sprintf("test: %ds", 555),
		Zone:        zones[0],
		Floor:       fmt.Sprintf("%d", 1),
		Source:      "manual",
		CreateTime:  timestamppb.New(*t),
		ResolveTime: timestamppb.New(*t),
	})
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	root := node.New("testdevice01")

	m := alert.NewModel()
	// run this test in January to capture the edge case of previous month
	testTime := time.Date(
		2024, 01, 01, 00, 00, 00, 651387237, time.UTC)
	addDummyAlerts(m, &testTime)
	server := alert.NewModelServer(m)
	root.Announce(root.Name(), node.HasServer(alertpb.RegisterAlertApiServer, alertpb.AlertApiServer(server)))

	now, _ := time.Parse(time.DateTime, "2023-11-15 11:36:00")
	now = now.Round(time.Second) // get rid of millis, etc

	now = sampleNow

	serv := auto.Services{
		Logger: logger,
		Node:   root,
		Now: func() time.Time {
			return testTime
		},
	}

	lifecycle := notificationsemail.Factory.New(serv)
	defer lifecycle.Stop()
	cfg := `{
	"name": "emails", "type": "meteremail",
	"destination": {
	"host": "smtp.gmail.com",
	"from": "OCW Paradise Build <vantiocwdev@gmail.com>",
	"to": ["Dean Redfern <dean.redfern@vanti.co.uk>"],
	"passwordFile" : ".localpassword",
	"sendTime": "* * * * MON-FRI"
	},
	"subject" : "test alerts",
	"source" : "testdevice01",
	"templateArgs" : {
		"emailTitle" : "Test email title 54321"
	}
}`

	_, err = lifecycle.Configure([]byte(cfg))
	if err != nil {
		panic(err)
	}
	_, err = lifecycle.Start()
	if err != nil {
		panic(err)
	}

	time.Sleep(5 * time.Second)
}
