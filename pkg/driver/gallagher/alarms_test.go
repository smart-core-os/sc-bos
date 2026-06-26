package gallagher

import (
	"context"
	"strconv"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
)

// newTestController builds a controller with a ring of size n and adds count
// events, ids "e0".."e{count-1}" in oldest-first order.
func newTestController(t *testing.T, n, count int) *SecurityEventController {
	t.Helper()
	sc := newSecurityEventController(nil, zap.NewNop(), n)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		ev := newSecurityEvent(base.Add(time.Duration(i)*time.Minute), "e"+strconv.Itoa(i), "msg", 0, "src", "name")
		sc.addSecurityEvent(context.Background(), ev)
	}
	return sc
}

// ids extracts the event ids from a response in returned order.
func ids(resp *securityeventpb.ListSecurityEventsResponse) []string {
	out := make([]string, 0, len(resp.SecurityEvents))
	for _, e := range resp.SecurityEvents {
		out = append(out, e.Id)
	}
	return out
}

func TestListSecurityEvents(t *testing.T) {
	tests := []struct {
		name      string
		ringSize  int
		numEvents int
		pageSize  int32
		wantIDs   []string
		wantTotal int32
		wantToken string
	}{
		{
			name:      "empty ring",
			ringSize:  5,
			numEvents: 0,
			pageSize:  10,
			wantIDs:   nil,
			wantTotal: 0,
			wantToken: "",
		},
		{
			name:      "partially full, newest first",
			ringSize:  5,
			numEvents: 3,
			pageSize:  10,
			wantIDs:   []string{"e2", "e1", "e0"},
			wantTotal: 3,
			wantToken: "",
		},
		{
			name:      "full and wrapped keeps newest",
			ringSize:  3,
			numEvents: 5,
			pageSize:  10,
			wantIDs:   []string{"e4", "e3", "e2"},
			wantTotal: 3,
			wantToken: "",
		},
		{
			name:      "page size zero defaults to all",
			ringSize:  5,
			numEvents: 4,
			pageSize:  0,
			wantIDs:   []string{"e3", "e2", "e1", "e0"},
			wantTotal: 4,
			wantToken: "",
		},
		{
			name:      "first page of many",
			ringSize:  5,
			numEvents: 5,
			pageSize:  2,
			wantIDs:   []string{"e4", "e3"},
			wantTotal: 5,
			wantToken: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := newTestController(t, tt.ringSize, tt.numEvents)
			resp, err := sc.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{PageSize: tt.pageSize})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := ids(resp); !equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", got, tt.wantIDs)
			}
			if resp.TotalSize != tt.wantTotal {
				t.Errorf("TotalSize = %d, want %d", resp.TotalSize, tt.wantTotal)
			}
			if resp.NextPageToken != tt.wantToken {
				t.Errorf("NextPageToken = %q, want %q", resp.NextPageToken, tt.wantToken)
			}
		})
	}
}

// TestListSecurityEvents_Paging walks every page and checks the full sequence,
// including the boundary where the last page lands exactly on the oldest event
// (the case that previously produced a "-1" token).
func TestListSecurityEvents_Paging(t *testing.T) {
	sc := newTestController(t, 4, 4)

	var got []string
	token := ""
	pages := 0
	for {
		resp, err := sc.ListSecurityEvents(context.Background(), &securityeventpb.ListSecurityEventsRequest{
			PageSize:  2,
			PageToken: token,
		})
		if err != nil {
			t.Fatalf("page %d: unexpected error: %v", pages, err)
		}
		got = append(got, ids(resp)...)
		if resp.NextPageToken == "" {
			break
		}
		token = resp.NextPageToken
		pages++
		if pages > 10 {
			t.Fatal("pagination did not terminate")
		}
	}

	want := []string{"e3", "e2", "e1", "e0"}
	if !equal(got, want) {
		t.Errorf("paged ids = %v, want %v", got, want)
	}
}

func TestListSecurityEvents_InvalidArgs(t *testing.T) {
	sc := newTestController(t, 5, 3)

	tests := []struct {
		name string
		req  *securityeventpb.ListSecurityEventsRequest
	}{
		{"negative page size", &securityeventpb.ListSecurityEventsRequest{PageSize: -1}},
		{"non-numeric token", &securityeventpb.ListSecurityEventsRequest{PageSize: 2, PageToken: "abc"}},
		{"negative token", &securityeventpb.ListSecurityEventsRequest{PageSize: 2, PageToken: "-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sc.ListSecurityEvents(context.Background(), tt.req)
			if status.Code(err) != codes.InvalidArgument {
				t.Errorf("got err %v, want InvalidArgument", err)
			}
		})
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
