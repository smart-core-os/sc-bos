package bms

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
)

// Actions defines the only side effects the automation can have.
// This is intended to allow easier testing of the business logic, a bit like a DAO would for database access.
type Actions interface {
	UpdateAirTemperature(ctx context.Context, req *traits.UpdateAirTemperatureRequest, ws *WriteState) error
	UpdateModeValues(ctx context.Context, req *traits.UpdateModeValuesRequest, ws *WriteState) error
}

// ClientActions creates a new Actions backed by node.ClientConner clients.
func ClientActions(clients node.ClientConner) Actions {
	conn := clients.ClientConn()
	return &clientActions{
		airTemperatureClient: traits.NewAirTemperatureApiClient(conn),
		modeClient:           traits.NewModeApiClient(conn),
	}
}

type clientActions struct {
	airTemperatureClient traits.AirTemperatureApiClient
	modeClient           traits.ModeApiClient
}

func (a clientActions) UpdateAirTemperature(ctx context.Context, req *traits.UpdateAirTemperatureRequest, ws *WriteState) error {
	got, err := a.airTemperatureClient.UpdateAirTemperature(ctx, req)
	ws.AirTemperatures[req.Name] = Value[*traits.AirTemperature]{
		V:   got,
		At:  ws.Now(),
		Err: err,
	}
	return err
}

func (a clientActions) UpdateModeValues(ctx context.Context, req *traits.UpdateModeValuesRequest, ws *WriteState) error {
	// there's a "bug" in the field mask validation for maps, so we have to use this hack to avoid clearing values
	oldValues, err := a.modeClient.GetModeValues(ctx, &traits.GetModeValuesRequest{Name: req.Name})
	if err != nil {
		ws.Modes[req.Name] = Value[*traits.ModeValues]{
			V:   nil,
			At:  ws.Now(),
			Err: err,
		}
		return err
	}
	for k, v := range oldValues.Values {
		if _, ok := req.ModeValues.Values[k]; !ok {
			req.ModeValues.Values[k] = v
		}
	}

	got, err := a.modeClient.UpdateModeValues(ctx, req)
	ws.Modes[req.Name] = Value[*traits.ModeValues]{
		V:   got,
		At:  ws.Now(),
		Err: err,
	}
	if err != nil {
		return err
	}
	// check the response matches what we're expecting (i.e. did some but not all modes get written)
	var mismatchValues []string
	for key, sendVal := range req.ModeValues.Values {
		if got.Values[key] != sendVal {
			mismatchValues = append(mismatchValues, fmt.Sprintf("%s[%s!=%s]", key, got.Values[key], sendVal))
		}
	}
	if len(mismatchValues) > 0 {
		return UnexpectedResponseError{
			Message: fmt.Sprintf("mode %v", strings.Join(mismatchValues, ",")),
			Got:     got,
			Want:    req.ModeValues,
		}
	}
	return err
}

// CacheWriteAction wraps an Actions and caches successful writes for cacheExpiry.
// Calling action methods with the same arguments will return the cached result and not perform the write so long as the cache is not expired.
func CacheWriteAction(actions Actions, cacheExpiry time.Duration) Actions {
	return cacheWriteActions{Actions: actions, cacheExpiry: cacheExpiry}
}

type cacheWriteActions struct {
	Actions
	cacheExpiry time.Duration
}

func (a cacheWriteActions) UpdateAirTemperature(ctx context.Context, req *traits.UpdateAirTemperatureRequest, ws *WriteState) error {
	if oldWrite, ok := ws.AirTemperatures[req.Name]; ok {
		if cacheValid(oldWrite, ws.Now(), a.cacheExpiry, func(v *traits.AirTemperature) bool {
			return math.Abs(v.GetTemperatureSetPoint().GetValueCelsius()-req.GetState().GetTemperatureSetPoint().GetValueCelsius()) < 0.01
		}) {
			oldWrite.hit()
			ws.AirTemperatures[req.Name] = oldWrite // update the hit count
			return oldWrite.Err
		}
	}
	return a.Actions.UpdateAirTemperature(ctx, req, ws)
}

func (a cacheWriteActions) UpdateModeValues(ctx context.Context, req *traits.UpdateModeValuesRequest, ws *WriteState) error {
	if oldWrite, ok := ws.Modes[req.Name]; ok {
		if cacheValid(oldWrite, ws.Now(), a.cacheExpiry, func(v *traits.ModeValues) bool {
			cv := v.GetValues()
			for rk, rv := range req.GetModeValues().GetValues() {
				if cv[rk] != rv {
					return false
				}
			}
			return true
		}) {
			oldWrite.hit()
			ws.Modes[req.Name] = oldWrite // update the hit count
			return oldWrite.Err
		}
	}
	return a.Actions.UpdateModeValues(ctx, req, ws)
}

