package resourceutilisationpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/resourceutilisationpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model stores a single ResourceUtilisation value.
type Model struct {
	value *resource.Value // of *resourceutilisationpb.ResourceUtilisation
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&resourceutilisationpb.ResourceUtilisation{})}
	return &Model{
		value: resource.NewValue(append(defaultOptions, opts...)...),
	}
}

func (m *Model) GetResourceUtilisation(opts ...resource.ReadOption) (*resourceutilisationpb.ResourceUtilisation, error) {
	return m.value.Get(opts...).(*resourceutilisationpb.ResourceUtilisation), nil
}

func (m *Model) SetResourceUtilisation(v *resourceutilisationpb.ResourceUtilisation, opts ...resource.WriteOption) (*resourceutilisationpb.ResourceUtilisation, error) {
	res, err := m.value.Set(v, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*resourceutilisationpb.ResourceUtilisation), nil
}

func (m *Model) PullResourceUtilisation(ctx context.Context, opts ...resource.ReadOption) <-chan PullResourceUtilisationChange {
	return resources.PullValue[*resourceutilisationpb.ResourceUtilisation](ctx, m.value.Pull(ctx, opts...))
}

type PullResourceUtilisationChange = resources.ValueChange[*resourceutilisationpb.ResourceUtilisation]
