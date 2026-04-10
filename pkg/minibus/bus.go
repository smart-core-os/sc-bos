package minibus

import (
	"container/list"
	"context"
	"sync"
)

type Bus[T any] struct {
	listenerM sync.RWMutex
	listeners list.List // use linked list to make edits easier - i.e. removing from the middle
}

// Send sends a message to all active listeners and returns how many listeners accepted the event.
func (b *Bus[T]) Send(ctx context.Context, event T) int {
	sentToOne := make(chan bool)
	sent := b.send(ctx, event, sentToOne)
	var accepted int
	for sent > 0 {
		select {
		case <-ctx.Done():
			return accepted
		case success := <-sentToOne:
			sent--
			if success {
				accepted++
			}

		}
	}
	return accepted
}

// send sends event to each listener, returning how many listeners it sent to.
// Listeners will receive event in their own go routines the completion of each being sent to sentToOne.
// Callers should expect exactly sent messages on sentToOne.
func (b *Bus[T]) send(ctx context.Context, event T, sentToOne chan<- bool) (sent int) {
	b.listenerM.RLock()
	defer b.listenerM.RUnlock()

	size := b.listeners.Len()
	if size == 0 {
		return 0
	}
	for el := b.listeners.Front(); el != nil; el = el.Next() {
		lis := el.Value.(*listener[T])
		go func() {
			// we send in a goroutine to avoid blocking waiting for a receiver to accept the event
			sentToOne <- lis.send(ctx, event)
		}()
	}

	return size
}

func (b *Bus[T]) Listen(ctx context.Context) <-chan T {
	ch := make(chan T)

	l := &listener[T]{
		ch:  ch,
		ctx: ctx,
	}

	b.listenerM.Lock()
	el := b.listeners.PushBack(l)
	b.listenerM.Unlock()

	go func() {
		<-ctx.Done()
		b.listenerM.Lock()
		defer b.listenerM.Unlock()
		b.listeners.Remove(el)
		l.stop()
	}()

	return ch
}

type listener[T any] struct {
	m   sync.RWMutex
	ch  chan T
	ctx context.Context
}

func (l *listener[T]) send(ctx context.Context, event T) (sent bool) {
	l.m.RLock()
	defer l.m.RUnlock()

	select {
	case <-ctx.Done():
		// send context cancelled
		return false

	case <-l.ctx.Done():
		// listen context cancelled
		return false

	case l.ch <- event:
		// event sent successfully
		return true
	}
}

func (l *listener[T]) stop() {
	l.m.Lock()
	defer l.m.Unlock()
	if l.ch != nil {
		close(l.ch)
		l.ch = nil
	}
}
