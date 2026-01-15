// Command test-meteremail tests the [meteremail] package, sending to a real email address.
package main

import (
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/meteremail"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

var sampleNow = time.Date(2024, 01, 19, 0, 0, 0, 0, time.Local)

func addDummyMeters(root *node.Node) {
	var models []*meter.Model
	meterNames := []string{"elecmeter1", "elecmeter2", "watermeter1", "watermeter2"}
	for _, meterName := range meterNames {
		m := meter.NewModel()
		m.RecordReading(123.45)
		models = append(models, m)
		client := node.WithClients(meterpb.WrapApi(meter.NewModelServer(m)))
		root.Announce(meterName, node.HasTrait(meter.TraitName, client))
	}
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	root := node.New("test")
	addDummyMeters(root)

	now, _ := time.Parse(time.DateTime, "2023-11-15 11:36:00")
	now = now.Round(time.Second) // get rid of millis, etc

	now = sampleNow

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
