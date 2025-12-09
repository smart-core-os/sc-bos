package devices

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	timepb "github.com/smart-core-os/sc-api/go/types/time"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

// getTraitInfo returns the data we support exporting via the DevicesApi download endpoint.
func (s *Server) getTraitInfo() map[string]traitInfo {
	// Note: I want this to be dynamic, however, there are a few issues preventing this:
	// 1. The protoregistry.Files and Types does not include gateway discovered types so we can't use that
	// 2. Not all data for traits is useful, so we need to filter out the useful data
	// 3. Property names aren't that useful to users so some transformation is needed
	// 4. Not all traits include the relevant info in their Api aspect. Check the meter one for where the unit comes from.

	return map[string]traitInfo{
		string(accesspb.TraitName): {
			headers: []string{"access.grant", "access.reason", "access.actor.name", "access.actor.title", "access.actor.displayname", "access.actor.email"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := gen.NewAccessApiClient(s.m.ClientConn())
				data, err := c.GetLastAccessAttempt(ctx, &gen.GetLastAccessAttemptRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return accessAttemptToRow(data), nil
			},
		},
		string(meter.TraitName): {
			headers: []string{"meter.usage", "meter.unit"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := gen.NewMeterApiClient(s.m.ClientConn())
				data, err := c.GetMeterReading(ctx, &gen.GetMeterReadingRequest{Name: name})
				if err != nil {
					return nil, err
				}

				var unit string
				ci := gen.NewMeterInfoClient(s.m.ClientConn())
				if info, err := ci.DescribeMeterReading(ctx, &gen.DescribeMeterReadingRequest{Name: name}); err == nil {
					unit = info.GetUsageUnit()
				}
				return meterReadingToRow(data, unit), nil
			},
			history: func(name string, period *timepb.Period, pageSize int32) *historyCursor {
				c := gen.NewMeterHistoryClient(s.m.ClientConn())
				ci := gen.NewMeterInfoClient(s.m.ClientConn())
				var unit string
				return &historyCursor{
					getPage: func(ctx context.Context, token string) ([]historyRecord, string, error) {
						if token == "" {
							// fetch info the first time
							if info, err := ci.DescribeMeterReading(ctx, &gen.DescribeMeterReadingRequest{Name: name}); err == nil {
								unit = info.GetUsageUnit()
							}
						}

						page, err := c.ListMeterReadingHistory(ctx, &gen.ListMeterReadingHistoryRequest{
							Name:      name,
							PageToken: token,
							PageSize:  pageSize,
							Period:    period,
						})
						if err != nil {
							return nil, "", err
						}

						records := make([]historyRecord, 0, len(page.MeterReadingRecords))
						for _, record := range page.MeterReadingRecords {
							records = append(records, historyRecord{
								at:   record.GetRecordTime().AsTime(),
								vals: meterReadingToRow(record.GetMeterReading(), unit),
							})
						}
						return records, page.NextPageToken, nil
					},
				}
			},
		},
		string(statuspb.TraitName): {
			headers: []string{"status.level", "status.description", "status.recordtime"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := gen.NewStatusApiClient(s.m.ClientConn())
				data, err := c.GetCurrentStatus(ctx, &gen.GetCurrentStatusRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return statusLogToRow(data), nil
			},
		},
		string(trait.AirQualitySensor): {
			headers: []string{"iaq.co2", "iaq.voc", "iaq.pressure", "iaq.comfort", "iaq.infectionrisk", "iaq.score", "iaq.pm1", "iaq.pm25", "iaq.pm10", "iaq.airchange"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewAirQualitySensorApiClient(s.m.ClientConn())
				data, err := c.GetAirQuality(ctx, &traits.GetAirQualityRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return airQualityToRow(data), nil
			},
			history: func(name string, period *timepb.Period, pageSize int32) *historyCursor {
				c := gen.NewAirQualitySensorHistoryClient(s.m.ClientConn())
				return &historyCursor{
					getPage: func(ctx context.Context, token string) ([]historyRecord, string, error) {
						page, err := c.ListAirQualityHistory(ctx, &gen.ListAirQualityHistoryRequest{
							Name:      name,
							PageToken: token,
							PageSize:  pageSize,
							Period:    period,
						})
						if err != nil {
							return nil, "", err
						}

						records := make([]historyRecord, 0, len(page.AirQualityRecords))
						for _, record := range page.AirQualityRecords {
							records = append(records, historyRecord{
								at:   record.GetRecordTime().AsTime(),
								vals: airQualityToRow(record.GetAirQuality()),
							})
						}
						return records, page.NextPageToken, nil
					},
				}
			},
		},
		string(trait.AirTemperature): {
			headers: []string{"airtemperature.temperature", "airtemperature.humidity", "airtemperature.setpoint", "airtemperature.mode"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewAirTemperatureApiClient(s.m.ClientConn())
				data, err := c.GetAirTemperature(ctx, &traits.GetAirTemperatureRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return airTemperatureToRow(data), nil
			},
			history: func(name string, period *timepb.Period, pageSize int32) *historyCursor {
				c := gen.NewAirTemperatureHistoryClient(s.m.ClientConn())
				return &historyCursor{
					getPage: func(ctx context.Context, token string) ([]historyRecord, string, error) {
						page, err := c.ListAirTemperatureHistory(ctx, &gen.ListAirTemperatureHistoryRequest{
							Name:      name,
							PageToken: token,
							PageSize:  pageSize,
							Period:    period,
						})
						if err != nil {
							return nil, "", err
						}

						records := make([]historyRecord, 0, len(page.AirTemperatureRecords))
						for _, record := range page.AirTemperatureRecords {
							records = append(records, historyRecord{
								at:   record.GetRecordTime().AsTime(),
								vals: airTemperatureToRow(record.GetAirTemperature()),
							})
						}
						return records, page.NextPageToken, nil
					},
				}
			},
		},
		string(trait.Electric): {
			headers: []string{"electric.current", "electric.voltage", "electric.powerfactor", "electric.realpower", "electric.apparentpower", "electric.reactivepower"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewElectricApiClient(s.m.ClientConn())
				data, err := c.GetDemand(ctx, &traits.GetDemandRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return electricDemandToRow(data), nil
			},
			history: func(name string, period *timepb.Period, pageSize int32) *historyCursor {
				c := gen.NewElectricHistoryClient(s.m.ClientConn())
				return &historyCursor{
					getPage: func(ctx context.Context, token string) ([]historyRecord, string, error) {
						page, err := c.ListElectricDemandHistory(ctx, &gen.ListElectricDemandHistoryRequest{
							Name:      name,
							PageToken: token,
							PageSize:  pageSize,
							Period:    period,
						})
						if err != nil {
							return nil, "", err
						}

						records := make([]historyRecord, 0, len(page.ElectricDemandRecords))
						for _, record := range page.ElectricDemandRecords {
							records = append(records, historyRecord{
								at:   record.GetRecordTime().AsTime(),
								vals: electricDemandToRow(record.GetElectricDemand()),
							})
						}
						return records, page.NextPageToken, nil
					},
				}
			},
		},
		string(trait.EnterLeaveSensor): {
			headers: []string{"enterleave.entertotal", "enterleave.leavetotal"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewEnterLeaveSensorApiClient(s.m.ClientConn())
				data, err := c.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return enterLeaveEventToRow(data), nil
			},
		},
		string(trait.FanSpeed): {
			headers: []string{"fanspeed.percentage", "fanspeed.preset"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewFanSpeedApiClient(s.m.ClientConn())
				data, err := c.GetFanSpeed(ctx, &traits.GetFanSpeedRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return fanSpeedToRow(data), nil
			},
		},
		string(trait.Light): {
			headers: []string{"light.level", "light.preset"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewLightApiClient(s.m.ClientConn())
				data, err := c.GetBrightness(ctx, &traits.GetBrightnessRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return brightnessToRow(data), nil
			},
		},
		string(trait.OccupancySensor): {
			headers: []string{"occupancy.state", "occupancy.peoplecount", "occupancy.changetime"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewOccupancySensorApiClient(s.m.ClientConn())
				data, err := c.GetOccupancy(ctx, &traits.GetOccupancyRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return occupancyToRow(data), nil
			},
			history: func(name string, period *timepb.Period, pageSize int32) *historyCursor {
				c := gen.NewOccupancySensorHistoryClient(s.m.ClientConn())
				return &historyCursor{
					getPage: func(ctx context.Context, token string) ([]historyRecord, string, error) {
						page, err := c.ListOccupancyHistory(ctx, &gen.ListOccupancyHistoryRequest{
							Name:      name,
							PageToken: token,
							PageSize:  pageSize,
							Period:    period,
						})
						if err != nil {
							return nil, "", err
						}

						records := make([]historyRecord, 0, len(page.OccupancyRecords))
						for _, record := range page.OccupancyRecords {
							records = append(records, historyRecord{
								at:   record.GetRecordTime().AsTime(),
								vals: occupancyToRow(record.GetOccupancy()),
							})
						}
						return records, page.NextPageToken, nil
					},
				}
			},
		},
		string(trait.OpenClose): {
			headers: []string{"openclose.openpercent"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := traits.NewOpenCloseApiClient(s.m.ClientConn())
				data, err := c.GetPositions(ctx, &traits.GetOpenClosePositionsRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return openClosePositionsToRow(data), nil
			},
		},
		string(soundsensorpb.TraitName): {
			headers: []string{"sound.soundpressurelevel"},
			get: func(ctx context.Context, name string) (map[string]string, error) {
				c := gen.NewSoundSensorApiClient(s.m.ClientConn())
				data, err := c.GetSoundLevel(ctx, &gen.GetSoundLevelRequest{Name: name})
				if err != nil {
					return nil, err
				}
				return soundLevelToRow(data), nil
			},
			history: func(name string, period *timepb.Period, pageSize int32) *historyCursor {
				c := gen.NewSoundSensorHistoryClient(s.m.ClientConn())
				return &historyCursor{
					getPage: func(ctx context.Context, token string) ([]historyRecord, string, error) {
						page, err := c.ListSoundLevelHistory(ctx, &gen.ListSoundLevelHistoryRequest{
							Name:      name,
							PageToken: token,
							PageSize:  pageSize,
							Period:    period,
						})
						if err != nil {
							return nil, "", err
						}

						records := make([]historyRecord, 0, len(page.SoundLevelRecords))
						for _, record := range page.SoundLevelRecords {
							records = append(records, historyRecord{
								at:   record.GetRecordTime().AsTime(),
								vals: soundLevelToRow(record.GetSoundLevel()),
							})
						}
						return records, page.NextPageToken, nil
					},
				}
			},
		},
	}
}

type historyCursor struct {
	getPage func(ctx context.Context, token string) ([]historyRecord, string, error)
	gotPage bool            // have we called getPage yet?
	page    []historyRecord // page[0] is the next record to return
	err     error
	token   string
}

// Head returns the next record in the history, or io.EOF if there are no more records.
// Head may request the data from the server if more data is available.
// Call use() on the returned record to advance the cursor.
func (c *historyCursor) Head(ctx context.Context) (historyRecord, error) {
	// get if we've never got before, or we need another page (and there is one)
	if !c.gotPage || len(c.page) == 0 && c.token != "" {
		c.gotPage = true
		c.page, c.token, c.err = c.getPage(ctx, c.token)
	}
	if c.err != nil {
		return historyRecord{}, c.err
	}
	if len(c.page) == 0 {
		return historyRecord{}, io.EOF
	}
	record := c.page[0]
	record.use = func() {
		c.page = c.page[1:]
	}
	return record, nil
}

type historyRecord struct {
	at   time.Time
	vals map[string]string
	use  func()
}

func accessAttemptToRow(d *gen.AccessAttempt) map[string]string {
	vals := make(map[string]string)
	vals["access.grant"] = d.GetGrant().String()
	vals["access.reason"] = d.GetReason()
	if actor := d.GetActor(); actor != nil {
		vals["access.actor.name"] = actor.Name
		vals["access.actor.title"] = actor.Title
		vals["access.actor.displayname"] = actor.DisplayName
		vals["access.actor.email"] = actor.Email
	}
	return vals
}

func meterReadingToRow(d *gen.MeterReading, unit string) map[string]string {
	return map[string]string{
		"meter.usage": fmt.Sprintf("%.3f", d.Usage),
		"meter.unit":  unit,
	}
}

func statusLogToRow(d *gen.StatusLog) map[string]string {
	vals := make(map[string]string)
	vals["status.level"] = d.GetLevel().String()
	vals["status.description"] = d.GetDescription()
	if d.RecordTime != nil {
		vals["status.recordtime"] = d.GetRecordTime().AsTime().String()
	}
	return vals
}

func airQualityToRow(d *traits.AirQuality) map[string]string {
	vals := make(map[string]string)
	if d.CarbonDioxideLevel != nil {
		vals["iaq.co2"] = fmt.Sprintf("%.3f", *d.CarbonDioxideLevel)
	}
	if d.VolatileOrganicCompounds != nil {
		vals["iaq.voc"] = fmt.Sprintf("%.3f", *d.VolatileOrganicCompounds)
	}
	if d.AirPressure != nil {
		vals["iaq.pressure"] = fmt.Sprintf("%.3f", *d.AirPressure)
	}
	if d.Comfort != traits.AirQuality_COMFORT_UNSPECIFIED {
		vals["iaq.comfort"] = d.Comfort.String()
	}
	if d.InfectionRisk != nil {
		vals["iaq.infectionrisk"] = fmt.Sprintf("%.3f", *d.InfectionRisk)
	}
	if d.Score != nil {
		vals["iaq.score"] = fmt.Sprintf("%.3f", *d.Score)
	}
	if d.ParticulateMatter_1 != nil {
		vals["iaq.pm1"] = fmt.Sprintf("%.3f", *d.ParticulateMatter_1)
	}
	if d.ParticulateMatter_25 != nil {
		vals["iaq.pm25"] = fmt.Sprintf("%.3f", *d.ParticulateMatter_25)
	}
	if d.ParticulateMatter_10 != nil {
		vals["iaq.pm10"] = fmt.Sprintf("%.3f", *d.ParticulateMatter_10)
	}
	if d.AirChangePerHour != nil {
		vals["iaq.airchange"] = fmt.Sprintf("%.3f", *d.AirChangePerHour)
	}
	return vals
}

func airTemperatureToRow(d *traits.AirTemperature) map[string]string {
	vals := make(map[string]string)
	if d.AmbientTemperature != nil {
		vals["airtemperature.temperature"] = fmt.Sprintf("%.1f", d.AmbientTemperature.ValueCelsius)
	}
	if d.AmbientHumidity != nil {
		vals["airtemperature.humidity"] = fmt.Sprintf("%.1f", *d.AmbientHumidity)
	}
	if d.GetTemperatureSetPoint() != nil {
		vals["airtemperature.setpoint"] = fmt.Sprintf("%.1f", d.GetTemperatureSetPoint().ValueCelsius)
	}
	if d.Mode != traits.AirTemperature_MODE_UNSPECIFIED {
		vals["airtemperature.mode"] = d.Mode.String()
	}
	return vals
}

func electricDemandToRow(d *traits.ElectricDemand) map[string]string {
	vals := make(map[string]string)
	if d.Current != 0 {
		vals["electric.current"] = fmt.Sprintf("%.3f", d.Current)
	}
	if d.Voltage != nil {
		vals["electric.voltage"] = fmt.Sprintf("%.3f", *d.Voltage)
	}
	if d.PowerFactor != nil {
		vals["electric.powerfactor"] = fmt.Sprintf("%.3f", *d.PowerFactor)
	}
	if d.RealPower != nil {
		vals["electric.realpower"] = fmt.Sprintf("%.3f", *d.RealPower)
	}
	if d.ApparentPower != nil {
		vals["electric.apparentpower"] = fmt.Sprintf("%.3f", *d.ApparentPower)
	}
	if d.ReactivePower != nil {
		vals["electric.reactivepower"] = fmt.Sprintf("%.3f", *d.ReactivePower)
	}
	return vals
}

func enterLeaveEventToRow(d *traits.EnterLeaveEvent) map[string]string {
	vals := make(map[string]string)
	vals["enterleave.entertotal"] = fmt.Sprintf("%d", d.EnterTotal)
	vals["enterleave.leavetotal"] = fmt.Sprintf("%d", d.LeaveTotal)
	return vals
}

func fanSpeedToRow(d *traits.FanSpeed) map[string]string {
	vals := make(map[string]string)
	vals["fanspeed.percentage"] = fmt.Sprintf("%.1f", d.Percentage)
	vals["fanspeed.preset"] = d.Preset
	return vals
}

func brightnessToRow(d *traits.Brightness) map[string]string {
	vals := make(map[string]string)
	vals["light.level"] = fmt.Sprintf("%.1f", d.LevelPercent)
	if d.Preset != nil {
		vals["light.preset"] = d.Preset.Name
	}
	return vals
}

func occupancyToRow(d *traits.Occupancy) map[string]string {
	vals := make(map[string]string)
	vals["occupancy.state"] = d.State.String()
	vals["occupancy.peoplecount"] = fmt.Sprintf("%d", d.PeopleCount)
	if d.StateChangeTime != nil {
		vals["occupancy.changetime"] = d.StateChangeTime.AsTime().String()
	}
	return vals
}

func openClosePositionsToRow(d *traits.OpenClosePositions) map[string]string {
	vals := make(map[string]string)
	if len(d.States) != 1 {
		return vals
	}
	pos := d.States[0]
	vals["openclose.openpercent"] = fmt.Sprintf("%.1f", pos.OpenPercent)
	return vals
}

func soundLevelToRow(d *gen.SoundLevel) map[string]string {
	return map[string]string{
		"sound.soundpressurelevel": fmt.Sprintf("%.1f", d.GetSoundPressureLevel()),
	}
}
