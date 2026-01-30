package rx

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	scslices "github.com/smart-core-os/sc-bos/pkg/util/slices"
)

func TestSet_Sub(t *testing.T) {
	s := NewSet(scslices.NewSorted("001", "002", "003"))
	feed := make(chan string, 10)
	go func() {
		for v := range feed {
			s.Set(v)
		}
	}()

	now, events := s.Sub(t.Context())
	var gotNow []string
	for _, v := range now.All {
		gotNow = append(gotNow, v)
	}
	if diff := cmp.Diff([]string{"001", "002", "003"}, gotNow); diff != "" {
		t.Errorf("Set.Sub() now mismatch (-want +got):\n%s", diff)
	}

	feed <- "004"
	got, err := chans.RecvWithin(events, 5*time.Second)
	if err != nil {
		t.Fatalf("Set.Sub() events receive failed: %v", err)
	}
	if diff := cmp.Diff(Change[string]{Type: Add, New: "004"}, got); diff != "" {
		t.Errorf("Set.Sub() event mismatch (-want +got):\n%s", diff)
	}

	feed <- "002"
	got, err = chans.RecvWithin(events, 5*time.Second)
	if err != nil {
		t.Fatalf("Set.Sub() events receive failed: %v", err)
	}
	if diff := cmp.Diff(Change[string]{Type: Update, Old: "002", New: "002"}, got); diff != "" {
		t.Errorf("Set.Sub() event mismatch (-want +got):\n%s", diff)
	}
}

func TestSet_concurrency(t *testing.T) {
	s := NewSet(scslices.NewSorted("pre000", "pre001", "pre002"))

	var writers sync.WaitGroup
	// this routine adds items to the set
	writers.Add(1)
	go func() {
		defer writers.Done() // go routine finished
		for i := range 1000 {
			s.Set(fmt.Sprintf("set%03d", i))
		}
	}()

	readCtx, stopReadCtx := context.WithCancel(context.Background())
	go func() {
		writers.Wait()
		stopReadCtx()
	}()

	var readers sync.WaitGroup
	for i := range 1000 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			now, events := s.Sub(readCtx)
			seen := make(map[string]struct{})
			for _, v := range now.All {
				if _, ok := seen[v]; ok {
					t.Errorf("[%03d] duplicate value in slice set: %s", i, v)
				}
				seen[v] = struct{}{}
			}
			startKeys := slices.Collect(maps.Keys(seen))
			slices.Sort(startKeys)
			for e := range events {
				if e.Type != Add {
					t.Errorf("[%03d] unexpected event type: want Add:0, got %v", i, e.Type)
				}
				if _, ok := seen[e.New]; ok {
					t.Errorf("[%03d] duplicate value in event: %+v", i, e)
				}
				seen[e.New] = struct{}{}
			}
		}()
	}

	writers.Wait()
	readers.Wait()
}
