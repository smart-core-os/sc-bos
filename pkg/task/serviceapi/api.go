// Package serviceapi implements servicespb.ServicesApiServer backed by a service.Map.
package serviceapi

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/page"
	"github.com/smart-core-os/sc-golang/pkg/masks"
)

// Api implements a servicespb.ServicesApiServer backed by service.Map.
type Api struct {
	servicespb.UnimplementedServicesApiServer
	m   *service.Map
	now func() time.Time

	knownTypes []string
	store      Store
	logger     *zap.Logger
}

func NewApi(m *service.Map, opts ...Option) *Api {
	a := &Api{m: m, now: time.Now}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Api) GetService(_ context.Context, request *servicespb.GetServiceRequest) (*servicespb.Service, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id missing")
	}

	r := a.m.Get(request.Id)
	if r == nil {
		return nil, status.Error(codes.NotFound, "id not found")
	}

	p := recordToProto(r)
	masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask)).Filter(p)
	return p, nil
}

func (a *Api) PullService(request *servicespb.PullServiceRequest, server servicespb.ServicesApi_PullServiceServer) error {
	if request.Id == "" {
		return status.Error(codes.InvalidArgument, "id missing")
	}

	r := a.m.Get(request.Id)
	if r == nil {
		return status.Error(codes.NotFound, "id not found")
	}

	ctx, stop := context.WithCancel(server.Context())
	defer stop()

	// we watch the map for changes because we want to stop listening to the service if it's not in the map anymore
	mapChanges := a.m.Listen(ctx)

	var serviceChanges <-chan service.State
	if request.UpdatesOnly {
		serviceChanges = r.Service.StateChanges(ctx)
	} else {
		var state service.State
		state, serviceChanges = r.Service.StateAndChanges(ctx)
		change := stateToPullServiceResponse(request.Name, r.Id, r.Kind, state)
		if err := server.Send(change); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case c := <-mapChanges:
			if c.NewValue == nil && c.OldValue != nil && c.OldValue.Id == request.Id {
				// the service was removed
				return nil
			}
		case c := <-serviceChanges:
			change := stateToPullServiceResponse(request.Name, r.Id, r.Kind, c)
			if err := server.Send(change); err != nil {
				return err
			}
		}
	}
}

func (a *Api) CreateService(ctx context.Context, request *servicespb.CreateServiceRequest) (*servicespb.Service, error) {
	id, kind, state := protoToState(request.Service)
	id, state, err := a.m.Create(id, kind, state)
	if err != nil {
		return nil, err
	}

	if err := a.storeConfig(ctx, id, kind, state.Config); err != nil {
		// todo: revert the update
		return nil, err
	}
	return stateToProto(id, kind, state), nil
}

func (a *Api) DeleteService(ctx context.Context, request *servicespb.DeleteServiceRequest) (*servicespb.Service, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id missing")
	}

	state, err := a.m.Delete(request.Id)
	if errors.Is(err, service.ErrNotFound) && request.AllowMissing {
		err = nil
	}
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "id not found")
		}
		return nil, err
	}
	return stateToProto(request.Id, "", state), nil
}

func (a *Api) ListServices(_ context.Context, request *servicespb.ListServicesRequest) (*servicespb.ListServicesResponse, error) {
	idFunc := func(r *service.Record) string { return r.Id }
	values, totalSize, nextPageToken, err := page.List(request, idFunc, func() []*service.Record {
		return a.m.Values()
	})
	if err != nil {
		return nil, err
	}

	res := &servicespb.ListServicesResponse{
		Services:      make([]*servicespb.Service, len(values)),
		TotalSize:     int32(totalSize),
		NextPageToken: nextPageToken,
	}
	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	for i, value := range values {
		res.Services[i] = filter.FilterClone(recordToProto(value)).(*servicespb.Service)
	}

	return res, nil
}

func (a *Api) PullServices(request *servicespb.PullServicesRequest, server servicespb.ServicesApi_PullServicesServer) error {
	ctx, stop := context.WithCancel(server.Context())
	defer stop()

	for c := range a.pullServices(ctx, request) {
		change := &servicespb.PullServicesResponse{Changes: []*servicespb.PullServicesResponse_Change{c}}
		if err := server.Send(change); err != nil {
			return err
		}
	}

	return nil
}

