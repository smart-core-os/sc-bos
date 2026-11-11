package merge

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const (
	BacNetCommsError = "BACnet Communication Error"

	SystemName = "BACnet"
)

func IntoTrait(client *comm.Client, devices known.Context, faultCheck *healthpb.FaultCheck, traitConfig config.RawTrait, logger *zap.Logger) (node.SelfAnnouncer, error) {
	fault := newTraitFault(faultCheck, traitConfig.Health.FaultAfterDuration())
	// todo: implement some traits that pull data from different bacnet devices.
	switch traitConfig.Kind {
	case trait.AirQualitySensor:
		return newAirQualitySensor(client, devices, fault, traitConfig, logger)
	case trait.AirTemperature:
		return newAirTemperature(client, devices, fault, traitConfig, logger)
	case trait.Electric:
		return newElectric(client, devices, fault, traitConfig, logger)
	case trait.Emergency:
		return newEmergency(client, devices, fault, traitConfig, logger)
	case trait.EnergyStorage:
		return newEnergyStorage(client, devices, fault, traitConfig, logger)
	case trait.FanSpeed:
		return newFanSpeed(client, devices, fault, traitConfig, logger)
	case trait.Light:
		return newLight(client, devices, fault, traitConfig, logger)
	case meterpb.TraitName:
		return newMeter(client, devices, fault, traitConfig, logger)
	case trait.Mode:
		return newMode(client, devices, fault, traitConfig, logger)
	case trait.OccupancySensor:
		return newOccupancy(client, devices, fault, traitConfig, logger)
	case trait.OnOff:
		return newOnOff(client, devices, fault, traitConfig, logger)
	case accesspb.TraitName:
		return newAccess(client, devices, fault, traitConfig, logger)
	case securityeventpb.TraitName:
		return newSecurityEvent(client, devices, fault, traitConfig, logger)
	case soundsensorpb.TraitName:
		return newSoundSensor(client, devices, fault, traitConfig, logger)
	case temperaturepb.TraitName:
		return newTemperature(client, devices, fault, traitConfig, logger)
	case transportpb.TraitName:
		return newTransport(client, devices, fault, traitConfig, logger)
	case UdmiMergeName, udmipb.TraitName:
		return newUdmiMerge(client, devices, fault, traitConfig, logger)
	}
	return nil, ErrTraitNotSupported
}

// traitFault drives a trait's health check from the outcome of each poll.
// A failed poll only makes the check abnormal once reads have been failing for at least after,
// so a one-off dropped or slow read doesn't raise a fault that clears itself a poll later.
// A clean poll clears the check straight away.
//
// That clean poll also ends the streak, so a device that fails most polls but succeeds at least once
// every after never goes abnormal. Accepted on purpose: requiring a run of good polls before clearing
// would leave a real recovery showing as a fault for up to after, and a device that intermittent still
// answers, so its trait values keep updating.
//
// The poll loop and Get/Update calls poll concurrently, so update is safe to call from many goroutines.
// A nil *traitFault, or one with a nil FaultCheck, does nothing.
type traitFault struct {
	fc    *healthpb.FaultCheck
	after time.Duration

	mu           sync.Mutex
	failingSince time.Time // zero while the last poll succeeded
}

func newTraitFault(fc *healthpb.FaultCheck, after time.Duration) *traitFault {
	return &traitFault{fc: fc, after: after}
}

// update records the outcome of one poll of the trait name of kind trait, whose reads returned errs.
// Nil entries in errs are successful reads.
func (f *traitFault) update(name string, trait trait.Name, errs []error) {
	if f == nil || f.fc == nil {
		return
	}

	var descriptions []string
	for _, err := range errs {
		if err == nil {
			continue
		}
		descriptions = append(descriptions, err.Error())
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(descriptions) == 0 {
		// No (non-nil) errors, clear any existing fault for this trait.
		f.failingSince = time.Time{}
		f.fc.ClearFaults()
		return
	}
	now := time.Now()
	if f.failingSince.IsZero() {
		f.failingSince = now
	}
	if now.Sub(f.failingSince) < f.after {
		return
	}
	f.fc.SetFault(&healthpb.HealthCheck_Error{
		SummaryText: fmt.Sprintf("%s[%s] has %d errors", name, trait.String(), len(descriptions)),
		DetailsText: fmt.Sprintf("Trait %s errors: %s", trait, strings.Join(descriptions, "; ")),
		Code: &healthpb.HealthCheck_Error_Code{
			Code:   BacNetCommsError,
			System: SystemName,
		},
	})
}

func raisePointAlarm(point string, code string, summary string, value string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&healthpb.HealthCheck_Error{
		SummaryText: summary,
		DetailsText: fmt.Sprintf("Error on point %q, value: %s", point, value),
		Code: &healthpb.HealthCheck_Error_Code{
			Code:   code,
			System: SystemName,
		},
	})
}

func removePointAlarm(code string, fc *healthpb.FaultCheck) {
	fc.RemoveFault(&healthpb.HealthCheck_Error{
		Code: &healthpb.HealthCheck_Error_Code{
			Code: code,
		},
	})
}

// PollTaskProvider is implemented by traits that expose their background poll task,
// allowing the driver to attach keepalive behaviour independently of client subscriptions.
type PollTaskProvider interface {
	PollTask() *task.Intermittent
}
