package minibus

import (
	"context"
	"sync"
)

type Bus struct {
	listenerM sync.RWMutex
	listeners []*listener
}

func (b *Bus) Send(ctx context.Context, event any) (ok bool) {
	// create a copy of the listeners so avoid holding the mutex a long time
	var listeners []*listener
	b.listenerM.RLock()
	for _, l := range b.listeners {
		listeners = append(listeners, l)
	}
	b.listenerM.RUnlock()

	needGc := false

	// send the event to each listener that's not closed
	for _, l := range listeners {
		ok, active := l.send(ctx, event)
		if !ok {
			return false
		}
		if !active {
			// the listen context on this listener has been cancelled, we need to collect the garbage
			needGc = true
		}
	}

	if needGc {
		b.collect()
	}

	return true
}

func (b *Bus) collect() {
	b.listenerM.Lock()
	defer b.listenerM.Unlock()

	var activeListeners []*listener
	for _, l := range b.listeners {
		if l.alive() {
			activeListeners = append(activeListeners, l)
		}
	}

	b.listeners = activeListeners
}

func (b *Bus) Listen(ctx context.Context) <-chan any {
	ch := make(chan any)

	l := &listener{
		ch:  ch,
		ctx: ctx,
	}

	go func() {
		<-ctx.Done()
		l.stop()
	}()

	// store the listener
	b.listenerM.Lock()
	defer b.listenerM.Unlock()
	b.listeners = append(b.listeners, l)

	return ch
}

type listener struct {
	m   sync.RWMutex
	ch  chan any
	ctx context.Context
}

func (l *listener) send(ctx context.Context, event any) (ok bool, active bool) {
	l.m.RLock()
	defer l.m.RUnlock()

	select {
	case <-ctx.Done():
		// send context cancelled
		return false, true

	case <-l.ctx.Done():
		// listen context cancelled
		// this is considered a success even though the message is not sent
		return true, false

	case l.ch <- event:
		// event sent successfully
		return true, true
	}
}

func (l *listener) stop() {
	l.m.Lock()
	defer l.m.Unlock()
	if l.ch != nil {
		close(l.ch)
		l.ch = nil
	}
}

func (l *listener) alive() bool {
	return l.ctx.Err() == nil
}