func cacheValid[T any](oldWrite Value[T], now time.Time, cacheExpiry time.Duration, f func(v T) bool) bool {
	if !f(oldWrite.V) {
		return false
	}
	if cacheExpiry > 0 && now.Sub(oldWrite.At) > cacheExpiry {
		return false
	}
	// here is where logic that has different cache expiry for error writes
	return true
}

type ActionCounts struct {
	TotalWrites int

	ModeUpdates map[string]map[string][]DeviceName // [key][value]names
	ModeWrites  []string

	AirTemperatureUpdates map[float64][]DeviceName // [setPoint]names
	AirTemperatureWrites  []string
}

func (a ActionCounts) Changes() []string {
	var res []string
	for setPoint, names := range a.AirTemperatureUpdates {
		if l := len(names); l > 1 {
			res = append(res, fmt.Sprintf("%dx setPoint=%.1f", l, setPoint))
		} else {
			res = append(res, fmt.Sprintf("setPoint=%.1f", setPoint))
		}
	}
	for key, values := range a.ModeUpdates {
		for value, names := range values {
			if l := len(names); l > 1 {
				res = append(res, fmt.Sprintf("%dx %s=%s", l, key, value))
			} else {
				res = append(res, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}
	return res
}

// CountActions wraps an Actions and counts the number of invocations.
func CountActions(actions Actions) (Actions, *ActionCounts) {
	impl := &countActions{Actions: actions, ActionCounts: &ActionCounts{}}
	return impl, impl.ActionCounts
}

type countActions struct {
	Actions
	*ActionCounts
}

func (a *countActions) UpdateAirTemperature(ctx context.Context, req *traits.UpdateAirTemperatureRequest, ws *WriteState) error {
	a.TotalWrites++
	a.AirTemperatureWrites = append(a.AirTemperatureWrites, req.Name)
	if a.AirTemperatureUpdates == nil {
		a.AirTemperatureUpdates = make(map[float64][]DeviceName)
	}
	setPoint := req.GetState().GetTemperatureSetPoint().GetValueCelsius()
	a.AirTemperatureUpdates[setPoint] = append(a.AirTemperatureUpdates[setPoint], req.Name)
	return a.Actions.UpdateAirTemperature(ctx, req, ws)
}

func (a *countActions) UpdateModeValues(ctx context.Context, req *traits.UpdateModeValuesRequest, ws *WriteState) error {
	a.TotalWrites++
	a.ModeWrites = append(a.ModeWrites, req.Name)
	if a.ModeUpdates == nil {
		a.ModeUpdates = make(map[string]map[string][]DeviceName)
	}
	for k, v := range req.GetModeValues().GetValues() {
		if _, ok := a.ModeUpdates[k]; !ok {
			a.ModeUpdates[k] = make(map[string][]DeviceName)
		}
		a.ModeUpdates[k][v] = append(a.ModeUpdates[k][v], req.Name)
	}
	return a.Actions.UpdateModeValues(ctx, req, ws)
}

// NilActions is an Actions that has no side effects.
// Actions are still recorded in the WriteState.
type NilActions struct{}

func (_ NilActions) UpdateAirTemperature(_ context.Context, req *traits.UpdateAirTemperatureRequest, ws *WriteState) error {
	ws.AirTemperatures[req.Name] = Value[*traits.AirTemperature]{
		V:  req.State,
		At: ws.Now(),
	}
	return nil
}

func (_ NilActions) UpdateModeValues(_ context.Context, req *traits.UpdateModeValuesRequest, ws *WriteState) error {
	ws.Modes[req.Name] = Value[*traits.ModeValues]{
		V:  req.ModeValues,
		At: ws.Now(),
	}
	return nil
}

// LogActions wraps an Actions and logs all invocations.
func LogActions(actions Actions, logger *zap.Logger) Actions {
	return logActions{Actions: actions, logger: logger}
}

type logActions struct {
	Actions
	logger *zap.Logger
}

func (a logActions) UpdateAirTemperature(ctx context.Context, req *traits.UpdateAirTemperatureRequest, ws *WriteState) error {
	err := a.Actions.UpdateAirTemperature(ctx, req, ws)
	a.logger.Debug("Actions.UpdateAirTemperature",
		zap.String("name", req.Name),
		zap.Float64("setPoint", req.GetState().GetTemperatureSetPoint().GetValueCelsius()),
		zap.Error(err))
	return err
}

func (a logActions) UpdateModeValues(ctx context.Context, req *traits.UpdateModeValuesRequest, ws *WriteState) error {
	err := a.Actions.UpdateModeValues(ctx, req, ws)
	a.logger.Debug("Actions.UpdateModeValues",
		zap.String("name", req.Name),
		zap.Any("values", req.GetModeValues().GetValues()),
		zap.Error(err))
	return err
}
