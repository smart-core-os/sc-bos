package auto

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/energystoragepb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

type EnergyStorageDeviceType string

const (
	EnergyStorageDeviceTypeBattery EnergyStorageDeviceType = "battery"
	EnergyStorageDeviceTypeEV      EnergyStorageDeviceType = "ev"
	EnergyStorageDeviceTypeDrone   EnergyStorageDeviceType = "drone"
)

type energyStorageProfile struct {
	MaxCapacityKwh  float32
	MaxRangeKm      float32
	Voltage         *float32Range // nil means no voltage reporting
	ChargeRate      float32Range
	DischargeRate   float32Range
	JourneyLength   *float32Range // typical journey distance for mobile devices
	UpdateInterval  durationRange
	PluggedInChance float64
}

// IsMobile returns true if the device can move (has range > 0)
func (p energyStorageProfile) IsMobile() bool {
	return p.MaxRangeKm > 0
}

var deviceProfiles = map[EnergyStorageDeviceType]energyStorageProfile{
	EnergyStorageDeviceTypeBattery: {
		MaxCapacityKwh:  0.1,                               // 100Wh for small battery
		Voltage:         &float32Range{Min: 3.0, Max: 4.2}, // Low voltage battery - reported
		ChargeRate:      float32Range{Min: 0.5, Max: 2.0},  // Slower charge rates
		DischargeRate:   float32Range{Min: 0.2, Max: 1.0},
		UpdateInterval:  durationRange{Min: 30 * time.Second, Max: 2 * time.Minute},
		PluggedInChance: 0.8, // Usually plugged in
	},
	EnergyStorageDeviceTypeEV: {
		MaxCapacityKwh:  75.0,  // 75kWh for EV
		MaxRangeKm:      450.0, // Standard EV range
		ChargeRate:      float32Range{Min: 1.0, Max: 3.0},
		DischargeRate:   float32Range{Min: 0.3, Max: 1.5},
		JourneyLength:   &float32Range{Min: 25, Max: 150}, // Typical daily commute/errands
		UpdateInterval:  durationRange{Min: 45 * time.Second, Max: 90 * time.Second},
		PluggedInChance: 0.7,
	},
	EnergyStorageDeviceTypeDrone: {
		MaxCapacityKwh:  2.0,                              // 2kWh for drone
		MaxRangeKm:      50.0,                             // Limited flight range
		ChargeRate:      float32Range{Min: 2.0, Max: 5.0}, // Fast charging for quick turnaround
		DischargeRate:   float32Range{Min: 1.0, Max: 3.0}, // Faster discharge during flight
		JourneyLength:   &float32Range{Min: 2, Max: 15},   // Typical drone flight pattern
		UpdateInterval:  durationRange{Min: 15 * time.Second, Max: 45 * time.Second},
		PluggedInChance: 0.5, // Often flying/in use
	},
}

