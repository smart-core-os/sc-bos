package transport

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/sync/errgroup"
)

var WriteTimeoutError = errors.New("write timed out waiting for connection")

// Connection represents a connection to another system, and implements Transport, keeping a ConnectionState
// for handling reconnects, etc...
type Connection struct {
	conf  ConnectionConfig
	bo    backoff.BackOff
	state *ConnectionState

	dial func() error
	rwc  io.ReadWriteCloser
}

func NewConnection(conf ConnectionConfig, bo backoff.BackOff, dial func() error, rwc io.ReadWriteCloser) *Connection {
	conf.defaults()
	return &Connection{
		bo:    bo,
		conf:  conf,
		state: NewConnectionState(),
		dial:  dial,
		rwc:   rwc,
	}
}

func (c *Connection) Read(p []byte) (n int, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	isConnected, onConnected := c.state.connectedChanges(ctx)

	if !isConnected {
		<-onConnected // wait for connection
	}
	cancel()

	n, err = c.rwc.Read(p)
	if err != nil {
		c.state.update(Disconnected)
		if err == io.EOF {
			err = nil
		}
	}
	return
}

func (c *Connection) Write(p []byte) (n int, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	isConnected, onConnected := c.state.connectedChanges(ctx)

	if !isConnected {
		timer := time.NewTimer(c.conf.WriteTimeout.Duration)

		select {
		case <-timer.C: // timeout
			cancel()
			return 0, WriteTimeoutError
		case <-onConnected: // wait for connection
		}
	}
	cancel()

	n, err = c.rwc.Write(p)
	if err != nil {
		c.state.update(Disconnected)
	}
	return n, err
}

func (c *Connection) Close() error {
	return c.rwc.Close()
}

func (c *Connection) Connect(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return c.watchConnection(ctx)
	})
	return group.Wait()
}

func (c *Connection) WaitForStateChange(ctx context.Context, sourceState State) (state State, changed bool) {
	return c.state.WaitForStateChange(ctx, sourceState)
}

func (c *Connection) watchConnection(ctx context.Context) error {
	reconnect := func() {
		c.state.update(Disconnected)

		err := c.dial()
		if err != nil {
			c.state.update(Disconnected)
		} else {
			c.state.update(Connected)
			c.bo.Reset() // we connected, so reset backoff
		}
	}
	reconnect()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// wait for backoff
		b := c.bo.NextBackOff()
		time.Sleep(b)

		ctx, cancelOnStateChange := context.WithCancel(ctx)
		currentState, onStateChange := c.state.stateChanges(ctx)
		if currentState == Connected {
			// wait for an error, then re-dial
			<-onStateChange
			cancelOnStateChange()
			reconnect()
		} else {
			cancelOnStateChange()
			reconnect()
		}
	}
}
