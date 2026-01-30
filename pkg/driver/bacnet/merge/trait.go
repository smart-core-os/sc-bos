package merge

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/accesspb"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/securityevent"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/temperaturepb"
	transportpb "github.com/smart-core-os/sc-bos/pkg/gentrait/transport"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

const (
	BacNetCommsError = "BACnet Communication Error"

	SystemName = "BACnet"
)

func IntoTrait(client *gobacnet.Client, devices known.Context, faultCheck *gen_healthpb.FaultCheck, traitConfig config.RawTrait, logger *zap.Logger) (node.SelfAnnouncer, error) {
	// todo: implement some traits that pull data from different bacnet devices.
	switch traitConfig.Kind {
	case trait.AirQualitySensor:
		return newAirQualitySensor(client, devices, faultCheck, traitConfig, logger)
	case trait.AirTemperature:
		return newAirTemperature(client, devices, faultCheck, traitConfig, logger)
	case trait.Electric:
		return newElectric(client, devices, faultCheck, traitConfig, logger)
	case trait.Emergency:
		return newEmergency(client, devices, faultCheck, traitConfig, logger)
	case trait.EnergyStorage:
		return newEnergyStorage(client, devices, faultCheck, traitConfig, logger)
	case trait.FanSpeed:
		return newFanSpeed(client, devices, faultCheck, traitConfig, logger)
	case trait.Light:
		return newLight(client, devices, faultCheck, traitConfig, logger)
	case meter.TraitName:
		return newMeter(client, devices, faultCheck, traitConfig, logger)
	case trait.Mode:
		return newMode(client, devices, faultCheck, traitConfig, logger)
	case trait.OccupancySensor:
		return newOccupancy(client, devices, faultCheck, traitConfig, logger)
	case trait.OnOff:
		return newOnOff(client, devices, faultCheck, traitConfig, logger)
	case accesspb.TraitName:
		return newAccess(client, devices, faultCheck, traitConfig, logger)
	case securityevent.TraitName:
		return newSecurityEvent(client, devices, faultCheck, traitConfig, logger)
	case temperaturepb.TraitName:
		return newTemperature(client, devices, faultCheck, traitConfig, logger)
	case transportpb.TraitName:
		return newTransport(client, devices, faultCheck, traitConfig, logger)
	case UdmiMergeName, udmipb.TraitName:
		return newUdmiMerge(client, devices, faultCheck, traitConfig, logger)
	}
	return nil, ErrTraitNotSupported
}

func updateTraitFaultCheck(ctx context.Context, faultCheck *gen_healthpb.FaultCheck, name string, trait trait.Name, errs []error) {
	if faultCheck == nil {
		return
	}
	if len(errs) == 0 {
		// Clear any existing fault for this trait.
		faultCheck.ClearFaults()
		return
	}

	var descriptions []string
	for _, err := range errs {
		descriptions = append(descriptions, err.Error())

	}
	faultCheck.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
		State: healthpb.HealthCheck_Reliability_UNRELIABLE,
		LastError: &healthpb.HealthCheck_Error{
			SummaryText: fmt.Sprintf("%s[%s] has %d errors", name, trait.String(), len(errs)),
			DetailsText: fmt.Sprintf("Trait %s errors: %s", trait, strings.Join(descriptions, "; ")),
			Code: &healthpb.HealthCheck_Error_Code{
				Code:   BacNetCommsError,
				System: SystemName,
			},
		},
	})
}

func raisePointAlarm(point string, code string, summary string, fc *gen_healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&healthpb.HealthCheck_Error{
		SummaryText: summary,
		DetailsText: "An alarm has been detected on point: " + point,
		Code: &healthpb.HealthCheck_Error_Code{
			Code:   code,
			System: SystemName,
		},
	})
}

func removePointAlarm(code string, fc *gen_healthpb.FaultCheck) {
	fc.RemoveFault(&healthpb.HealthCheck_Error{
		Code: &healthpb.HealthCheck_Error_Code{
			Code: code,
		},
	})
}
