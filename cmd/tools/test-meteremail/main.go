// Command test-meteremail tests the [meteremail] package, sending to a real email address.
package main

import (
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/meteremail"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

var sampleNow = time.Date(2024, 01, 19, 0, 0, 0, 0, time.Local)

func addDummyMeters(root *node.Node) {
	meterNames := []string{"elecmeter1", "elecmeter2", "watermeter1", "watermeter2"}
	for _, meterName := range meterNames {
		m := meterpb.NewModel()
		m.RecordReading(123.45)
		root.Announce(meterName,
			node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(m))),
			node.HasTrait(meterpb.TraitName))
	}
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	root := node.New("test")
	addDummyMeters(root)

	now := sampleNow

	serv := auto.Services{
		Logger: logger,
		Node:   root,
		Now: func() time.Time {
			return now.Add(-2 * time.Second)
		},
	}
	lifecycle := meteremail.Factory.New(serv)
	defer lifecycle.Stop()
	cfg := `{
	"name": "emails", 
	"type": "meteremail",
	"destination": {
	"host": "smtp.gmail.com",
	"from": "OCW Paradise Build <vantiocwdev@gmail.com>",
	"to": ["Dean Redfern <dean.redfern@vanti.co.uk>", "Vanti OCW Dev <vantiocwdev@gmail.com>"],
	"passwordFile" : ".localpassword",
	"sendTime": "* * * * MON-FRI"
	},
	"electricMeters" : [
					"elecmeter1",
					"elecmeter2"
					],
	"waterMeters" : [ 
					"watermeter1",
					"watermeter2"
					],
	"timing" : {
		"timeout" : "9s",
		"backoffStart" : "19s",
		"backoffMax" : "59s",
		"numRetries" : 7
	},
	"templateArgs" : {
		"emailTitle" : "hello title",
		"subjectTemplate" : "hello subject"
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
