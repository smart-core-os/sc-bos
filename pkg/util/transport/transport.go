package transport

import (
	"context"
	"io"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

type State int

const (
	Idle State = iota
	Connected
	Disconnected
)

// Transport presents a way to communicate with another system over arbitrary protocols such as those built on TCP
// for an example usage, see cmd/tools/tcp-conn-test
type Transport interface {
	io.ReadWriteCloser

	Connect(ctx context.Context) error
	WaitForStateChange(ctx context.Context, sourceState State) (state State, changed bool)
}

// ConnectionState stores a State and provides methods for subscribing to changes to the state
type ConnectionState struct {
	// control access to state & bus
	lock sync.RWMutex
	// the current state of the connection
	state       State
	connected   minibus.Bus[bool]
	stateChange minibus.Bus[State]
}

// NewConnectionState creates a new ConnectionState
func NewConnectionState() *ConnectionState {
	return &ConnectionState{}
}

// WaitForStateChange waits until the State changes from sourceState or ctx expires.
// Changed is true returned in former case and false in latter.
func (c *ConnectionState) WaitForStateChange(ctx context.Context, sourceState State) (state State, changed bool) {
	ctx, cancel := context.WithCancel(ctx)
	currentState, onStateChange := c.stateChanges(ctx)
	defer cancel() // unsubscribe if we're not going to listen
	if currentState != sourceState {
		return currentState, true
	}
	select {
	case <-ctx.Done():
		return sourceState, false
	case e := <-onStateChange:
		return e, true
	}
}

// update will update the stored state, if its changed, and emit ConnectedTopic and StateChangeTopic
// events on the bus where appropriate
func (c *ConnectionState) update(s State) {
	c.lock.RLock()
	if c.state == s {
		c.lock.RUnlock()
		return
	}
	c.lock.RUnlock()
	c.lock.Lock()
	defer c.lock.Unlock() // bus emits must also be in the lock
	c.state = s
	if s == Connected {
		c.connected.Send(context.Background(), true)
	}
	c.stateChange.Send(context.Background(), s)
}

// connectedChanges returns the current value of isConnected, and a channel which will receive events when the
// state changes to Connected.
func (c *ConnectionState) connectedChanges(ctx context.Context) (isConnected bool, onConnected <-chan bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	isConnected = c.state == Connected
	onConnected = c.connected.Listen(ctx) // be sure to subscribe in the lock, so we don't miss an update
	return
}

// stateChanges returns the current value of state, and a channel which will receive events when the
// state changes.
func (c *ConnectionState) stateChanges(ctx context.Context) (state State, onStateChange <-chan State) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	state = c.state
	onStateChange = c.stateChange.Listen(ctx) // be sure to subscribe in the lock, so we don't miss an update
	return
}
