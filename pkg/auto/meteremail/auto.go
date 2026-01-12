// Package meteremail provides an automation that collects the instantaneous meter readings for a set of given devices.
// The automation uses the Meter API to fetch meter readings on a configurable fixed date,
// formats a summary email using html/template and sends it to some recipients using smtp with a CSV file for detailed readings.
// Test program for meter reading automation is in 'cmd/tools/test-meteremail/main.go'.
package meteremail

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/meteremail/config"
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

const AutoName = "meteremail"

var Factory auto.Factory = factory{}

type factory struct{}

type autoImpl struct {
	*service.Service[config.Root]
	auto.Services
}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &autoImpl{Services: services}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(config.ReadBytes))
	a.Logger = a.Logger.Named(AutoName)
	return a
}

func (_ factory) ConfigBlocks() []block.Block {
	return config.Blocks
}

// getMeterReadingAndSource gets the meter reading for the given meter and also the location metadata for the meter
func (a *autoImpl) getMeterReadingAndSource(ctx context.Context, meterName string, meterType MeterType,
	meterClient meterpb.MeterApiClient, metadataClient traits.MetadataApiClient, timing *config.Timing) (*config.Source, *MeterReading, error) {

	meterReq := &meterpb.GetMeterReadingRequest{
		Name: meterName,
	}

	meterRes, err := retryT(ctx, timing, func(ctx context.Context) (*meterpb.MeterReading, error) {
		withTimeoutCtx, cancel := context.WithTimeout(ctx, timing.Timeout.Duration)
		defer cancel()
		return meterClient.GetMeterReading(withTimeoutCtx, meterReq)
	})
	if err != nil {
		a.Logger.Warn("failed to fetch meter readings for meter ", zap.String("meter", meterName), zap.Error(err))
		return nil, nil, err
	}

	meterReading := &MeterReading{MeterType: meterType, Date: time.Now(), Reading: meterRes.Usage}
	source := &config.Source{Name: meterName}

	metadataReq := &traits.GetMetadataRequest{
		Name: meterName,
	}

	withTimeoutCtx, cancel := context.WithTimeout(ctx, timing.Timeout.Duration)
	defer cancel()
	metadataRes, err := metadataClient.GetMetadata(withTimeoutCtx, metadataReq)
	if err != nil {
		// not a major problem if we can't get metadata as we still have the name to ID the meter
		a.Logger.Warn("failed to fetch meta data for meter", zap.String("meter", meterName), zap.Error(err))
	}
	if metadataRes.Location != nil {
		source.Floor = metadataRes.Location.Floor
		source.Zone = metadataRes.Location.Zone
	}
	return source, meterReading, nil
}

// generateSummaryReports calculates the total energy per zone and appends the totals to attrs.EnergySummaryReports & attrs.WaterSummaryReports
func generateSummaryReports(attrs *Attrs) {

	floorKeys := attrs.getFloorKeys()
	for _, floorName := range floorKeys {
		zones := attrs.ReadingsByFloorZone[floorName]
		zoneKeys := make([]string, 0, len(zones))
		for k := range zones {
			zoneKeys = append(zoneKeys, k)
		}
		sort.Strings(zoneKeys)

		for _, zoneName := range zoneKeys {
			zoneTotalEnergy := float32(0.0)
			zoneTotalWater := float32(0.0)
			meters := zones[zoneName]
			for _, meter := range meters {
				if meter.MeterReading.MeterType == MeterTypeElectric {
					zoneTotalEnergy += meter.MeterReading.Reading
				}
				if meter.MeterReading.MeterType == MeterTypeWater {
					zoneTotalWater += meter.MeterReading.Reading
				}
			}
			attrs.EnergySummaryReports = append(attrs.EnergySummaryReports, SummaryReport{Floor: floorName, Zone: zoneName, TotalReading: zoneTotalEnergy})
			attrs.WaterSummaryReports = append(attrs.WaterSummaryReports, SummaryReport{Floor: floorName, Zone: zoneName, TotalReading: zoneTotalWater})
		}
	}
}

// createMeterReadingsFile creates a CSV file with detailed meter readings, organised by floor then zone.
// Returns the raw bytes of the file
//
//goland:noinspection GoUnhandledErrorResult
func (a *autoImpl) createMeterReadingsFile(attrs *Attrs) []byte {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "Electric Meter Readings\n")

	floorKeys := attrs.getFloorKeys()
	for _, floorName := range floorKeys {
		zones := attrs.ReadingsByFloorZone[floorName]
		zoneKeys := make([]string, 0, len(zones))
		for k := range zones {
			zoneKeys = append(zoneKeys, k)
		}
		sort.Strings(zoneKeys)

		for _, zoneName := range zoneKeys {
			fmt.Fprintf(buf, "%s - %s\n", floorName, zoneName)
			fmt.Fprintf(buf, "Name                          ,Floor	,Zone	,Reading (kWh)	\n")
			meters := zones[zoneName]
			for _, meter := range meters {
				if meter.MeterReading.MeterType == MeterTypeElectric {
					fmt.Fprintf(buf, "%s,%s,%s,%f\n", meter.Source.Name, floorName, zoneName, meter.MeterReading.Reading)
				}
			}
		}
	}

	fmt.Fprintf(buf, "\n\nWater Meter Readings\n")

	for _, floorName := range floorKeys {
		zones := attrs.ReadingsByFloorZone[floorName]
		zoneKeys := make([]string, 0, len(zones))
		for k := range zones {
			zoneKeys = append(zoneKeys, k)
		}
		sort.Strings(zoneKeys)

		for _, zoneName := range zoneKeys {
			fmt.Fprintf(buf, "%s - %s\n", floorName, zoneName)
			fmt.Fprintf(buf, "Name                          ,Floor	,Zone	,Reading (m3)	\n")
			meters := zones[zoneName]
			for _, meter := range meters {
				if meter.MeterReading.MeterType == MeterTypeWater {
					fmt.Fprintf(buf, "%s,%s,%s,%f\n", meter.Source.Name, floorName, zoneName, meter.MeterReading.Reading)
				}
			}
		}
	}
	return buf.Bytes()
}

