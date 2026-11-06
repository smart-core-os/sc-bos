package history

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

func Test_deduper_Changed(t *testing.T) {
	reading := func() *meterpb.MeterReading { return &meterpb.MeterReading{Usage: 1} }

	t.Run("maxInterval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			d := newDeduper[*meterpb.MeterReading](nil, newRecency(time.Hour))
			if !d.Changed(t.Context(), reading()) {
				t.Fatal("first reading not accepted")
			}
			time.Sleep(59 * time.Minute)
			if d.Changed(t.Context(), reading()) {
				t.Fatal("equal reading accepted inside maxInterval")
			}
			time.Sleep(time.Minute)
			if !d.Changed(t.Context(), reading()) {
				t.Fatal("equal reading not accepted once maxInterval passed")
			}
			if d.Changed(t.Context(), reading()) {
				t.Fatal("equal reading accepted straight after an accepted one")
			}
		})
	})

	t.Run("failed write", func(t *testing.T) {
		rec := newRecency(0)
		d := newDeduper[*meterpb.MeterReading](nil, rec)
		if !d.Changed(t.Context(), reading()) {
			t.Fatal("first reading not accepted")
		}
		rec.failed()
		if !d.Changed(t.Context(), reading()) {
			t.Fatal("equal reading not accepted after a failed write")
		}
		if d.Changed(t.Context(), reading()) {
			t.Fatal("equal reading accepted after the retry was accepted")
		}
	})

	t.Run("stale poll", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			d := newDeduper[*meterpb.MeterReading](nil, newRecency(time.Hour))
			var pollCtx context.Context
			poll := withReadStart(func(ctx context.Context, _ chan<- []byte) error {
				pollCtx = ctx
				return nil
			})
			_ = poll(t.Context(), nil)
			time.Sleep(time.Second)
			if !d.Changed(t.Context(), &meterpb.MeterReading{Usage: 2}) {
				t.Fatal("pulled reading not accepted")
			}
			if d.Changed(pollCtx, reading()) {
				t.Fatal("reading from a poll that began before the pulled one was accepted")
			}
		})
	})

	t.Run("no maxInterval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			d := newDeduper[*meterpb.MeterReading](nil, newRecency(0))
			if !d.Changed(t.Context(), reading()) {
				t.Fatal("first reading not accepted")
			}
			time.Sleep(1000 * time.Hour)
			if d.Changed(t.Context(), reading()) {
				t.Fatal("equal reading accepted without a maxInterval")
			}
			if !d.Changed(t.Context(), &meterpb.MeterReading{Usage: 2}) {
				t.Fatal("changed reading not accepted")
			}
		})
	})
}
