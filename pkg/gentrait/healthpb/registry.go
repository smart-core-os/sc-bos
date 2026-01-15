package healthpb

import (
	"errors"
	"fmt"
	"sync"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

var (
	// ErrAlreadyExists is returned when attempting to create a health check with the same name and ID as an existing check.
	ErrAlreadyExists = errors.New("already exists")
	// ErrInvalid is returned when arguments to Registry or Checks methods are invalid.
	ErrInvalid = errors.New("invalid")
)

// Registry manages a collection of health checks.
type Registry struct {
	mu     sync.RWMutex
	byName map[string]*namedChecks

	onNameCreate  func(name string)
	onCheckCreate func(name string, c *healthpb.HealthCheck) *healthpb.HealthCheck
	onCheckUpdate func(name string, c *healthpb.HealthCheck)
	onCheckDelete func(name, id string)
	onNameDelete  func(name string)
}

// NewRegistry creates a new Registry instance.
func NewRegistry(opts ...RegistryOption) *Registry {
	r := &Registry{}
	for _, o := range opts {
		o.apply(r)
	}
	return r
}

// RegistryOption is an option for configuring a Registry.
type RegistryOption interface {
	apply(*Registry)
}

type registryOptionFunc func(*Registry)

func (f registryOptionFunc) apply(r *Registry) {
	f(r)
}

// WithOnNameCreate configures a callback that is invoked when a new name is created.
func WithOnNameCreate(f func(name string)) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.onNameCreate = f
	})
}

// WithOnCheckCreate configures a callback that is invoked when a new check is created.
// The callback may return a different HealthCheck instance to be used instead of the one passed in.
func WithOnCheckCreate(f func(name string, c *healthpb.HealthCheck) *healthpb.HealthCheck) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.onCheckCreate = f
	})
}

// WithOnCheckUpdate configures a callback that is invoked when a check is updated.
func WithOnCheckUpdate(f func(name string, c *healthpb.HealthCheck)) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.onCheckUpdate = f
	})
}

// WithOnCheckDelete configures a callback that is invoked when a check is deleted.
func WithOnCheckDelete(f func(name, id string)) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.onCheckDelete = f
	})
}

// WithOnNameDelete configures a callback that is invoked when the last check for a name is deleted.
func WithOnNameDelete(f func(name string)) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.onNameDelete = f
	})
}

func (r *Registry) GetCheck(name, id string) *healthpb.HealthCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nc, ok := r.byName[name]
	if !ok {
		return nil
	}
	c, ok := nc.byId[id]
	if !ok {
		return nil
	}
	return c.check
}

// ForOwner returns a Checks instance that can create checks owned by the given owner.
// The owner should be unique within the node.
func (r *Registry) ForOwner(owner string) *Checks {
	return &Checks{
		r:     r,
		owner: owner,
	}
}

// addCheck adds c, identified by id, as a health check against name.
// ErrAlreadyExists is returned if id already exists for name.
func (r *Registry) addCheck(name, id string, c *checkBase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byName == nil {
		r.byName = make(map[string]*namedChecks)
	}
	nc, ok := r.byName[name]
	if !ok {
		nc = &namedChecks{
			n:    name,
			byId: make(map[string]*checkBase),
		}
		r.byName[name] = nc
		if r.onNameCreate != nil {
			r.onNameCreate(name)
		}
	}
	_, exists := nc.byId[id]
	if exists {
		return fmt.Errorf("%w: %s[%s]", ErrAlreadyExists, name, id)
	}

	// any non-present fields that should exist are filled in here
	r.initHealthCheck(c.check)

	if r.onCheckCreate != nil {
		created := r.onCheckCreate(name, c.check)
		if created != nil {
			c.check = created
		}
	}
	c.onCommit = func(c *healthpb.HealthCheck) {
		if r.onCheckUpdate != nil {
			r.onCheckUpdate(name, c)
		}
	}
	c.onDispose = func(c *healthpb.HealthCheck) {
		// clean up our own state
		r.mu.Lock()
		delete(nc.byId, id)
		deleteName := len(nc.byId) == 0
		if deleteName {
			delete(r.byName, name)
		}
		r.mu.Unlock()
		if r.onCheckDelete != nil {
			r.onCheckDelete(name, id)
		}
		if deleteName && r.onNameDelete != nil {
			r.onNameDelete(name)
		}
	}
	nc.byId[id] = c
	return nil
}

func (r *Registry) initHealthCheck(hc *healthpb.HealthCheck) {
	if hc.GetCreateTime() == nil {
		hc.CreateTime = timestamppb.Now()
	}
}

// Checks provides a mechanism to create managed health checks against named devices.
//
// For each factory method:
//   - [healthpb.HealthCheck.Id] must be unique per name per [Checks] instance, or [ErrAlreadyExists] is returned. An absent id is equivalent to the empty string.
//   - The [healthpb.HealthCheck] instance must not be modified after the call.
//   - [ErrInvalid] is returned if the passed [healthpb.HealthCheck] is not valid for the type of check being created.
type Checks struct {
	r     *Registry
	owner string
}

// NewBoundsCheck creates a new health check that updates normality by comparing a value against bounds.
// The returned BoundsCheck takes ownership of c, populating any missing fields as necessary.
func (hc *Checks) NewBoundsCheck(name string, c *healthpb.HealthCheck) (*BoundsCheck, error) {
	hc.adjustId(c)
	check, err := newBoundsCheck(c)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if err := hc.r.addCheck(name, c.Id, check.checkBase); err != nil {
		return nil, err
	}
	return check, nil
}

// NewFaultCheck creates a new health check that tracks normality via one or more faults.
// The returned FaultCheck takes ownership of c, populating any missing fields as necessary.
func (hc *Checks) NewFaultCheck(name string, c *healthpb.HealthCheck) (*FaultCheck, error) {
	hc.adjustId(c)
	check, err := newFaultCheck(c)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if err := hc.r.addCheck(name, c.Id, check.checkBase); err != nil {
		return nil, err
	}
	return check, nil
}

func (hc *Checks) adjustId(c *healthpb.HealthCheck) {
	c.Id = AbsID(hc.owner, c.Id)
}

// AbsID returns the absolute ID for a check owned by owner with the given checkID.
// The AbsID will be present on health checks received via Registry callbacks.
func AbsID(owner, checkID string) string {
	if checkID == "" {
		return owner
	}
	return owner + ":" + checkID
}

type namedChecks struct {
	n    string
	byId map[string]*checkBase
}