func (a *Api) pullServices(ctx context.Context, request *servicespb.PullServicesRequest) <-chan *servicespb.PullServicesResponse_Change {
	out := make(chan *servicespb.PullServicesResponse_Change)
	// protects out to make sure that the watchRecord goroutines can never get hold of
	// out after it's closed.
	outSem := make(chan chan *servicespb.PullServicesResponse_Change, 1)
	outSem <- out

	ctx, stop := context.WithCancel(ctx)

	// publish sends change to out unless ctx is done.
	// Returns true if sending to out won, false if ctx is done.
	publish := func(change *servicespb.PullServicesResponse_Change) bool {
		// it's not valid to send on a channel that might be closed, even if another select case
		// (sucn as <-ctx.Done()) is already complete.
		// Therefore we protect out with outSem, so that publish can only get hold of the channel
		// when it isn't closed yet. After outSem is closed, it will never be sent back to outSem.
		var out chan *servicespb.PullServicesResponse_Change
		select {
		case <-ctx.Done():
			return false
		case out = <-outSem:
		}
		defer func() {
			outSem <- out
		}()

		select {
		case <-ctx.Done():
			return false
		case out <- change:
			return true
		}
	}
	// watchRecord listens to changes in records service and publishes to out until ctx is done.
	// Calling stop should cancel ctx.
	watchRecord := func(ctx context.Context, stop context.CancelFunc, record *service.Record, updateOnly bool) {
		defer stop() // we shouldn't need this, ctx cancellation is the only way to exit this func anyway

		var last *servicespb.Service // used for updates as OldValue

		var serviceChanges <-chan service.State
		if updateOnly {
			serviceChanges = record.Service.StateChanges(ctx)
		} else {
			var state service.State
			state, serviceChanges = record.Service.StateAndChanges(ctx)
			last = stateToProto(record.Id, record.Kind, state)
			change := &servicespb.PullServicesResponse_Change{
				Name:       request.Name,
				ChangeTime: timestamppb.New(a.now()),
				Type:       types.ChangeType_ADD,
				NewValue:   last,
			}
			if !publish(change) {
				return
			}
		}

		for state := range serviceChanges {
			old := last
			last = stateToProto(record.Id, record.Kind, state)
			change := &servicespb.PullServicesResponse_Change{
				Name:       request.Name,
				ChangeTime: timestamppb.New(a.now()),
				Type:       types.ChangeType_UPDATE,
				OldValue:   old,
				NewValue:   last,
			}
			if !publish(change) {
				return
			}
		}
	}

	listeners := make(map[string]context.CancelFunc)

	changes := a.m.Listen(ctx) // do this before getting the map values

	for _, record := range a.m.Values() {
		ctx, stop := context.WithCancel(ctx)
		listeners[record.Id] = stop
		go watchRecord(ctx, stop, record, request.UpdatesOnly)
	}

	go func() {
		defer func() {
			// close out and don't send it back to outSem, so other goroutines don't try to send to it
			close(<-outSem)
		}()
		defer stop()
		for {
			select {
			case <-ctx.Done():
				return
			case c, ok := <-changes:
				if !ok {
					return // changes closed
				}
				if c.OldValue != nil && c.NewValue == nil {
					// remove
					if stop, ok := listeners[c.OldValue.Id]; ok {
						delete(listeners, c.OldValue.Id)
						stop()
						change := &servicespb.PullServicesResponse_Change{
							Name:       request.Name,
							ChangeTime: timestamppb.New(c.ChangeTime),
							Type:       types.ChangeType_REMOVE,
							OldValue:   recordToProto(c.OldValue),
						}
						if !publish(change) {
							return
						}
					}
				} else if c.OldValue == nil && c.NewValue != nil {
					// add
					ctx, stop := context.WithCancel(ctx)
					listeners[c.NewValue.Id] = stop
					// false here forces watchRecord to publish the ADD event
					go watchRecord(ctx, stop, c.NewValue, false)
				}
			}
		}
	}()

	return out
}

func (a *Api) StartService(_ context.Context, request *servicespb.StartServiceRequest) (*servicespb.Service, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id missing")
	}

	r := a.m.Get(request.Id)
	if r == nil {
		return nil, status.Error(codes.NotFound, "id not found")
	}

	state, err := r.Service.Start()
	if errors.Is(err, service.ErrAlreadyStarted) {
		if !request.AllowActive {
			return nil, status.Error(codes.FailedPrecondition, "already started")
		}
		err = nil // clear the error
	}
	if err != nil {
		return nil, err
	}

	// todo: starting/stopping a service should be saved somewhere. Unfortunately it isn't.

	return stateToProto(r.Id, r.Kind, state), nil
}

func (a *Api) ConfigureService(ctx context.Context, request *servicespb.ConfigureServiceRequest) (*servicespb.Service, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id missing")
	}
	// todo: should we warn if attempting to write config that is empty?

	r := a.m.Get(request.Id)
	if r == nil {
		return nil, status.Error(codes.NotFound, "id not found")
	}

	state, err := r.Service.Configure([]byte(request.ConfigRaw))
	if err != nil {
		return nil, err
	}

	// note, during update type is determined based on the existing type
	if err := a.storeConfig(ctx, request.Id, "", state.Config); err != nil {
		// todo: revert the update
		return nil, err
	}

	return stateToProto(r.Id, r.Kind, state), nil
}

