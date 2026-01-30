package memstore

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/smart-core-os/sc-bos/pkg/history"
)

// makeSlice returns a slice of n records, each 1 hour apart in ascending order, with the last record being now.
func makeSlice(n int) (slice, time.Time) {
	now, _ := time.Parse(time.RFC3339, "2023-11-16T10:00:00Z")
	now = now.Round(time.Second) // get rid of millis, etc
	all := make(slice, n)
	for i := range all {
		t := now.Add(time.Duration(i-n+1) * time.Hour)
		all[i] = history.Record{ID: createTimeToID(t), CreateTime: t, Payload: []byte(fmt.Sprint(i))}
	}
	return all, now
}

func TestStore_Append_gc(t *testing.T) {
	const n = 10
	all, now := makeSlice(n)

	tests := []struct {
		maxAge   time.Duration
		maxCount int64
		want     slice
	}{
		{0 * time.Hour, 0, all},     // no gc needed, no cap
		{n * time.Hour, 0, all},     // no gc needed, age cap is too big
		{0 * time.Hour, n, all},     // no gc needed, count cap is to big
		{n * time.Hour, n, all},     // no gc needed, caps are to big
		{2 * time.Hour, 0, all[7:]}, // There are three records between now and -2h: 10, 9, and 8
		{0 * time.Hour, 3, all[7:]},
		{3 * time.Hour, 3, all[7:]}, // count beats age
		{2 * time.Hour, 4, all[7:]}, // age beats count
	}
	for _, tt := range tests {
		name := fmt.Sprintf("maxAge=%v,maxCount=%d", tt.maxAge, tt.maxCount)
		t.Run(name, func(t *testing.T) {
			s := &Store{
				slice:    all[: n-1 : n-1], // make sure slice has no spare capacity to avoid Append assigning to the underlying array of all
				maxAge:   tt.maxAge,
				maxCount: tt.maxCount,
				now:      func() time.Time { return now },
			}
			_, err := s.Append(context.Background(), all[n-1].Payload)
			if err != nil {
				t.Errorf("Append() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, s.slice); diff != "" {
				t.Errorf("Append() slice diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSlice_Read(t *testing.T) {
	const n = 10
	all, now := makeSlice(n)
	var empty history.Record
	beforeStart1 := history.Record{CreateTime: now.Add(-n*time.Hour - 2*time.Hour), Payload: []byte("-2")}
	beforeStart2 := history.Record{CreateTime: now.Add(-n*time.Hour - time.Hour), Payload: []byte("-1")}
	afterEnd1 := history.Record{CreateTime: now.Add(time.Hour), Payload: []byte("+1")}
	afterEnd2 := history.Record{CreateTime: now.Add(2 * time.Hour), Payload: []byte("+2")}

	tests := []struct {
		from, to history.Record
		want     []history.Record
	}{
		{empty, empty, all},                   // no from/to
		{empty, all[5], all[:5]},              // no from
		{all[5], empty, all[5:]},              // no to
		{all[3], all[7], all[3:7]},            // range
		{beforeStart1, beforeStart2, slice{}}, // before all
		{afterEnd1, afterEnd2, slice{}},       // after all
		{beforeStart1, afterEnd2, all},        // wider range
		{beforeStart1, all[1], all[:1]},
		{all[n-2], afterEnd2, all[n-2:]},
	}

	formatRecord := func(r history.Record) string {
		if r.IsZero() {
			return "-"
		}
		return string(r.Payload)
	}
	for _, tt := range tests {
		name := fmt.Sprintf("[%s,%s)", formatRecord(tt.from), formatRecord(tt.to))
		t.Run(name, func(t *testing.T) {
			s := all.Slice(tt.from, tt.to)

			t.Run("Read", func(t *testing.T) {
				dst := make([]history.Record, len(all)+4)
				read, err := s.Read(context.Background(), dst)
				if err != nil {
					t.Errorf("Read() error = %v", err)
					return
				}
				if diff := cmp.Diff(tt.want, dst[:read]); diff != "" {
					t.Errorf("Read() diff (-want +got):\n%s", diff)
				}
			})

			t.Run("ReadDesc", func(t *testing.T) {
				dst := make([]history.Record, len(all)+4)
				read, err := s.ReadDesc(context.Background(), dst)
				if err != nil {
					t.Errorf("ReadDesc() error = %v", err)
					return
				}
				want := make([]history.Record, len(tt.want))
				copy(want, tt.want)
				slices.Reverse(want)
				if diff := cmp.Diff(want, dst[:read]); diff != "" {
					t.Errorf("ReadDesc() diff (-want +got):\n%s", diff)
				}
			})

			// can only test partial responses if we want more than 1 record
			if len(tt.want) <= 1 {
				return
			}

			t.Run("Read partial", func(t *testing.T) {
				dst := make([]history.Record, len(tt.want)-1)
				read, err := s.Read(context.Background(), dst)
				if err != nil {
					t.Errorf("Read() error = %v", err)
					return
				}
				if diff := cmp.Diff(tt.want[:len(dst)], dst[:read]); diff != "" {
					t.Errorf("Read() diff (-want +got):\n%s", diff)
				}
			})

			t.Run("ReadDesc partial", func(t *testing.T) {
				dst := make([]history.Record, len(tt.want)-1)
				want := make([]history.Record, len(tt.want)-1)
				copy(want, tt.want[1:])
				slices.Reverse(want)
				read, err := s.ReadDesc(context.Background(), dst)
				if err != nil {
					t.Errorf("ReadDesc() error = %v", err)
					return
				}
				if diff := cmp.Diff(want, dst[:read]); diff != "" {
					t.Errorf("ReadDesc() diff (-want +got):\n%s", diff)
				}
			})
		})
	}
}
