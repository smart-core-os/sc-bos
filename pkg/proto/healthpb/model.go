package healthpb

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

const TraitName trait.Name = "smartcore.bos.Health"

// Model stores health checks against a single entity.
type Model struct {
	checks *resource.Collection // of *healthpb.HealthCheck, keyed by id
}

func NewModel(opts ...resource.Option) *Model {
	return &Model{
		checks: resource.NewCollection(opts...),
	}
}

func (m *Model) GetHealthCheck(id string, opts ...resource.ReadOption) (*HealthCheck, error) {
	res, ok := m.checks.Get(id, opts...)
	if !ok {
		return nil, status.Error(codes.NotFound, id)
	}
	return res.(*HealthCheck), nil
}

func (m *Model) CreateHealthCheck(check *HealthCheck, opts ...resource.WriteOption) (*HealthCheck, error) {
	res, err := m.checks.Add(check.Id, check, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*HealthCheck), nil
}

func (m *Model) UpdateHealthCheck(check *HealthCheck, opts ...resource.WriteOption) (*HealthCheck, error) {
	if check.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	opts = append([]resource.WriteOption{resource.WithMerger(healthCheckMerge)}, opts...)
	res, err := m.checks.Update(check.Id, check, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*HealthCheck), nil
}

func (m *Model) DeleteHealthCheck(id string, opts ...resource.WriteOption) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	_, err := m.checks.Delete(id, opts...)
	if err != nil {
		return err
	}
	return nil
}

func (m *Model) PullHealthCheck(ctx context.Context, id string, opts ...resource.ReadOption) <-chan resources.ValueChange[*HealthCheck] {
	return resources.PullValue[*HealthCheck](ctx, m.checks.PullID(ctx, id, opts...))
}

func (m *Model) ListHealthChecks(opts ...resource.ReadOption) []*HealthCheck {
	list := m.checks.List(opts...)
	res := make([]*HealthCheck, len(list))
	for i, item := range list {
		res[i] = item.(*HealthCheck)
	}
	return res
}

func (m *Model) PullHealthChecks(ctx context.Context, opts ...resource.ReadOption) <-chan resources.CollectionChange[*HealthCheck] {
	return resources.PullCollection[*HealthCheck](ctx, m.checks.Pull(ctx, opts...))
}

func healthCheckMerge(mask *masks.FieldUpdater, dst, src proto.Message) {
	srcVal := src.(*HealthCheck)
	dstVal := dst.(*HealthCheck)
	MergeCheck(mask.Merge, dstVal, srcVal)
}
