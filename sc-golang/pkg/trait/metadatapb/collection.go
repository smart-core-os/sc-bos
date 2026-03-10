package metadatapb

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Collection struct {
	metadata *resource.Collection
}

func NewCollection(opts ...resource.Option) *Collection {
	collection := resource.NewCollection(opts...)
	return &Collection{
		metadata: collection,
	}
}

func (m *Collection) GetMetadata(name string, opts ...resource.ReadOption) (*traits.Metadata, error) {
	res, ok := m.metadata.Get(name, opts...)
	if !ok {
		return nil, status.Error(codes.NotFound, "metadata not found")
	}
	return res.(*traits.Metadata), nil
}

func (m *Collection) UpdateMetadata(name string, metadata *traits.Metadata, opts ...resource.WriteOption) (*traits.Metadata, error) {
	res, err := m.metadata.Update(name, metadata, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.Metadata), nil
}

func (m *Collection) DeleteMetadata(name string, opts ...resource.WriteOption) (*traits.Metadata, error) {
	old, err := m.metadata.Delete(name, opts...)
	var oldMetadata *traits.Metadata
	if old != nil {
		oldMetadata = old.(*traits.Metadata)
	}
	return oldMetadata, err
}

// MergeMetadata writes any present fields in metadata to the existing data.
// Traits that exist in the given metadata are merged with existing traits, so that each trait appears only once and
// the 'more' maps are merged.
func (m *Collection) MergeMetadata(name string, metadata *traits.Metadata, opts ...resource.WriteOption) (*traits.Metadata, error) {
	newOpts := make([]resource.WriteOption, 1, len(opts)+1)
	newOpts[0] = resource.InterceptBefore(metadataMergeInterceptor)
	newOpts = append(newOpts, opts...)
	return m.UpdateMetadata(name, metadata, newOpts...)
}

func (m *Collection) UpdateTraitMetadata(name string, traitMetadata *traits.TraitMetadata, opts ...resource.WriteOption) (*traits.Metadata, error) {
	return m.MergeMetadata(name, &traits.Metadata{Traits: []*traits.TraitMetadata{traitMetadata}}, opts...)
}

func (m *Collection) PullMetadata(ctx context.Context, name string, opts ...resource.ReadOption) <-chan *traits.PullMetadataResponse_Change {
	send := make(chan *traits.PullMetadataResponse_Change)

	// when ctx is cancelled, then the resource will close recv for us
	recv := m.metadata.PullID(ctx, name, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			protoChange := metadataValueChangeToProto(change)
			protoChange.Name = name

			select {
			case <-ctx.Done():
				return
			case send <- protoChange:
			}
		}
	}()

	return send
}

func (m *Collection) PullAllMetadata(ctx context.Context, opts ...resource.ReadOption) <-chan CollectionChange {
	send := make(chan CollectionChange)

	recv := m.metadata.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			select {
			case <-ctx.Done():
				return
			case send <- collectionChangeFromResource(change):
			}
		}
	}()
	return send
}

func (m *Collection) ListMetadata(opts ...resource.ReadOption) []*traits.Metadata {
	protoMsgs := m.metadata.List(opts...)
	metadata := make([]*traits.Metadata, len(protoMsgs))
	for i, protoMsg := range protoMsgs {
		metadata[i] = protoMsg.(*traits.Metadata)
	}
	return metadata
}

type CollectionChange struct {
	Name          string
	ChangeTime    time.Time
	ChangeType    types.ChangeType
	OldValue      *traits.Metadata
	NewValue      *traits.Metadata
	LastSeedValue bool
}

func collectionChangeFromResource(change *resource.CollectionChange) CollectionChange {
	result := CollectionChange{
		Name:          change.Id,
		ChangeTime:    change.ChangeTime,
		ChangeType:    change.ChangeType,
		LastSeedValue: change.LastSeedValue,
	}
	if oldV, ok := change.OldValue.(*traits.Metadata); ok {
		result.OldValue = oldV
	}
	if newV, ok := change.NewValue.(*traits.Metadata); ok {
		result.NewValue = newV
	}
	return result
}
