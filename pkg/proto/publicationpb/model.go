package publicationpb

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

type Model struct {
	publications *resource.Collection // of *traits.Publication
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		publications: resource.NewCollection(args.publicationsOptions...),
	}
}

func (m *Model) CreatePublication(publication *Publication, opts ...resource.WriteOption) (*Publication, error) {
	args := calcWriteArgs(opts...)
	return toPublication(m.publications.Add(publication.Id, publication,
		resource.WithGenIDIfAbsent(), resource.WithIDCallback(func(id string) {
			publication.Id = id
		}),
		m.withComputedProperties(args),
	))
}

func (m *Model) GetPublication(id string, opts ...resource.ReadOption) (*Publication, bool) {
	msg, ok := m.publications.Get(id, opts...)
	if msg == nil {
		return nil, ok
	}
	return msg.(*Publication), ok
}

func (m *Model) UpdatePublication(id string, publication *Publication, opts ...resource.WriteOption) (*Publication, error) {
	args := calcWriteArgs(opts...)
	opts = append([]resource.WriteOption{m.withComputedProperties(args)}, opts...)
	return toPublication(m.publications.Update(id, publication, opts...))
}

func (m *Model) DeletePublication(id string, opts ...resource.WriteOption) (*Publication, error) {
	return toPublication(m.publications.Delete(id, opts...))
}

//goland:noinspection GoNameStartsWithPackageName
type PublicationChange struct {
	ChangeTime time.Time
	Value      *Publication
}

func (m *Model) PullPublication(ctx context.Context, id string, opts ...resource.ReadOption) <-chan PublicationChange {
	send := make(chan PublicationChange)
	go func() {
		defer close(send)
		for change := range m.publications.PullID(ctx, id, opts...) {
			select {
			case <-ctx.Done():
				return
			case send <- PublicationChange{ChangeTime: change.ChangeTime, Value: change.Value.(*Publication)}:
			}
		}
	}()
	return send
}

func (m *Model) ListPublications(opts ...resource.ReadOption) []*Publication {
	msgs := m.publications.List(opts...)
	items := make([]*Publication, len(msgs))
	for i, msg := range msgs {
		items[i] = msg.(*Publication)
	}
	return items
}

type PublicationsChange struct {
	ID         string
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   *Publication
	NewValue   *Publication
}

func (m *Model) PullPublications(ctx context.Context, opts ...resource.ReadOption) <-chan PublicationsChange {
	send := make(chan PublicationsChange)
	go func() {
		defer close(send)
		for change := range m.publications.Pull(ctx, opts...) {
			event := PublicationsChange{
				ID:         change.Id,
				ChangeTime: change.ChangeTime,
				ChangeType: change.ChangeType,
			}
			if change.OldValue != nil {
				event.OldValue = change.OldValue.(*Publication)
			}
			if change.NewValue != nil {
				event.NewValue = change.NewValue.(*Publication)
			}

			select {
			case <-ctx.Done():
				return
			case send <- event:
			}
		}
	}()
	return send
}

func (m *Model) withComputedProperties(args writeArgs) resource.WriteOption {
	return resource.InterceptAfter(func(old, new proto.Message) {
		newVal := new.(*Publication)
		if args.resetReceipt {
			if newVal.Audience != nil {
				newVal.Audience.ReceiptTime = nil
				newVal.Audience.Receipt = Publication_Audience_NO_SIGNAL
				newVal.Audience.ReceiptRejectedReason = ""
			}
		}
		if args.newPublishTime {
			newVal.PublishTime = timestamppb.New(m.publications.Clock().Now())
		}
		if args.newVersion {
			newVal.Version = mintVersion(newVal)
		}
	})
}

func toPublication(msg proto.Message, err error) (*Publication, error) {
	if msg == nil {
		return nil, err
	}
	return msg.(*Publication), err
}

func mintVersion(p *Publication) string {
	// Hash properties we are happy to include in our version calculation.
	// Note: don't use proto.Marshal as it is non-deterministic
	hash := md5.New()
	io.WriteString(hash, "v1")
	io.WriteString(hash, p.Id)
	hash.Write(p.Body)
	io.WriteString(hash, p.MediaType)
	io.WriteString(hash, p.GetAudience().GetName())
	return fmt.Sprintf("%x", hash.Sum(nil))
}
