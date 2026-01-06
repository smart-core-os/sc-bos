package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/smart-core-os/sc-bos/pkg/app/appconf"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/driver/alldrivers"
	mockcfg "github.com/smart-core-os/sc-bos/pkg/driver/mock/config"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	airqualitycfg "github.com/smart-core-os/sc-bos/pkg/zone/feature/airquality/config"
	occupancycfg "github.com/smart-core-os/sc-bos/pkg/zone/feature/occupancy/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"

	"github.com/smart-core-os/sc-bos/pkg/zone/allzones"
	meterscfg "github.com/smart-core-os/sc-bos/pkg/zone/feature/meter/config"
)

var (
	lookBack time.Duration
	dbUrl    string
	app      string
)

func init() {
	flag.DurationVar(&lookBack, "look-back", time.Hour*24*30*2, "amount of time to populate database history for starting from now, going backwards")
	flag.StringVar(&dbUrl, "db-url", "postgres://postgres:postgres@localhost:5432/smart_core", "database url")
	flag.StringVar(&app, "appconf", "app.conf.json", "app configuration file")
}

func main() {
	flag.Parse()

	appConf, err := appconf.LoadLocalConfig(path.Dir(app), path.Base(app))
	if err != nil {
		panic(err)
	}

	var sd seedDevices
	err = parseZoneConfig(&sd, appConf)
	if err != nil {
		panic(err)
	}

	err = parseDeviceConfig(&sd, appConf)
	if err != nil {
		panic(err)
	}
	sd.normalise()

	ctx := context.Background()

	conf, err := pgxpool.ParseConfig(dbUrl)

	if err != nil {
		panic(err)
	}

	conf.MaxConns = 4
	conf.MinConns = 1

	db, err := pgxpool.ConnectConfig(ctx, conf)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// before running the seeding tasks in parallel, make sure the DB exists to avoid error on first run:
	// CREATE TABLE IF NOT EXISTS duplicate key value violates unique constraint
	_, err = pgxstore.SetupStoreFromPool(ctx, "dummy", db)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, d := range sd.airQuality {
			err = SeedAirTemperature(ctx, db, d, lookBack)
			if err != nil {
				panic(err)
			}
			fmt.Printf("seeded air temperature device %s\n", d)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, d := range sd.electric {
			err = SeedMeter(ctx, db, d, lookBack)
			if err != nil {
				panic(err)
			}
			fmt.Printf("seeded meter device %s\n", d)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, d := range sd.airTemperature {
			err = SeedAirTemperature(ctx, db, d, lookBack)
			if err != nil {
				panic(err)
			}
			fmt.Printf("seeded air temperature device %s\n", d)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, d := range sd.soundSensor {
			err = SeedSoundSensor(ctx, db, d, lookBack)
			if err != nil {
				panic(err)
			}
			fmt.Printf("seeded sound sensor device %s\n", d)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, d := range sd.occupancy {
			err = SeedOccupancy(ctx, db, d, lookBack)
			if err != nil {
				panic(err)
			}
			fmt.Printf("seeded occupancy device %s\n", d)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, d := range sd.allocation {
			err = SeedAllocation(ctx, db, d, lookBack)
			if err != nil {
				panic(err)
			}
			fmt.Printf("seeded allocation device %s\n", d)
		}
	}()

	wg.Wait()
}

func loadSystemConfig() (sysconf.Config, error) {
	systemConfig := sysconf.Default()

	systemConfig.ZoneFactories = allzones.Factories()
	systemConfig.DriverFactories = alldrivers.Factories()

	err := sysconf.Load(&systemConfig)
	return systemConfig, err
}

func parseZoneConfig(sd *seedDevices, appConf *appconf.Config) error {
	for _, conf := range appConf.Zones {
		if conf.Type != "area" {
			continue
		}
		aq := airqualitycfg.Root{}
		occ := occupancycfg.Root{}
		mtr := meterscfg.Root{}

		buf, err := conf.MarshalJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(buf, &aq)
		if err != nil {
			return err
		}

		if len(aq.AirQualitySensors) > 0 {
			sd.airQuality = append(sd.airQuality, conf.Name)
		}

		err = json.Unmarshal(buf, &occ)
		if err != nil {
			return err
		}

		if len(occ.OccupancySensors) > 0 || len(occ.EnterLeaveOccupancySensors) > 0 {
			sd.occupancy = append(sd.occupancy, conf.Name)
		}

		err = json.Unmarshal(buf, &mtr)
		if err != nil {
			return err
		}

		if len(mtr.Meters) > 0 {
			sd.electric = append(sd.electric, conf.Name)
		}
		for _, group := range mtr.MeterGroups {
			for _, met := range group {
				sd.electric = append(sd.electric, met)
			}
		}

	}

	return nil
}

type seedDevices struct {
	airQuality     []string
	electric       []string
	airTemperature []string
	soundSensor    []string
	occupancy      []string
	allocation     []string
}

func (sd *seedDevices) normalise() {
	slices.Sort(sd.airQuality)
	slices.Sort(sd.electric)
	slices.Sort(sd.airTemperature)
	slices.Sort(sd.soundSensor)
	slices.Sort(sd.occupancy)
	slices.Compact(sd.airQuality)
	slices.Compact(sd.electric)
	slices.Compact(sd.airTemperature)
	slices.Compact(sd.soundSensor)
	slices.Compact(sd.occupancy)
}

func parseDeviceConfig(sd *seedDevices, appConf *appconf.Config) error {
	for _, dr := range appConf.Drivers {
		buf, err := dr.MarshalJSON()
		if err != nil {
			return err
		}

		var devices mockcfg.Root
		err = json.Unmarshal(buf, &devices)
		if err != nil {
			return err
		}

		for _, device := range devices.Devices {
			for _, trt := range device.Traits {
				switch trait.Name(trt.Name) {
				case trait.AirQualitySensor:
					sd.airQuality = append(sd.airQuality, device.Name)
				case trait.Electric:
					sd.electric = append(sd.electric, device.Name)
				case trait.AirTemperature:
					sd.airTemperature = append(sd.airTemperature, device.Name)
				case soundsensorpb.TraitName:
					sd.soundSensor = append(sd.soundSensor, device.Name)
				case allocationpb.TraitName:
					sd.allocation = append(sd.allocation, device.Name)
				}
			}
		}
	}

	return nil
}
