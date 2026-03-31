package minibus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestBus_OneToMany(t *testing.T) {
	var bus Bus[int]

	listenCtx, stopListen := context.WithCancel(context.Background())

	// start the listeners
	const numListeners = 10
	var listenChs []<-chan int
	for range numListeners {
		listenChs = append(listenChs, bus.Listen(listenCtx))
	}

	// one goroutine sends
	go func() {
		defer stopListen()
		for i := range 100 {
			sent := bus.Send(context.Background(), i)
			if sent != numListeners {
				t.Logf("Sent != expected, want %v, got %v", numListeners, sent)
			}
		}
	}()

	// several goroutines should all receive all the elements
	var group sync.WaitGroup
	for listenIndex, listenCh := range listenChs {
		group.Go(func() {
			collected := collector(listenCh)
			if len(collected) != 100 {
				t.Errorf("{%d} expected to collect 100 items but got %d", listenIndex, len(collected))
				return
			}
			for i := range 100 {
				if collected[i] != i {
					t.Errorf("{%d} collected[%d] = %d", listenIndex, i, collected[i])
				}
			}
		})
	}
}

func TestBus_DontWaitForSend(t *testing.T) {
	var bus Bus[string]

	events := bus.Listen(context.Background())
	assertWillBlock(t, events)

	waitForSend := make(chan int, 1)
	go func() {
		waitForSend <- bus.Send(context.Background(), "foo")
	}()
	assertWillBlock(t, waitForSend)

	assertChanVal(t, events, "foo")
	assertChanVal(t, waitForSend, 1)
}

func collector[T any](source <-chan T) (collected []T) {
	for data := range source {
		collected = append(collected, data)
	}
	return
}

func assertWillBlock[T any](t *testing.T, ch <-chan T) {
	select {
	case v := <-ch:
		t.Fatalf("Expecting blocking chan but got value %v", v)
	case <-time.After(50 * time.Millisecond):
	}
}

func assertChanVal[T comparable](t *testing.T, ch <-chan T, v T) {
	select {
	case got, ok := <-ch:
		if !ok {
			t.Fatalf("Channel closed waiting for %v", v)
		}
		if got != v {
			t.Fatalf("Chan value is not as expected, want %v, got %v", v, got)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("Timeout waiting for chan value %v", v)
	}
}
