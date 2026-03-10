package minibus

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestDropExcess(t *testing.T) {
	source := make(chan any)
	consumer := DropExcess(source)

	// we can send a value and it gets buffered...
	sendQuickly(t, source, "hello")
	// ...and we can then receive it
	expectRecv(t, consumer, "hello")
	// ... which leaves the channel empty
	expectEmpty(t, consumer)

	// if we send multiple messages in quick succession...
	sendQuickly(t, source, "foo")
	sendQuickly(t, source, "bar")
	sendQuickly(t, source, "baz")
	// ... then the older ones will be dropped
	expectRecv(t, consumer, "baz")
	expectEmpty(t, consumer)

	// closing the source will close the consumer
	close(source)
	expectClosed(t, consumer)
}

func sendQuickly(t *testing.T, dest chan<- any, value any) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	select {
	case dest <- value:
	case <-timer.C:
		t.Fatal("channel send took longer than expected")
	}
}

func recvQuickly(t *testing.T, src <-chan any) any {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	select {
	case value, ok := <-src:
		if !ok {
			t.Fatal("channel was closed unexpectedly")
		}
		return value
	case <-timer.C:
		t.Fatal("channel send took longer than expected")
		return nil
	}
}

func expectRecv(t *testing.T, src <-chan any, expect any, opts ...cmp.Option) {
	value := recvQuickly(t, src)
	if diff := cmp.Diff(expect, value, opts...); diff != "" {
		t.Errorf("Received unexpected value: %s", diff)
	}
}

func expectEmpty(t *testing.T, src <-chan any) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	select {
	case value, ok := <-src:
		if !ok {
			t.Fatal("expected channel to be empty, but it is closed")
		}
		t.Errorf("expected channel to be empty, but got %v", value)
	case <-timer.C:
	}
}

func expectClosed(t *testing.T, ch <-chan any) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	select {
	case value, ok := <-ch:
		if ok {
			t.Errorf("expected channel to be closed, but got value %v", value)
		}
	case <-timer.C:
		t.Error("expected channel to be closed, but it remains open")
	}
}
