package alltraits

import (
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/accesspb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/anprcamera"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/button"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/dalipb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/emergencylightpb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/meter"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/mqttpb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/report"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/securityevent"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/serviceticketpb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/soundsensorpb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/temperaturepb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/transport"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/udmipb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/wastepb"
)

var serviceRegistry = map[trait.Name][]grpc.ServiceDesc{
	trait.AirQualitySensor: {traits.AirQualitySensorApi_ServiceDesc, traits.AirQualitySensorInfo_ServiceDesc, gen.AirQualitySensorHistory_ServiceDesc},
	trait.AirTemperature:   {traits.AirTemperatureApi_ServiceDesc, traits.AirTemperatureInfo_ServiceDesc, gen.AirTemperatureHistory_ServiceDesc},
	trait.Booking:          {traits.BookingApi_ServiceDesc, traits.BookingInfo_ServiceDesc},
	trait.BrightnessSensor: {traits.BrightnessSensorApi_ServiceDesc, traits.BrightnessSensorInfo_ServiceDesc},
	trait.Channel:          {traits.ChannelApi_ServiceDesc, traits.ChannelInfo_ServiceDesc},
	trait.Color:            {traits.ColorApi_ServiceDesc, traits.ColorInfo_ServiceDesc},
	trait.Count:            {traits.CountApi_ServiceDesc, traits.CountInfo_ServiceDesc},
	trait.Electric:         {traits.ElectricApi_ServiceDesc, traits.ElectricInfo_ServiceDesc, gen.ElectricHistory_ServiceDesc},
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
	trait.OccupancySensor:  {traits.OccupancySensorApi_ServiceDesc, traits.OccupancySensorInfo_ServiceDesc, gen.OccupancySensorHistory_ServiceDesc},
	trait.OnOff:            {traits.OnOffApi_ServiceDesc, traits.OnOffInfo_ServiceDesc},
	trait.OpenClose:        {traits.OpenCloseApi_ServiceDesc, traits.OpenCloseInfo_ServiceDesc},
	trait.Parent:           {traits.ParentApi_ServiceDesc, traits.ParentInfo_ServiceDesc},
	trait.Publication:      {traits.PublicationApi_ServiceDesc},
	trait.Ptz:              {traits.PtzApi_ServiceDesc, traits.PtzInfo_ServiceDesc},
	trait.Speaker:          {traits.SpeakerApi_ServiceDesc, traits.SpeakerInfo_ServiceDesc},
	trait.Vending:          {traits.VendingApi_ServiceDesc, traits.VendingInfo_ServiceDesc},

	// sc-bos private traits
	accesspb.TraitName:         {gen.AccessApi_ServiceDesc},
	anprcamera.TraitName:       {gen.AnprCameraApi_ServiceDesc},
	button.TraitName:           {gen.ButtonApi_ServiceDesc},
	dalipb.TraitName:           {gen.DaliApi_ServiceDesc},
	emergencylightpb.TraitName: {gen.DaliApi_ServiceDesc, gen.EmergencyLightApi_ServiceDesc},
	healthpb.TraitName:         {gen.HealthApi_ServiceDesc, gen.HealthHistory_ServiceDesc},
	meter.TraitName:            {gen.MeterApi_ServiceDesc, gen.MeterInfo_ServiceDesc, gen.MeterHistory_ServiceDesc},
	mqttpb.TraitName:           {gen.MqttService_ServiceDesc},
	report.TraitName:           {gen.ReportApi_ServiceDesc},
	securityevent.TraitName:    {gen.SecurityEventApi_ServiceDesc},
	serviceticketpb.TraitName:  {gen.ServiceTicketApi_ServiceDesc, gen.ServiceTicketInfo_ServiceDesc},
	soundsensorpb.TraitName:    {gen.SoundSensorApi_ServiceDesc, gen.SoundSensorInfo_ServiceDesc},
	statusTraitName:            {gen.StatusApi_ServiceDesc, gen.StatusHistory_ServiceDesc},
	temperaturepb.TraitName:    {gen.TemperatureApi_ServiceDesc},
	transport.TraitName:        {gen.TransportApi_ServiceDesc, gen.TransportInfo_ServiceDesc},
	udmipb.TraitName:           {gen.UdmiService_ServiceDesc},
	wastepb.TraitName:          {gen.WasteApi_ServiceDesc, gen.WasteInfo_ServiceDesc},
}

// ServiceDesc returns the gRPC service descriptors for all services associated with the given trait.
func ServiceDesc(t trait.Name) []grpc.ServiceDesc {
	return serviceRegistry[t]
}

// had to add this to resolve an import cycle
// TODO: resolve import cycle
const statusTraitName trait.Name = "smartcore.bos.Status"
