package trait

import (
	"strings"
)

type Name string

// Local returns the last part of a fully qualified trait name.
func (n Name) Local() string {
	traitName := string(n)
	lastDot := strings.LastIndex(traitName, ".")
	if lastDot == -1 {
		return traitName
	}
	return traitName[lastDot+1:]
}

func (n Name) String() string {
	return string(n)
}

const (
	Access           Name = "smartcore.traits.Access"
	AirQualitySensor Name = "smartcore.traits.AirQualitySensor"
	AirTemperature   Name = "smartcore.traits.AirTemperature"
	Booking          Name = "smartcore.traits.Booking"
	BrightnessSensor Name = "smartcore.traits.BrightnessSensor"
	Channel          Name = "smartcore.traits.Channel"
	Color            Name = "smartcore.traits.Color"
	Count            Name = "smartcore.traits.Count"
	Electric         Name = "smartcore.traits.Electric"
	Emergency        Name = "smartcore.traits.Emergency"
	EnergyStorage    Name = "smartcore.traits.EnergyStorage"
	EnterLeaveSensor Name = "smartcore.traits.EnterLeaveSensor"
	ExtendRetract    Name = "smartcore.traits.ExtendRetract"
	FanSpeed         Name = "smartcore.traits.FanSpeed"
	Hail             Name = "smartcore.traits.Hail"
	InputSelect      Name = "smartcore.traits.InputSelect"
	Light            Name = "smartcore.traits.Light"
	LockUnlock       Name = "smartcore.traits.LockUnlock"
	Metadata         Name = "smartcore.traits.Metadata"
	Meter            Name = "smartcore.traits.Meter"
	Microphone       Name = "smartcore.traits.Microphone"
	Mode             Name = "smartcore.traits.Mode"
	MotionSensor     Name = "smartcore.traits.MotionSensor"
	OccupancySensor  Name = "smartcore.traits.OccupancySensor"
	OnOff            Name = "smartcore.traits.OnOff"
	OpenClose        Name = "smartcore.traits.OpenClose"
	Parent           Name = "smartcore.traits.Parent"
	Press            Name = "smartcore.traits.Press"
	Publication      Name = "smartcore.traits.Publication"
	Ptz              Name = "smartcore.traits.Ptz"
	Speaker          Name = "smartcore.traits.Speaker"
	Temperature      Name = "smartcore.traits.Temperature"
	Vending          Name = "smartcore.traits.Vending"
	Waste            Name = "smartcore.traits.Waste"
)
