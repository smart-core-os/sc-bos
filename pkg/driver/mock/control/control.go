// Package control implements the MockDeviceApi gRPC service for the mock driver.
package control

import (
	"context"
	"errors"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/mockpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// ForceFunc updates a trait model to the value described by valueJSON (protojson-encoded).
type ForceFunc func(valueJSON string) error

// Controller implements MockDeviceApiServer and dispatches MockDeviceApi calls.
type Controller struct {
	mockpb.UnimplementedMockDeviceApiServer

	mu         sync.RWMutex
	devices    map[string]map[string]ForceFunc                    // deviceName -> traitName -> fn
	lifecycles map[string]map[string]map[string]service.Lifecycle // deviceName -> traitName -> id -> automation
}

func New() *Controller {
	return &Controller{
		devices:    make(map[string]map[string]ForceFunc),
		lifecycles: make(map[string]map[string]map[string]service.Lifecycle),
	}
}

// Register records fn as the handler for forcing deviceName's traitName.
// Returns an undo function that removes the registration.
func (c *Controller) Register(deviceName, traitName string, fn ForceFunc) func() {
	c.mu.Lock()
	if c.devices[deviceName] == nil {
		c.devices[deviceName] = make(map[string]ForceFunc)
	}
	c.devices[deviceName][traitName] = fn
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		if m := c.devices[deviceName]; m != nil {
			delete(m, traitName)
		}
		c.mu.Unlock()
	}
}

// RegisterLifecycle records slc as the automation identified by id for deviceName's traitName.
// Returns an undo function that removes the registration.
func (c *Controller) RegisterLifecycle(deviceName, traitName, id string, slc service.Lifecycle) func() {
	c.mu.Lock()
	if c.lifecycles[deviceName] == nil {
		c.lifecycles[deviceName] = make(map[string]map[string]service.Lifecycle)
	}
	if c.lifecycles[deviceName][traitName] == nil {
		c.lifecycles[deviceName][traitName] = make(map[string]service.Lifecycle)
	}
	c.lifecycles[deviceName][traitName][id] = slc
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		if traits := c.lifecycles[deviceName]; traits != nil {
			if ids := traits[traitName]; ids != nil {
				delete(ids, id)
			}
		}
		c.mu.Unlock()
	}
}

func (c *Controller) ForceTraitValue(_ context.Context, req *mockpb.ForceTraitValuesRequest) (*mockpb.ForceTraitValuesResponse, error) {
	c.mu.RLock()
	type pending struct {
		fn    ForceFunc
		trait string
		json  string
	}
	var calls []pending
	for _, v := range req.GetValues() {
		var fn ForceFunc
		if traits, ok := c.devices[req.GetName()]; ok {
			fn = traits[v.GetTrait()]
		}
		if fn == nil {
			c.mu.RUnlock()
			return nil, status.Errorf(codes.NotFound, "mock device %q trait %q not found", req.GetName(), v.GetTrait())
		}
		calls = append(calls, pending{fn: fn, trait: v.GetTrait(), json: v.GetValueProtojson()})
	}
	c.mu.RUnlock()

	for _, p := range calls {
		if err := p.fn(p.json); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "force trait %q: %v", p.trait, err)
		}
	}
	return &mockpb.ForceTraitValuesResponse{}, nil
}

func (c *Controller) SetDeviceAutomation(_ context.Context, req *mockpb.SetDeviceAutomationsRequest) (*mockpb.SetDeviceAutomationsResponse, error) {
	c.mu.RLock()
	type target struct {
		slc    service.Lifecycle
		active bool
	}
	var targets []target
	for _, a := range req.GetAutomations() {
		found := false
		deviceTraits := c.lifecycles[req.GetName()]
		for traitName, ids := range deviceTraits {
			if a.GetTrait() != "" && a.GetTrait() != traitName {
				continue
			}
			for id, slc := range ids {
				if a.GetId() != "" && a.GetId() != id {
					continue
				}
				targets = append(targets, target{slc: slc, active: a.GetActive()})
				found = true
			}
		}
		// If trait is empty it means "all traits"; no match is a valid no-op (device may have no automations).
		// If trait is non-empty and nothing matched, the caller specified a trait that isn't registered.
		if !found && a.GetTrait() != "" {
			c.mu.RUnlock()
			return nil, status.Errorf(codes.NotFound, "mock device %q trait %q has no automation", req.GetName(), a.GetTrait())
		}
	}
	c.mu.RUnlock()

	for _, t := range targets {
		var err error
		if t.active {
			_, err = t.slc.Start()
		} else {
			_, err = t.slc.Stop()
		}
		if err != nil && !errors.Is(err, service.ErrAlreadyStarted) && !errors.Is(err, service.ErrAlreadyStopped) {
			return nil, status.Errorf(codes.Internal, "set automation active=%v: %v", t.active, err)
		}
	}
	return &mockpb.SetDeviceAutomationsResponse{}, nil
}
