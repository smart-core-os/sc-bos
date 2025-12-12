package alltraits

import (
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
)

func NewClient(ptr any, conn grpc.ClientConnInterface) (ok bool) {
	// yes, this is ugly and terrible but we need it to support the legacy node package API
	// This should be binned off as soon as all users have been migrated to the new API
	switch ptr := ptr.(type) {
	case *traits.AirQualitySensorApiClient:
		*ptr = traits.NewAirQualitySensorApiClient(conn)
	case *traits.AirTemperatureApiClient:
		*ptr = traits.NewAirTemperatureApiClient(conn)
	case *traits.BookingApiClient:
		*ptr = traits.NewBookingApiClient(conn)
	case *traits.BrightnessSensorApiClient:
		*ptr = traits.NewBrightnessSensorApiClient(conn)
	case *traits.ChannelApiClient:
		*ptr = traits.NewChannelApiClient(conn)
	case *traits.ColorApiClient:
		*ptr = traits.NewColorApiClient(conn)
	case *traits.CountApiClient:
		*ptr = traits.NewCountApiClient(conn)
	case *traits.ElectricApiClient:
		*ptr = traits.NewElectricApiClient(conn)
	case *traits.EmergencyApiClient:
		*ptr = traits.NewEmergencyApiClient(conn)
	case *traits.EnergyStorageApiClient:
		*ptr = traits.NewEnergyStorageApiClient(conn)
	case *traits.EnterLeaveSensorApiClient:
		*ptr = traits.NewEnterLeaveSensorApiClient(conn)
	case *traits.ExtendRetractApiClient:
		*ptr = traits.NewExtendRetractApiClient(conn)
	case *traits.FanSpeedApiClient:
		*ptr = traits.NewFanSpeedApiClient(conn)
	case *traits.HailApiClient:
		*ptr = traits.NewHailApiClient(conn)
	case *traits.InputSelectApiClient:
		*ptr = traits.NewInputSelectApiClient(conn)
	case *traits.LightApiClient:
		*ptr = traits.NewLightApiClient(conn)
	case *traits.LockUnlockApiClient:
		*ptr = traits.NewLockUnlockApiClient(conn)
	case *traits.MetadataApiClient:
		*ptr = traits.NewMetadataApiClient(conn)
	case *traits.MicrophoneApiClient:
		*ptr = traits.NewMicrophoneApiClient(conn)
	case *traits.ModeApiClient:
		*ptr = traits.NewModeApiClient(conn)
	case *traits.MotionSensorApiClient:
		*ptr = traits.NewMotionSensorApiClient(conn)
	case *traits.OccupancySensorApiClient:
		*ptr = traits.NewOccupancySensorApiClient(conn)
	case *traits.OnOffApiClient:
		*ptr = traits.NewOnOffApiClient(conn)
	case *traits.OpenCloseApiClient:
		*ptr = traits.NewOpenCloseApiClient(conn)
	case *traits.ParentApiClient:
		*ptr = traits.NewParentApiClient(conn)
	case *traits.PublicationApiClient:
		*ptr = traits.NewPublicationApiClient(conn)
	case *traits.PtzApiClient:
		*ptr = traits.NewPtzApiClient(conn)
	case *traits.SpeakerApiClient:
		*ptr = traits.NewSpeakerApiClient(conn)
	case *traits.VendingApiClient:
		*ptr = traits.NewVendingApiClient(conn)
	case *gen.AlertApiClient:
		*ptr = gen.NewAlertApiClient(conn)
	case *gen.ButtonApiClient:
		*ptr = gen.NewButtonApiClient(conn)
	case *gen.DaliApiClient:
		*ptr = gen.NewDaliApiClient(conn)
	case *gen.MeterApiClient:
		*ptr = gen.NewMeterApiClient(conn)
	case *gen.MqttServiceClient:
		*ptr = gen.NewMqttServiceClient(conn)
	case *gen.StatusApiClient:
		*ptr = gen.NewStatusApiClient(conn)
	case *gen.UdmiServiceClient:
		*ptr = gen.NewUdmiServiceClient(conn)
	case *gen.WasteApiClient:
		*ptr = gen.NewWasteApiClient(conn)

	case *traits.AirQualitySensorInfoClient:
		*ptr = traits.NewAirQualitySensorInfoClient(conn)
	case *traits.AirTemperatureInfoClient:
		*ptr = traits.NewAirTemperatureInfoClient(conn)
	case *traits.BookingInfoClient:
		*ptr = traits.NewBookingInfoClient(conn)
	case *traits.BrightnessSensorInfoClient:
		*ptr = traits.NewBrightnessSensorInfoClient(conn)
	case *traits.ChannelInfoClient:
		*ptr = traits.NewChannelInfoClient(conn)
	case *traits.ColorInfoClient:
		*ptr = traits.NewColorInfoClient(conn)
	case *traits.CountInfoClient:
		*ptr = traits.NewCountInfoClient(conn)
	case *traits.ElectricInfoClient:
		*ptr = traits.NewElectricInfoClient(conn)
	case *traits.EmergencyInfoClient:
		*ptr = traits.NewEmergencyInfoClient(conn)
	case *traits.EnergyStorageInfoClient:
		*ptr = traits.NewEnergyStorageInfoClient(conn)
	case *traits.EnterLeaveSensorInfoClient:
		*ptr = traits.NewEnterLeaveSensorInfoClient(conn)
	case *traits.ExtendRetractInfoClient:
		*ptr = traits.NewExtendRetractInfoClient(conn)
	case *traits.FanSpeedInfoClient:
		*ptr = traits.NewFanSpeedInfoClient(conn)
	case *traits.HailInfoClient:
		*ptr = traits.NewHailInfoClient(conn)
	case *traits.InputSelectInfoClient:
		*ptr = traits.NewInputSelectInfoClient(conn)
	case *traits.LightInfoClient:
		*ptr = traits.NewLightInfoClient(conn)
	case *traits.LockUnlockInfoClient:
		*ptr = traits.NewLockUnlockInfoClient(conn)
	case *traits.MetadataInfoClient:
		*ptr = traits.NewMetadataInfoClient(conn)
	case *traits.MicrophoneInfoClient:
		*ptr = traits.NewMicrophoneInfoClient(conn)
	case *traits.ModeInfoClient:
		*ptr = traits.NewModeInfoClient(conn)
	case *traits.OccupancySensorInfoClient:
		*ptr = traits.NewOccupancySensorInfoClient(conn)
	case *traits.OnOffInfoClient:
		*ptr = traits.NewOnOffInfoClient(conn)
	case *traits.OpenCloseInfoClient:
		*ptr = traits.NewOpenCloseInfoClient(conn)
	case *traits.ParentInfoClient:
		*ptr = traits.NewParentInfoClient(conn)
	case *traits.PtzInfoClient:
		*ptr = traits.NewPtzInfoClient(conn)
	case *traits.SpeakerInfoClient:
		*ptr = traits.NewSpeakerInfoClient(conn)
	case *traits.VendingInfoClient:
		*ptr = traits.NewVendingInfoClient(conn)
	case *gen.MeterInfoClient:
		*ptr = gen.NewMeterInfoClient(conn)
	case *gen.WasteInfoClient:
		*ptr = gen.NewWasteInfoClient(conn)

	case *gen.AllocationHistoryClient:
		*ptr = gen.NewAllocationHistoryClient(conn)
	case *gen.AirTemperatureHistoryClient:
		*ptr = gen.NewAirTemperatureHistoryClient(conn)
	case *gen.ElectricHistoryClient:
		*ptr = gen.NewElectricHistoryClient(conn)
	case *gen.OccupancySensorHistoryClient:
		*ptr = gen.NewOccupancySensorHistoryClient(conn)
	case *gen.AirQualitySensorHistoryClient:
		*ptr = gen.NewAirQualitySensorHistoryClient(conn)
	case *gen.MeterHistoryClient:
		*ptr = gen.NewMeterHistoryClient(conn)
	case *gen.StatusHistoryClient:
		*ptr = gen.NewStatusHistoryClient(conn)

	case *gen.HistoryAdminApiClient:
		*ptr = gen.NewHistoryAdminApiClient(conn)
	case *gen.HubApiClient:
		*ptr = gen.NewHubApiClient(conn)
	case *gen.TenantApiClient:
		*ptr = gen.NewTenantApiClient(conn)

	default:
		return false
	}
	return true
}