// groupByFloorAndZone take the data from sources in attrs and group them into a map of floor -> zone -> readings
// for easy summarising & aggregation
func groupByFloorAndZone(attrs *Attrs) {

	if attrs.ReadingsByFloorZone == nil {
		attrs.ReadingsByFloorZone = make(map[string]map[string][]Stats)
	}

	for _, stat := range attrs.Stats {

		if _, ok := attrs.ReadingsByFloorZone[stat.Source.Floor]; !ok {
			attrs.ReadingsByFloorZone[stat.Source.Floor] = make(map[string][]Stats)
		}

		attrs.ReadingsByFloorZone[stat.Source.Floor][stat.Source.Zone] = append(attrs.ReadingsByFloorZone[stat.Source.Floor][stat.Source.Zone], stat)
	}
}

func applyDefaults(timing *config.Timing) {
	if timing.Timeout.Duration == 0 {
		timing.Timeout = jsontypes.Duration{Duration: 10 * time.Second}
	}
	if timing.NumRetries == 0 {
		timing.NumRetries = 3
	}
	if timing.BackoffStart.Duration == 0 {
		timing.BackoffStart = jsontypes.Duration{Duration: 2 * time.Second}
	}
	if timing.BackoffMax.Duration == 0 {
		timing.BackoffMax = jsontypes.Duration{Duration: 30 * time.Second}
	}
	if timing.BackoffMax.Duration < timing.BackoffStart.Duration {
		timing.BackoffMax = timing.BackoffStart
	}
}

func (a *autoImpl) applyConfig(ctx context.Context, cfg config.Root) error {
	logger := a.Logger
	logger = logger.With(zap.String("snmp.addr", cfg.Destination.Addr()))
	applyDefaults(&cfg.Timing)

	meterClient := meterpb.NewMeterApiClient(a.Node.ClientConn())
	metadataClient := traits.NewMetadataApiClient(a.Node.ClientConn())

	sendTime := cfg.Destination.SendTime
	now := cfg.Now
	if now == nil {
		now = a.Now
	}
	if now == nil {
		now = time.Now
	}
	go func() {
		t := now()
		for {
			next := sendTime.Next(t)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(next)):
				// Use the time we were planning on running instead of the current time.
				// We do this to make output more predictable
				t = next
			}

			attrs := Attrs{
				Now:          t,
				Stats:        []Stats{},
				TemplateArgs: cfg.TemplateArgs,
			}

			logger.Debug("Meter email is being generated...", zap.Duration("timeout", cfg.Timing.Timeout.Duration))
			for _, meterName := range cfg.ElectricMeters {
				source, reading, err := a.getMeterReadingAndSource(ctx, meterName, MeterTypeElectric, meterClient, metadataClient, &cfg.Timing)
				if err == nil {
					attrs.Stats = append(attrs.Stats, Stats{Source: *source, MeterReading: *reading})
				} else {
					logger.Error("Error getting info for electric meter ", zap.String("meterName", meterName), zap.Error(err))
				}
			}

			for _, meterName := range cfg.WaterMeters {
				source, reading, err := a.getMeterReadingAndSource(ctx, meterName, MeterTypeWater, meterClient, metadataClient, &cfg.Timing)
				if err == nil {
					attrs.Stats = append(attrs.Stats, Stats{Source: *source, MeterReading: *reading})
				} else {
					logger.Error("Error getting info for water meter ", zap.String("meterName", meterName), zap.Error(err))
				}
			}

			// create map of floors/zones in map attrs.ReadingsByFloorZone
			groupByFloorAndZone(&attrs)

			// generate the detailed meter readings CSV attachment file
			attachmentName := "meter-readings-" + time.Now().Format("2006-01-02") + ".csv"
			file := a.createMeterReadingsFile(&attrs)
			attachmentCfg := config.AttachmentCfg{
				AttachmentName: attachmentName,
				Attachment:     file,
			}

			// generate the readings summary which is displayed in the email body
			generateSummaryReports(&attrs)
			err := retry(ctx, &cfg.Timing, func(ctx context.Context) error {
				return sendEmail(cfg.Destination, attachmentCfg, attrs, logger)
			})
			if err != nil {
				logger.Warn("failed to send email", zap.Error(err))
			} else {
				logger.Info("email sent")
			}
		}
	}()

	return nil
}

func retry(ctx context.Context, timing *config.Timing, f func(context.Context) error) error {
	return task.Run(ctx, func(ctx context.Context) (task.Next, error) {
		return 0, f(ctx)
	}, task.WithBackoff(timing.BackoffStart.Duration, timing.BackoffMax.Duration), task.WithRetry(timing.NumRetries))
}

func retryT[T any](ctx context.Context, timing *config.Timing, f func(context.Context) (T, error)) (T, error) {
	var t T
	err := retry(ctx, timing, func(ctx context.Context) error {
		var err error
		t, err = f(ctx)
		return err
	})
	return t, err
}