func EnergyStorage(model *energystoragepb.Model, kind EnergyStorageDeviceType) service.Lifecycle {
	profile, exists := deviceProfiles[kind]
	if !exists {
		profile = deviceProfiles[EnergyStorageDeviceTypeEV] // Default fallback
	}

	type state int
	const (
		stateIdle state = iota
		stateCharging
		stateDischarging
	)
	states := []state{stateIdle, stateCharging, stateDischarging}

	s := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			// Initialize with a random starting percentage
			currentPercentage := float32Between(20, 80)
			var currentState state = stateIdle
			var stateStartTime *timestamppb.Timestamp

			for {
				// Randomly change state occasionally
				if randomBool(0.3) { // 30% chance to change state
					newState := oneOf(states...)
					if newState != currentState {
						currentState = newState
						stateStartTime = timestamppb.Now()
					}
				}

				// Calculate percentage change rates based on device profile
				var chargeRate, dischargeRate float32

				// Update percentage based on current state
				switch currentState {
				case stateIdle:
					// Slight random drift when idle
					currentPercentage += float32Between(-0.5, 0.5)
				case stateCharging:
					chargeRate = profile.ChargeRate.Random()
					currentPercentage += chargeRate
				case stateDischarging:
					dischargeRate = profile.DischargeRate.Random()
					currentPercentage -= dischargeRate
				}

				// Clamp percentage to realistic bounds
				if currentPercentage < 5 {
					currentPercentage = 5
					currentState = stateCharging // Auto-charge when low
					stateStartTime = timestamppb.Now()
				} else if currentPercentage > 95 {
					currentPercentage = 95
					currentState = stateIdle // Stop charging when full
				}

				// Build the energy level state with device-specific fields
				energyLevel := &energystoragepb.EnergyLevel{
					Quantity: &energystoragepb.EnergyLevel_Quantity{
						Percentage:  currentPercentage,
						EnergyKwh:   currentPercentage * profile.MaxCapacityKwh / 100,
						Descriptive: getDescriptiveThreshold(currentPercentage),
						Voltage:     getVoltageFromPercentage(currentPercentage, profile),
					},
					PluggedIn: currentState == stateCharging || randomBool(profile.PluggedInChance),
				}

				// Only set voltage if profile includes it
				if profile.Voltage != nil {
					energyLevel.Quantity.Voltage = getVoltageFromPercentage(currentPercentage, profile)
				}

				// Only set distance for mobile devices
				if profile.IsMobile() {
					energyLevel.Quantity.DistanceKm = currentPercentage * profile.MaxRangeKm / 100
				}

				// Set flow state with device-specific parameters
				switch currentState {
				case stateIdle:
					energyLevel.Flow = &energystoragepb.EnergyLevel_Idle{
						Idle: &energystoragepb.EnergyLevel_Steady{
							StartTime: stateStartTime,
						},
					}
				case stateCharging:
					target := &energystoragepb.EnergyLevel_Quantity{
						Percentage: float32Between(85, 100),
					}
					transfer := &energystoragepb.EnergyLevel_Transfer{
						StartTime: stateStartTime,
						Speed:     getSpeedFromRate(chargeRate),
						Target:    target,
					}
					// Only set target distance for mobile devices
					if profile.IsMobile() {
						target.DistanceKm = target.Percentage * profile.MaxRangeKm / 100
					}
					energyLevel.Flow = &energystoragepb.EnergyLevel_Charge{
						Charge: transfer,
					}
				case stateDischarging:
					target := &energystoragepb.EnergyLevel_Quantity{
						Percentage: float32Between(5, 25),
					}
					transfer := &energystoragepb.EnergyLevel_Transfer{
						StartTime: stateStartTime,
						Speed:     getSpeedFromRate(dischargeRate),
						Target:    target,
					}

					// Add distance info only for mobile devices
					if profile.IsMobile() {
						if profile.JourneyLength != nil {
							transfer.DistanceKm = profile.JourneyLength.Random()
						}
						target.DistanceKm = target.Percentage * profile.MaxRangeKm / 100
					}

					energyLevel.Flow = &energystoragepb.EnergyLevel_Discharge{
						Discharge: transfer,
					}
				}

				_, _ = model.UpdateEnergyLevel(energyLevel)

				select {
				case <-ctx.Done():
					return
				case <-time.After(profile.UpdateInterval.Random()):
				}
			}
		}()
		return nil
	}))
	_, _ = s.Configure([]byte(`""`))
	return s
}

func getDescriptiveThreshold(percentage float32) energystoragepb.EnergyLevel_Quantity_Threshold {
	switch {
	case percentage < 10:
		return energystoragepb.EnergyLevel_Quantity_CRITICALLY_LOW
	case percentage < 20:
		return energystoragepb.EnergyLevel_Quantity_LOW
	case percentage < 40:
		return energystoragepb.EnergyLevel_Quantity_MEDIUM
	case percentage < 80:
		return energystoragepb.EnergyLevel_Quantity_HIGH
	case percentage >= 95:
		return energystoragepb.EnergyLevel_Quantity_FULL
	default:
		return energystoragepb.EnergyLevel_Quantity_HIGH
	}
}

func getSpeedFromRate(rate float32) energystoragepb.EnergyLevel_Transfer_Speed {
	switch {
	case rate < 0.5:
		return energystoragepb.EnergyLevel_Transfer_EXTRA_SLOW
	case rate < 1.0:
		return energystoragepb.EnergyLevel_Transfer_SLOW
	case rate < 2.0:
		return energystoragepb.EnergyLevel_Transfer_NORMAL
	case rate < 2.5:
		return energystoragepb.EnergyLevel_Transfer_FAST
	default:
		return energystoragepb.EnergyLevel_Transfer_EXTRA_FAST
	}
}

func getVoltageFromPercentage(percentage float32, profile energyStorageProfile) *float32 {
	if profile.Voltage == nil {
		return nil
	}
	// Simulate realistic voltage curve based on device profile
	normalized := percentage / 100.0
	voltageRange := profile.Voltage.Max - profile.Voltage.Min
	baseVoltage := profile.Voltage.Min + (voltageRange * normalized)

	// Add realistic variation based on voltage range
	variationPercent := float32(0.02) // 2% variation
	if voltageRange < 10 {            // For low voltage devices like batteries
		variationPercent = 0.05 // 5% variation
	}
	variation := voltageRange * variationPercent * float32Between(-1.0, 1.0)
	ret := baseVoltage + variation
	return &ret
}
