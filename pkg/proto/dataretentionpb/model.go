package dataretentionpb

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

// Model stores a single DataRetention value.
type Model struct {
	value *resource.Value // of *DataRetention

	subsMu  sync.Mutex
	subs    atomic.Int32
	hasSubs chan struct{} // closed when subs transitions 0→1; replaced on each transition
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&DataRetention{})}
	return &Model{
		value:   resource.NewValue(append(defaultOptions, opts...)...),
		hasSubs: make(chan struct{}),
	}
}

func (m *Model) GetDataRetention(opts ...resource.ReadOption) (*DataRetention, error) {
	return m.value.Get(opts...).(*DataRetention), nil
}

func (m *Model) SetDataRetention(v *DataRetention, opts ...resource.WriteOption) (*DataRetention, error) {
	res, err := m.value.Set(v, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*DataRetention), nil
}

// PullDataRetention subscribes to changes. The polling goroutine only runs while
// there is at least one active subscriber, so subscribe/unsubscribe transitions
// are tracked here.
func (m *Model) PullDataRetention(ctx context.Context, opts ...resource.ReadOption) <-chan PullDataRetentionChange {
	m.addSub()
	ch := resources.PullValue[*DataRetention](ctx, m.value.Pull(ctx, opts...))
	go func() {
		<-ctx.Done()
		m.removeSub()
	}()
	return ch
}

// HasSubscribers reports whether there are any active PullDataRetention subscribers.
func (m *Model) HasSubscribers() bool {
	return m.subs.Load() > 0
}

// WaitForSubscriber returns a channel that is closed when the first subscriber
// connects. If there is already at least one subscriber, the returned channel
// is already closed so the caller unblocks immediately.
func (m *Model) WaitForSubscriber() <-chan struct{} {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()
	if m.subs.Load() > 0 {
		// Pre-closed channel so the caller's select unblocks immediately.
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return m.hasSubs
}

func (m *Model) addSub() {
	m.subsMu.Lock()
	if m.subs.Add(1) == 1 {
		close(m.hasSubs)
		m.hasSubs = make(chan struct{})
	}
	m.subsMu.Unlock()
}

func (m *Model) removeSub() {
	m.subs.Add(-1)
}

type PullDataRetentionChange = resources.ValueChange[*DataRetention]
