package main

import (
	"time"

	"github.com/timshannon/bolthold"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp"
	"github.com/smart-core-os/sc-bos/pkg/node"
)

func main() {
	logger, err := zap.NewDevelopment()

	if err != nil {
		panic(err)
	}

	announcer := node.New("exporthttp-test")

	err = announceOccupancy(announcer, "pir/01", 8)
	if err != nil {
		panic(err)
	}
	err = announceOccupancy(announcer, "pir/02", 2)
	if err != nil {
		panic(err)
	}
	err = announceOccupancy(announcer, "pir/03", 4)
	if err != nil {
		panic(err)
	}
	err = announceAirQuality(announcer, "smart-core/iaq/01", 112.1)
	if err != nil {
		panic(err)
	}

	err = announceTemperature(announcer, "FCU/01", 19.1)
	if err != nil {
		panic(err)
	}
	err = announceTemperature(announcer, "FCU/02", 21.3)
	if err != nil {
		panic(err)
	}

	err = announceMeter(announcer, "smart-core/meters/01", "mWh", time.Second, []float32{0, 1, 2, 12, 54, 100, 222, 654, 900, 1122, 1543})

	if err != nil {
		panic(err)
	}

	err = announceMeter(announcer, "smart-core/meters/03", "litres", time.Second, []float32{0, 1, 11, 111, 222, 433, 566, 888, 1002, 1023, 2000})

	if err != nil {
		panic(err)
	}

	// run this script twice to test that the previous execution times are being read correctly
	db, err := bolthold.Open("exporthttp-test.db", 0666, nil)
	if err != nil {
		panic(err)
	}

	srv := auto.Services{
		Logger:   logger,
		Database: db,
		Node:     announcer,
		Now:      func() time.Time { return time.Now() },
	}

	lifecycle := exporthttp.Factory.New(srv)
	_, err = lifecycle.Configure([]byte(cfg))

	if err != nil {
		panic(err)
	}

	_, err = lifecycle.Start()
	if err != nil {
		panic(err)
	}

	defer func() {
		_, err = lifecycle.Stop()
		if err != nil {
			panic(err)
		}
	}()

	// wait for all automations in exporthttp to finish
	time.Sleep(3 * time.Minute)
}

const (
	cfg = `
{
  "name":     "exporthttp-test",
  "type":     "exporthttp",
  "disabled": false,
  "baseUrl": "",
  "site":     "abc-test1",
  "authentication": {
	"type":       "Bearer",
	"secretFile": ""
  },
  "sources": {
	"occupancy":    {
	  "path": "occupancy",
	  "schedule": "0/1 * * * *",
	  "timeout": "15s",
	  "sensors":  ["pir/01","pir/02","pir/03"]
	},
	"temperature": {
	  "path": "temperature",
	  "schedule": "0/2 * * * *",
	  "timeout": "15s",
	  "sensors": ["FCU/01","FCU/02"]
	},
	"energy":       {
	  "path": "energy",
	  "schedule": "0/1 * * * *",
	  "timeout": "15s",
	  "meters": ["smart-core/meters/01"]
	},
	"airQuality":  {
	  "path": "air_quality",
	  "schedule": "0/1 * * * *",
	  "timeout": "15s",
	  "sensors": ["smart-core/iaq/01"]
	},
	"water": {
	  "path": "water",
	  "schedule": "0/1 * * * *",
	  "timeout": "15s",
	  "meters" : ["smart-core/meters/03"]
	}
  }
}`
)
