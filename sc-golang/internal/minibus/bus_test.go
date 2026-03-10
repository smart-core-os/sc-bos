package minibus

import (
	"context"
	"sync"
	"testing"
)

func TestBus_OneToMany(t *testing.T) {
	var bus Bus

	listenCtx, stopListen := context.WithCancel(context.Background())

	// start the listeners
	const numListeners = 10
	var listenChs []<-chan any
	for i := 0; i < numListeners; i++ {
		listenChs = append(listenChs, bus.Listen(listenCtx))
	}

	// one goroutine sends
	go func() {
		defer stopListen()
		for i := 0; i < 100; i++ {
			bus.Send(context.Background(), i)
		}
	}()

	// several goroutines should all receive all the elements
	var group sync.WaitGroup
	for listenIndex, listenCh := range listenChs {
		listenIndex, listenCh := listenIndex, listenCh
		group.Add(1)
		go func() {
			defer group.Done()
			collected := collector(listenCh)
			if len(collected) != 100 {
				t.Errorf("{%d} expected to collect 100 items but got %d", listenIndex, len(collected))
				return
			}
			for i := 0; i < 100; i++ {
				if collected[i].(int) != i {
					t.Errorf("{%d} collected[%d] = %d", listenIndex, i, collected[i])
				}
			}
		}()
	}
}

func collector(source <-chan any) (collected []any) {
	for data := range source {
		collected = append(collected, data)
	}
	return
}
