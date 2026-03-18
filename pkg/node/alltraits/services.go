package alltraits

import (
	"slices"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/anprcamerapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/bookingpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/channelpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/colorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/countpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/driver/dalipb"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencypb"
	"github.com/smart-core-os/sc-bos/pkg/proto/energystoragepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/extendretractpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/fanspeedpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hailpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/inputselectpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lockunlockpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/microphonepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/modepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/motionsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mqttpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/openclosepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/parentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/ptzpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/reportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/serviceticketpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/speakerpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/proto/vendingpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/wastepb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

var serviceRegistry = map[trait.Name][]grpc.ServiceDesc{
	trait.AirQualitySensor: {airqualitysensorpb.AirQualitySensorApi_ServiceDesc, airqualitysensorpb.AirQualitySensorInfo_ServiceDesc, airqualitysensorpb.AirQualitySensorHistory_ServiceDesc},
	trait.AirTemperature:   {airtemperaturepb.AirTemperatureApi_ServiceDesc, airtemperaturepb.AirTemperatureInfo_ServiceDesc, airtemperaturepb.AirTemperatureHistory_ServiceDesc},
	trait.Booking:          {bookingpb.BookingApi_ServiceDesc, bookingpb.BookingInfo_ServiceDesc},
	trait.BrightnessSensor: {brightnesssensorpb.BrightnessSensorApi_ServiceDesc, brightnesssensorpb.BrightnessSensorInfo_ServiceDesc},
	trait.Channel:          {channelpb.ChannelApi_ServiceDesc, channelpb.ChannelInfo_ServiceDesc},
	trait.Color:            {colorpb.ColorApi_ServiceDesc, colorpb.ColorInfo_ServiceDesc},
	trait.Count:            {countpb.CountApi_ServiceDesc, countpb.CountInfo_ServiceDesc},
	trait.Electric:         {electricpb.ElectricApi_ServiceDesc, electricpb.ElectricInfo_ServiceDesc, electricpb.ElectricHistory_ServiceDesc},
	trait.Emergency:        {emergencypb.EmergencyApi_ServiceDesc, emergencypb.EmergencyInfo_ServiceDesc},
	trait.EnergyStorage:    {energystoragepb.EnergyStorageApi_ServiceDesc, energystoragepb.EnergyStorageInfo_ServiceDesc},
	trait.EnterLeaveSensor: {enterleavesensorpb.EnterLeaveSensorApi_ServiceDesc, enterleavesensorpb.EnterLeaveSensorInfo_ServiceDesc},
	trait.ExtendRetract:    {extendretractpb.ExtendRetractApi_ServiceDesc, extendretractpb.ExtendRetractInfo_ServiceDesc},
	trait.FanSpeed:         {fanspeedpb.FanSpeedApi_ServiceDesc, fanspeedpb.FanSpeedInfo_ServiceDesc},
	trait.Hail:             {hailpb.HailApi_ServiceDesc, hailpb.HailInfo_ServiceDesc},
	trait.InputSelect:      {inputselectpb.InputSelectApi_ServiceDesc, inputselectpb.InputSelectInfo_ServiceDesc},
	trait.Light:            {lightpb.LightApi_ServiceDesc, lightpb.LightInfo_ServiceDesc},
	trait.LockUnlock:       {lockunlockpb.LockUnlockApi_ServiceDesc, lockunlockpb.LockUnlockInfo_ServiceDesc},
	trait.Metadata:         {metadatapb.MetadataApi_ServiceDesc, metadatapb.MetadataInfo_ServiceDesc},
	trait.Microphone:       {microphonepb.MicrophoneApi_ServiceDesc, microphonepb.MicrophoneInfo_ServiceDesc},
	trait.Mode:             {modepb.ModeApi_ServiceDesc, modepb.ModeInfo_ServiceDesc},
	trait.MotionSensor:     {motionsensorpb.MotionSensorApi_ServiceDesc},
	trait.OccupancySensor:  {occupancysensorpb.OccupancySensorApi_ServiceDesc, occupancysensorpb.OccupancySensorInfo_ServiceDesc, occupancysensorpb.OccupancySensorHistory_ServiceDesc},
	trait.OnOff:            {onoffpb.OnOffApi_ServiceDesc, onoffpb.OnOffInfo_ServiceDesc},
	trait.OpenClose:        {openclosepb.OpenCloseApi_ServiceDesc, openclosepb.OpenCloseInfo_ServiceDesc},
	trait.Parent:           {parentpb.ParentApi_ServiceDesc, parentpb.ParentInfo_ServiceDesc},
	trait.Publication:      {publicationpb.PublicationApi_ServiceDesc},
	trait.Ptz:              {ptzpb.PtzApi_ServiceDesc, ptzpb.PtzInfo_ServiceDesc},
	trait.Speaker:          {speakerpb.SpeakerApi_ServiceDesc, speakerpb.SpeakerInfo_ServiceDesc},
	trait.Vending:          {vendingpb.VendingApi_ServiceDesc, vendingpb.VendingInfo_ServiceDesc},

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