func (a *Api) StopService(_ context.Context, request *servicespb.StopServiceRequest) (*servicespb.Service, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id missing")
	}

	r := a.m.Get(request.Id)
	if r == nil {
		return nil, status.Error(codes.NotFound, "id not found")
	}

	state, err := r.Service.Stop()
	if errors.Is(err, service.ErrAlreadyStopped) {
		if !request.AllowInactive {
			return nil, status.Error(codes.FailedPrecondition, "already stopped")
		}
		err = nil // clear the error
	}
	if err != nil {
		return nil, err
	}
	// todo: starting/stopping a service should be saved somewhere. Unfortunately it isn't.
	return stateToProto(r.Id, r.Kind, state), nil
}

func (a *Api) GetServiceMetadata(_ context.Context, request *servicespb.GetServiceMetadataRequest) (*servicespb.ServiceMetadata, error) {
	md := a.newMetadata()
	a.seedMetadata(md, a.m.States())

	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	filter.Filter(md)

	return md, nil
}

func (a *Api) PullServiceMetadata(request *servicespb.PullServiceMetadataRequest, server servicespb.ServicesApi_PullServiceMetadataServer) error {
	md := a.newMetadata()

	ctx, stop := context.WithCancel(server.Context())
	defer stop()

	current, changes := a.m.GetAndListenState(ctx)
	a.seedMetadata(md, current)

	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

	var lastSent *servicespb.ServiceMetadata
	send := func(md *servicespb.ServiceMetadata) error {
		md = proto.Clone(md).(*servicespb.ServiceMetadata)
		filter.Filter(md)
		if proto.Equal(md, lastSent) {
			return nil // no change, don't send
		}
		lastSent = md
		change := &servicespb.PullServiceMetadataResponse_Change{
			Name:       request.Name,
			Metadata:   md,
			ChangeTime: timestamppb.Now(),
		}
		response := &servicespb.PullServiceMetadataResponse{Changes: []*servicespb.PullServiceMetadataResponse_Change{change}}
		return server.Send(response)
	}

	if !request.UpdatesOnly {
		if err := send(md); err != nil {
			return err
		}
	}

	for change := range changes {
		updateRecord(md, change.OldValue, change.NewValue)
		if err := send(md); err != nil {
			return err
		}
	}
	return nil
}

func (a *Api) newMetadata() *servicespb.ServiceMetadata {
	md := &servicespb.ServiceMetadata{
		TypeCounts: make(map[string]uint32),
	}
	for _, knownType := range a.knownTypes {
		md.TypeCounts[knownType] = 0
	}
	return md
}

func (a *Api) seedMetadata(md *servicespb.ServiceMetadata, states []*service.StateRecord) {
	for _, record := range states {
		incRecord(md, record, 1)
	}
}

func incRecord(md *servicespb.ServiceMetadata, record *service.StateRecord, inc int) {
	addIntP(&md.TotalCount, inc)
	md.TypeCounts[record.Kind] = addInt(md.TypeCounts[record.Kind], inc)
	s := record.State
	if s.Active {
		addIntP(&md.TotalActiveCount, inc)
	}
	if !s.Active && s.Err != nil {
		addIntP(&md.TotalErrorCount, inc)
	}
}

func updateRecord(md *servicespb.ServiceMetadata, oldVal, newVal *service.StateRecord) {
	// this does a little more than strictly needed, but works and is simple
	if oldVal != nil {
		incRecord(md, oldVal, -1)
	}
	if newVal != nil {
		incRecord(md, newVal, 1)
	}
}

func addIntP[N constraints.Integer](a *N, b int) {
	*a = addInt(*a, b)
}

func addInt[N constraints.Integer](a N, b int) N {
	if b < 0 {
		a2 := a - N(-b)
		// check for underflow
		if a2 > a {
			return a
		}
		return a2
	} else {
		a2 := a + N(b)
		// check for overflow
		if a2 < a {
			return a
		}
		return a2
	}
}

func (a *Api) storeConfig(ctx context.Context, name, typ string, data []byte) error {
	if a.store == nil {
		return nil
	}

	if len(data) == 0 {
		data = []byte("{}")
	}

	if err := a.store.SaveConfig(ctx, name, typ, data); err != nil {
		if a.logger != nil {
			a.logger.Warn("writing config file failed", zap.Error(err))
		}
		// todo: return an error here once we can reliably roll back any changes before the store.
	}
	return nil
}

type Store interface {
	// SaveConfig will persist the configuration for a service.
	// The data must be an encoded JSON object.
	// name identifies the service, and is required.
	// typ is the type of the service. If the store already contains a service with the given name, typ may be empty,
	// in which case the existing type is used. If the store does not contain a service with the given name, typ must
	// be non-empty.
	SaveConfig(ctx context.Context, name, typ string, data []byte) error
}
