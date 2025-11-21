package healthpb

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/util/resources"
)

const TraitName trait.Name = "smartcore.bos.Health"

// Model stores health checks against a single entity.
type Model struct {
	checks *resource.Collection // of *gen.HealthCheck, keyed by id
}

func NewModel(opts ...resource.Option) *Model {
	return &Model{
		checks: resource.NewCollection(opts...),
	}
}

func (m *Model) GetHealthCheck(id string, opts ...resource.ReadOption) (*gen.HealthCheck, error) {
	res, ok := m.checks.Get(id, opts...)
	if !ok {
		return nil, status.Error(codes.NotFound, id)
	}
	return res.(*gen.HealthCheck), nil
}

func (m *Model) CreateHealthCheck(check *gen.HealthCheck, opts ...resource.WriteOption) (*gen.HealthCheck, error) {
	res, err := m.checks.Add(check.Id, check, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*gen.HealthCheck), nil
}

func (m *Model) UpdateHealthCheck(check *gen.HealthCheck, opts ...resource.WriteOption) (*gen.HealthCheck, error) {
	if check.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	opts = append([]resource.WriteOption{resource.WithMerger(healthCheckMerge)}, opts...)
	res, err := m.checks.Update(check.Id, check, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*gen.HealthCheck), nil
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

func (m *Model) PullHealthCheck(ctx context.Context, id string, opts ...resource.ReadOption) <-chan resources.ValueChange[*gen.HealthCheck] {
	return resources.PullValue[*gen.HealthCheck](ctx, m.checks.PullID(ctx, id, opts...))
}

func (m *Model) ListHealthChecks(opts ...resource.ReadOption) []*gen.HealthCheck {
	list := m.checks.List(opts...)
	res := make([]*gen.HealthCheck, len(list))
	for i, item := range list {
		res[i] = item.(*gen.HealthCheck)
	}
	return res
}

func (m *Model) PullHealthChecks(ctx context.Context, opts ...resource.ReadOption) <-chan resources.CollectionChange[*gen.HealthCheck] {
	return resources.PullCollection[*gen.HealthCheck](ctx, m.checks.Pull(ctx, opts...))
}

func healthCheckMerge(mask *masks.FieldUpdater, dst, src proto.Message) {
	srcVal := src.(*gen.HealthCheck)
	dstVal := dst.(*gen.HealthCheck)
	MergeCheck(mask.Merge, dstVal, srcVal)
}
