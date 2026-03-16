package logpb

import (
	"context"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

const defaultBufCap = 1000

// Model stores the in-memory state for the Log trait:
//   - a fixed-size ring buffer of log messages (for PullLogMessages replay),
//   - a resource.Value holding the current LogLevel,
//   - a resource.Value holding the current LogMetadata.
type Model struct {
	// ring buffer
	mu      sync.Mutex
	buf     []*logpb.LogMessage
	bufHead int // next write position
	bufSize int // number of valid messages (0 <= bufSize <= len(buf))

	subsMu sync.Mutex
	subs   []chan []*logpb.LogMessage

	level    *resource.Value // of *logpb.LogLevel
	metadata *resource.Value // of *logpb.LogMetadata

	// OnUpdateLogLevel, if non-nil, is called after every successful UpdateLogLevel.
	// The system plugin uses this to propagate level changes to zap.AtomicLevel.
	OnUpdateLogLevel func(lvl *logpb.LogLevel)
}

// NewModel creates a Model with an in-memory ring buffer of capacity bufCap.
// If bufCap <= 0 the default capacity (1000) is used.
func NewModel(bufCap int) *Model {
	if bufCap <= 0 {
		bufCap = defaultBufCap
	}
	return &Model{
		buf: make([]*logpb.LogMessage, bufCap),
		level: resource.NewValue(resource.WithInitialValue(&logpb.LogLevel{
			Level: logpb.Level_INFO,
		})),
		metadata: resource.NewValue(resource.WithInitialValue(&logpb.LogMetadata{})),
	}
}

// AppendMessage adds msg to the ring buffer and notifies all subscribers.
// This method is safe to call from multiple goroutines concurrently.
func (m *Model) AppendMessage(msg *logpb.LogMessage) {
	m.mu.Lock()
	m.buf[m.bufHead] = msg
	m.bufHead = (m.bufHead + 1) % len(m.buf)
	if m.bufSize < len(m.buf) {
		m.bufSize++
	}
	m.mu.Unlock()

	m.subsMu.Lock()
	subs := make([]chan []*logpb.LogMessage, len(m.subs))
	copy(subs, m.subs)
	m.subsMu.Unlock()

	batch := []*logpb.LogMessage{msg}
	for _, ch := range subs {
		select {
		case ch <- batch:
		default:
			// slow subscriber: drop rather than block
		}
	}
}

// TailMessages returns the most recent n messages in chronological order.
// If n exceeds the number of buffered messages, all buffered messages are returned.
func (m *Model) TailMessages(n int) []*logpb.LogMessage {
	if n <= 0 {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if n > m.bufSize {
		n = m.bufSize
	}
	result := make([]*logpb.LogMessage, n)
	cap := len(m.buf)
	start := (m.bufHead - n + cap) % cap
	for i := range n {
		result[i] = m.buf[(start+i)%cap]
	}
	return result
}

// Subscribe registers a subscriber that receives batches of new messages.
// The returned channel is closed and the subscriber removed when cancel is called.
func (m *Model) Subscribe() (ch <-chan []*logpb.LogMessage, cancel func()) {
	inner := make(chan []*logpb.LogMessage, 32)
	m.subsMu.Lock()
	m.subs = append(m.subs, inner)
	m.subsMu.Unlock()
	return inner, func() {
		m.subsMu.Lock()
		defer m.subsMu.Unlock()
		for i, s := range m.subs {
			if s == inner {
				m.subs = append(m.subs[:i], m.subs[i+1:]...)
				close(inner)
				return
			}
		}
	}
}

// GetLogLevel returns the current log level.
func (m *Model) GetLogLevel(opts ...resource.ReadOption) (*logpb.LogLevel, error) {
	return m.level.Get(opts...).(*logpb.LogLevel), nil
}

// UpdateLogLevel sets the log level and notifies subscribers.
// If OnUpdateLogLevel is set it is called with the new level.
func (m *Model) UpdateLogLevel(lvl *logpb.LogLevel, opts ...resource.WriteOption) (*logpb.LogLevel, error) {
	res, err := m.level.Set(lvl, opts...)
	if err != nil {
		return nil, err
	}
	updated := res.(*logpb.LogLevel)
	if m.OnUpdateLogLevel != nil {
		m.OnUpdateLogLevel(updated)
	}
	return updated, nil
}

// PullLogLevel streams changes to the log level.
func (m *Model) PullLogLevel(ctx context.Context, opts ...resource.ReadOption) <-chan PullLogLevelChange {
	return resources.PullValue[*logpb.LogLevel](ctx, m.level.Pull(ctx, opts...))
}

// PullLogLevelChange is the type of values emitted by PullLogLevel.
type PullLogLevelChange = resources.ValueChange[*logpb.LogLevel]

// GetLogMetadata returns the current log metadata.
func (m *Model) GetLogMetadata(opts ...resource.ReadOption) (*logpb.LogMetadata, error) {
	return m.metadata.Get(opts...).(*logpb.LogMetadata), nil
}

// UpdateLogMetadata sets the log metadata and notifies subscribers.
func (m *Model) UpdateLogMetadata(md *logpb.LogMetadata, opts ...resource.WriteOption) (*logpb.LogMetadata, error) {
	res, err := m.metadata.Set(md, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*logpb.LogMetadata), nil
}

// PullLogMetadata streams changes to the log metadata.
func (m *Model) PullLogMetadata(ctx context.Context, opts ...resource.ReadOption) <-chan PullLogMetadataChange {
	return resources.PullValue[*logpb.LogMetadata](ctx, m.metadata.Pull(ctx, opts...))
}

// PullLogMetadataChange is the type of values emitted by PullLogMetadata.
type PullLogMetadataChange = resources.ValueChange[*logpb.LogMetadata]
