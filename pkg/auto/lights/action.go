package lights

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
)

// actions defines the only side effects the automation can have.
// This is intended to allow easier testing of the business logic, a bit like a DAO would for database access.
type actions interface {
	// UpdateBrightness sends a LightApiClient.UpdateBrightness request and stores successful result in state.
	UpdateBrightness(ctx context.Context, now time.Time, req *traits.UpdateBrightnessRequest, state *WriteState) error
}

// newClientActions creates an actions backed by node.ClientConner clients.
func newClientActions(clients node.ClientConner) actions {
	conn := clients.ClientConn()
	return &clientActions{
		lightClient: traits.NewLightApiClient(conn),
	}
}

type clientActions struct {
	lightClient traits.LightApiClient
}

func (a *clientActions) UpdateBrightness(ctx context.Context, now time.Time, req *traits.UpdateBrightnessRequest, state *WriteState) error {
	got, err := a.lightClient.UpdateBrightness(ctx, req)
	state.Brightness[req.Name] = Value[*traits.Brightness]{
		V:   got,
		At:  now,
		Err: err,
	}
	return err
}

// newCachedActions caches responses for a duration.
// Invoking actions with significantly different arguments (e.g. different brightness levels) will not use the cache.
func newCachedActions(a actions, expiry time.Duration) actions {
	if expiry < 0 {
		return a // no caching
	}
	return cachedActions{actions: a, expiry: expiry}
}

type cachedActions struct {
	actions
	expiry time.Duration
}

func (a cachedActions) UpdateBrightness(ctx context.Context, now time.Time, req *traits.UpdateBrightnessRequest, state *WriteState) error {
	if old, hasOld := state.Brightness[req.Name]; hasOld {
		if cacheValid(old, now, a.expiry, func(v *traits.Brightness) bool {
			if v == nil {
				return false
			}
			return v.LevelPercent == req.GetBrightness().GetLevelPercent()
		}) {
			old.hit()
			return old.Err
		}
	}
	return a.actions.UpdateBrightness(ctx, now, req, state)
}

// cacheValid returns true when a cache value exists, is in date, and hasn't changed according to eq.
func cacheValid[T any](oldWrite Value[T], now time.Time, cacheExpiry time.Duration, eq func(v T) bool) bool {
	if !eq(oldWrite.V) {
		return false
	}
	if cacheExpiry > 0 && now.Sub(oldWrite.At) > cacheExpiry {
		return false
	}
	// here is where logic that would have different cache expiry for error writes
	return true
}

type actionCounts struct {
	TotalWrites int

	BrightnessUpdates map[float32][]deviceName // [brightness]names
	BrightnessWrites  []deviceName             // which devices have been written to
}

// Changes returns a summary of the changes made by the actions.
func (a actionCounts) Changes() []string {
	var res []string
	for brightness, names := range a.BrightnessUpdates {
		if l := len(names); l > 1 {
			res = append(res, fmt.Sprintf("%dx brightness=%.1f%%", l, brightness))
		} else {
			res = append(res, fmt.Sprintf("brightness=%.1f%%", brightness))
		}
	}
	return res
}

// newCountActions counts all invocations of a and stores them in actionCounts.
func newCountActions(a actions) (actions, *actionCounts) {
	impl := &countActions{actions: a, actionCounts: &actionCounts{}}
	return impl, impl.actionCounts
}

type countActions struct {
	actions
	*actionCounts
}

func (a *countActions) UpdateBrightness(ctx context.Context, now time.Time, req *traits.UpdateBrightnessRequest, state *WriteState) error {
	a.TotalWrites++
	a.BrightnessWrites = append(a.BrightnessWrites, req.Name)
	if a.BrightnessUpdates == nil {
		a.BrightnessUpdates = make(map[float32][]deviceName)
	}
	a.BrightnessUpdates[req.Brightness.LevelPercent] = append(a.BrightnessUpdates[req.Brightness.LevelPercent], req.Name)

	return a.actions.UpdateBrightness(ctx, now, req, state)
}

// nilActions is an actions that has no side effects.
// Actions are still recorded in the WriteState.
type nilActions struct{}

func (nilActions) UpdateBrightness(ctx context.Context, now time.Time, req *traits.UpdateBrightnessRequest, state *WriteState) error {
	state.Brightness[req.Name] = Value[*traits.Brightness]{
		V:  req.Brightness,
		At: now,
	}
	return nil
}

// newLogActions logs to logger each time an action is invoked.
func newLogActions(a actions, logger *zap.Logger) actions {
	return &logActions{actions: a, logger: logger}
}

type logActions struct {
	actions
	logger *zap.Logger
}

func (a *logActions) UpdateBrightness(ctx context.Context, now time.Time, req *traits.UpdateBrightnessRequest, state *WriteState) error {
	err := a.actions.UpdateBrightness(ctx, now, req, state)
	a.logger.Debug("actions.UpdateBrightness",
		zap.String("name", req.Name),
		zap.Float32("level", req.Brightness.LevelPercent),
		zap.Error(err),
	)
	return err
}

// updateBrightnessLevel invokes actions.UpdateBrightness for each name with the given level.
func updateBrightnessLevel(ctx context.Context, now time.Time, state *WriteState, actions actions, level float32, names ...deviceName) error {
	var errs error
	for _, name := range names {
		err := actions.UpdateBrightness(ctx, now, &traits.UpdateBrightnessRequest{
			Name: name,
			Brightness: &traits.Brightness{
				LevelPercent: level,
			},
		}, state)
		errs = multierr.Append(errs, err)
	}
	return errs
}

// refreshBrightnessLevel invokes actions.UpdateBrightness for each name with the last written level.
func refreshBrightnessLevel(ctx context.Context, now time.Time, state *WriteState, actions actions, names ...deviceName) error {
	var errs error
	for _, name := range names {
		val, ok := state.Brightness[name]
		if !ok || val.V == nil {
			continue
		}
		err := actions.UpdateBrightness(ctx, now, &traits.UpdateBrightnessRequest{
			Name:       name,
			Brightness: val.V,
		}, state)
		errs = multierr.Append(errs, err)
	}
	return errs
}
