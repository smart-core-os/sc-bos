package alltraits

import (
	"slices"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/anprcamerapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/driver/dalipb"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mqttpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/reportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/serviceticketpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/proto/wastepb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

var serviceRegistry = map[trait.Name][]grpc.ServiceDesc{
	trait.AirQualitySensor: {traits.AirQualitySensorApi_ServiceDesc, traits.AirQualitySensorInfo_ServiceDesc, airqualitysensorpb.AirQualitySensorHistory_ServiceDesc},
	trait.AirTemperature:   {traits.AirTemperatureApi_ServiceDesc, traits.AirTemperatureInfo_ServiceDesc, airtemperaturepb.AirTemperatureHistory_ServiceDesc},
	trait.Booking:          {traits.BookingApi_ServiceDesc, traits.BookingInfo_ServiceDesc},
	trait.BrightnessSensor: {traits.BrightnessSensorApi_ServiceDesc, traits.BrightnessSensorInfo_ServiceDesc},
	trait.Channel:          {traits.ChannelApi_ServiceDesc, traits.ChannelInfo_ServiceDesc},
	trait.Color:            {traits.ColorApi_ServiceDesc, traits.ColorInfo_ServiceDesc},
	trait.Count:            {traits.CountApi_ServiceDesc, traits.CountInfo_ServiceDesc},
	trait.Electric:         {traits.ElectricApi_ServiceDesc, traits.ElectricInfo_ServiceDesc, electricpb.ElectricHistory_ServiceDesc},
	trait.Emergency:        {traits.EmergencyApi_ServiceDesc, traits.EmergencyInfo_ServiceDesc},
	trait.EnergyStorage:    {traits.EnergyStorageApi_ServiceDesc, traits.EnergyStorageInfo_ServiceDesc},
	trait.EnterLeaveSensor: {traits.EnterLeaveSensorApi_ServiceDesc, traits.EnterLeaveSensorInfo_ServiceDesc},
	trait.ExtendRetract:    {traits.ExtendRetractApi_ServiceDesc, traits.ExtendRetractInfo_ServiceDesc},
	trait.FanSpeed:         {traits.FanSpeedApi_ServiceDesc, traits.FanSpeedInfo_ServiceDesc},
	trait.Hail:             {traits.HailApi_ServiceDesc, traits.HailInfo_ServiceDesc},
	trait.InputSelect:      {traits.InputSelectApi_ServiceDesc, traits.InputSelectInfo_ServiceDesc},
	trait.Light:            {traits.LightApi_ServiceDesc, traits.LightInfo_ServiceDesc},
	trait.LockUnlock:       {traits.LockUnlockApi_ServiceDesc, traits.LockUnlockInfo_ServiceDesc},
	trait.Metadata:         {traits.MetadataApi_ServiceDesc, traits.MetadataInfo_ServiceDesc},
	trait.Microphone:       {traits.MicrophoneApi_ServiceDesc, traits.MicrophoneInfo_ServiceDesc},
	trait.Mode:             {traits.ModeApi_ServiceDesc, traits.ModeInfo_ServiceDesc},
	trait.MotionSensor:     {traits.MotionSensorApi_ServiceDesc},
	trait.OccupancySensor:  {traits.OccupancySensorApi_ServiceDesc, traits.OccupancySensorInfo_ServiceDesc, occupancysensorpb.OccupancySensorHistory_ServiceDesc},
	trait.OnOff:            {traits.OnOffApi_ServiceDesc, traits.OnOffInfo_ServiceDesc},
	trait.OpenClose:        {traits.OpenCloseApi_ServiceDesc, traits.OpenCloseInfo_ServiceDesc},
	trait.Parent:           {traits.ParentApi_ServiceDesc, traits.ParentInfo_ServiceDesc},
	trait.Publication:      {traits.PublicationApi_ServiceDesc},
	trait.Ptz:              {traits.PtzApi_ServiceDesc, traits.PtzInfo_ServiceDesc},
	trait.Speaker:          {traits.SpeakerApi_ServiceDesc, traits.SpeakerInfo_ServiceDesc},
	trait.Vending:          {traits.VendingApi_ServiceDesc, traits.VendingInfo_ServiceDesc},

	// sc-bos private traits
	allocationpb.TraitName:     {allocationpb.AllocationApi_ServiceDesc, allocationpb.AllocationHistory_ServiceDesc},
	logpb.TraitName:            {logpb.LogApi_ServiceDesc},
	accesspb.TraitName:         {accesspb.AccessApi_ServiceDesc},
	anprcamerapb.TraitName:     {anprcamerapb.AnprCameraApi_ServiceDesc},
	buttonpb.TraitName:         {buttonpb.ButtonApi_ServiceDesc},
	dalipb.TraitName:           {dalipb.DaliApi_ServiceDesc},
	emergencylightpb.TraitName: {dalipb.DaliApi_ServiceDesc, emergencylightpb.EmergencyLightApi_ServiceDesc},
	healthpb.TraitName:         {healthpb.HealthApi_ServiceDesc, healthpb.HealthHistory_ServiceDesc},
	meterpb.TraitName:          {meterpb.MeterApi_ServiceDesc, meterpb.MeterInfo_ServiceDesc, meterpb.MeterHistory_ServiceDesc},
	mqttpb.TraitName:           {mqttpb.MqttService_ServiceDesc},
	reportpb.TraitName:         {reportpb.ReportApi_ServiceDesc},
	resourceusepb.TraitName:    {resourceusepb.ResourceUseApi_ServiceDesc, resourceusepb.ResourceUseHistory_ServiceDesc},
	securityeventpb.TraitName:  {securityeventpb.SecurityEventApi_ServiceDesc},
	serviceticketpb.TraitName:  {serviceticketpb.ServiceTicketApi_ServiceDesc, serviceticketpb.ServiceTicketInfo_ServiceDesc},
	soundsensorpb.TraitName:    {soundsensorpb.SoundSensorApi_ServiceDesc, soundsensorpb.SoundSensorInfo_ServiceDesc},
	statusTraitName:            {statuspb.StatusApi_ServiceDesc, statuspb.StatusHistory_ServiceDesc},
	temperaturepb.TraitName:    {temperaturepb.TemperatureApi_ServiceDesc},
	transportpb.TraitName:      {transportpb.TransportApi_ServiceDesc, transportpb.TransportInfo_ServiceDesc, transportpb.TransportHistory_ServiceDesc},
	udmipb.TraitName:           {udmipb.UdmiService_ServiceDesc},
	wastepb.TraitName:          {wastepb.WasteApi_ServiceDesc, wastepb.WasteInfo_ServiceDesc},
}

func Names() []trait.Name {
	names := make([]trait.Name, 0, len(serviceRegistry))
	for name := range serviceRegistry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// ServiceDesc returns the gRPC service descriptors for all services associated with the given trait.
func ServiceDesc(t trait.Name) []grpc.ServiceDesc {
	return serviceRegistry[t]
}

// had to add this to resolve an import cycle
// TODO: resolve import cycle
const statusTraitName trait.Name = "smartcore.bos.Status"
