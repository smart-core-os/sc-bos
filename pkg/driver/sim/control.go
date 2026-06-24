package sim

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/fanspeedpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mockpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

// controller implements the MockDeviceApi for the sim driver. Unlike the mock
// driver — where each device is independent — a forced value here drives the
// coupled engine: forcing a room's occupancy, temperature set point, lighting or
// fan speed becomes a simulation input and the rest of the building responds.
// Forcing a derived output (power, metering, brightness, CO2, motion, enter/leave)
// is rejected, as those follow from the inputs above.
//
// SetDeviceAutomation toggles whether the engine drives an input: active=false
// freezes it at its current value, active=true releases it back to the simulation.
type controller struct {
	mockpb.UnimplementedMockDeviceApiServer
	b       *Building
	devices map[string]forceable // deviceName -> its room and forceable traits
}

// forceable records, for one device, the room it belongs to and which engine
// override each of its traits drives.
type forceable struct {
	room   *Room
	traits map[trait.Name]roomOverride
}

func newController(b *Building) *controller {
	return &controller{b: b, devices: make(map[string]forceable)}
}

// register records a device's forceable traits. Called during Expand, before the
// engine starts, so the map is read-only by the time requests can arrive.
func (c *controller) register(name string, room *Room, traits map[trait.Name]roomOverride) {
	c.devices[name] = forceable{room: room, traits: traits}
}

func (c *controller) lookup(name string) (forceable, error) {
	f, ok := c.devices[name]
	if !ok {
		return forceable{}, status.Errorf(codes.NotFound, "device %q is not a forceable simulated device", name)
	}
	return f, nil
}

func (c *controller) ForceTraitValue(_ context.Context, req *mockpb.ForceTraitValuesRequest) (*mockpb.ForceTraitValuesResponse, error) {
	f, err := c.lookup(req.GetName())
	if err != nil {
		return nil, err
	}
	for _, tv := range req.GetValues() {
		o, ok := f.traits[trait.Name(tv.GetTrait())]
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument,
				"trait %q on %q is derived from the simulation and cannot be forced; force occupancy, air temperature set point, lighting or fan speed instead",
				tv.GetTrait(), req.GetName())
		}
		v, err := parseOverride(o, tv.GetValueProtojson())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "force trait %q: %v", tv.GetTrait(), err)
		}
		room := f.room
		if err := c.b.submit(func() { room.set(o, v) }); err != nil {
			return nil, status.Error(codes.Unavailable, err.Error())
		}
	}
	return &mockpb.ForceTraitValuesResponse{}, nil
}

func (c *controller) SetDeviceAutomation(_ context.Context, req *mockpb.SetDeviceAutomationsRequest) (*mockpb.SetDeviceAutomationsResponse, error) {
	f, err := c.lookup(req.GetName())
	if err != nil {
		return nil, err
	}
	for _, a := range req.GetAutomations() {
		// An empty trait selects every forceable trait on the device.
		for tn, ov := range f.traits {
			if t := a.GetTrait(); t != "" && t != string(tn) {
				continue
			}
			room, o, active := f.room, ov, a.GetActive()
			err := c.b.submit(func() {
				if active {
					room.release(o)
				} else {
					room.hold(o)
				}
			})
			if err != nil {
				return nil, status.Error(codes.Unavailable, err.Error())
			}
		}
	}
	return &mockpb.SetDeviceAutomationsResponse{}, nil
}

// parseOverride extracts the engine input value from a trait's protojson value.
func parseOverride(o roomOverride, valueJSON string) (float64, error) {
	switch o {
	case overrideOccupancy:
		var v occupancysensorpb.Occupancy
		if err := protojson.Unmarshal([]byte(valueJSON), &v); err != nil {
			return 0, err
		}
		return float64(v.GetPeopleCount()), nil
	case overrideSetPoint:
		var v airtemperaturepb.AirTemperature
		if err := protojson.Unmarshal([]byte(valueJSON), &v); err != nil {
			return 0, err
		}
		sp := v.GetTemperatureSetPoint()
		if sp == nil {
			return 0, fmt.Errorf("expected temperatureSetPoint.valueCelsius")
		}
		return sp.GetValueCelsius(), nil
	case overrideLight:
		var v lightpb.Brightness
		if err := protojson.Unmarshal([]byte(valueJSON), &v); err != nil {
			return 0, err
		}
		return float64(v.GetLevelPercent()), nil
	case overrideFan:
		var v fanspeedpb.FanSpeed
		if err := protojson.Unmarshal([]byte(valueJSON), &v); err != nil {
			return 0, err
		}
		return float64(v.GetPercentage()), nil
	}
	return 0, fmt.Errorf("unsupported override")
}
