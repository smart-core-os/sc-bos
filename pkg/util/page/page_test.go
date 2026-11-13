package page

import (
	"encoding/base64"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestList(t *testing.T) {
	var allItems []string
	for i := range 1500 {
		allItems = append(allItems, fmt.Sprintf("item-%04d", i))
	}
	idFunc := func(r string) string { return r }

	t.Run("page progression", func(t *testing.T) {
		pages := []struct {
			name string
			size int32
			want []string
		}{
			{name: "first page, default size", want: allItems[0:50]},
			{name: "smaller page size", size: 10, want: allItems[50:60]},
			{name: "bigger page size", size: 140, want: allItems[60:200]},
			{name: "page size capped", size: 1100, want: allItems[200:1200]},
			{name: "last page, get more than available", size: 500, want: allItems[1200:1500]},
		}

		lastPageToken := ""
		for _, page := range pages {
			gotItems, gotSize, gotNextPageToken, err := List(testRequest{pageSize: page.size, pageToken: lastPageToken}, idFunc, func() []string {
				return allItems
			})
			lastPageToken = gotNextPageToken
			if err != nil {
				t.Errorf("List(%v) returned error: %v", page.name, err)
			}
			if gotSize != len(allItems) {
				t.Errorf("List(%v) got size %d, want %d", page.name, gotSize, len(allItems))
			}
			if diff := cmp.Diff(page.want, gotItems); diff != "" {
				t.Errorf("List(%v) got items mismatch (-want +got):\n%s", page.name, diff)
			}
			if gotItems[len(gotItems)-1] == allItems[len(allItems)-1] {
				// check no token for last page
				if gotNextPageToken != "" {
					t.Errorf("List(%v) got next page token %q, want empty", page.name, gotNextPageToken)
				}
			}
		}
	})

	// these tests all use allItems[:10] as the input
	lastPageTests := []struct {
		name string
		size int32
		want []string
	}{
		{name: "exact last page", size: 10, want: allItems[:10]},
		{name: "one more item", size: 11, want: allItems[:10]},
		{name: "one less item", size: 9, want: allItems[:9]},
	}
	for _, tt := range lastPageTests {
		t.Run(tt.name, func(t *testing.T) {
			gotItems, gotSize, gotNextPageToken, err := List(testRequest{pageSize: tt.size}, idFunc, func() []string {
				return allItems[:10]
			})
			if err != nil {
				t.Errorf("last page returned error: %v", err)
			}
			if gotSize != 10 {
				t.Errorf("last page size %d, want %d", gotSize, 10)
			}
			if diff := cmp.Diff(tt.want, gotItems); diff != "" {
				t.Errorf("last page items mismatch (-want +got):\n%s", diff)
			}
			if len(tt.want) < 10 {
				if gotNextPageToken == "" {
					t.Errorf("last page got empty next page token, want non-empty")
				}
			} else {
				if gotNextPageToken != "" {
					t.Errorf("last page got next page token %q, want empty", gotNextPageToken)
				}
			}
		})
	}

	// a negative page size used to index before the start of the list and panic (SCB-1470)
	negativeSizeTests := []struct {
		name  string
		size  int32
		items []string
		want  []string
	}{
		{name: "negative size, full list", size: -1, items: allItems, want: allItems[:50]},
		{name: "most negative size, full list", size: math.MinInt32, items: allItems, want: allItems[:50]},
		{name: "negative size, one item", size: -1, items: allItems[:1], want: allItems[:1]},
		{name: "negative size, empty list", size: -1, items: nil, want: nil},
	}
	for _, tt := range negativeSizeTests {
		t.Run(tt.name, func(t *testing.T) {
			gotItems, gotSize, _, err := List(testRequest{pageSize: tt.size}, idFunc, func() []string {
				return slices.Clone(tt.items)
			})
			if err != nil {
				t.Fatalf("List returned error: %v", err)
			}
			if gotSize != len(tt.items) {
				t.Errorf("List got size %d, want %d", gotSize, len(tt.items))
			}
			if diff := cmp.Diff(tt.want, gotItems, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("List got items mismatch (-want +got):\n%s", diff)
			}
		})
	}

	for _, tt := range []string{"bad base64", base64.StdEncoding.EncodeToString([]byte("bad proto"))} {
		name := fmt.Sprintf("bad page token (%s)", tt)
		t.Run(name, func(t *testing.T) {
			_, _, _, err := List(testRequest{pageToken: tt}, idFunc, func() []string {
				t.Fatalf("list function should not be called with bad token")
				return nil
			})
			if code := status.Code(err); code != codes.InvalidArgument {
				t.Errorf("bad token should result in InvalidArgument error, got %v", err)
			}
		})
	}

	t.Run("options", func(t *testing.T) {
		t.Run("WithDefaultPageSize", func(t *testing.T) {
			gotItems, gotSize, gotNextPageToken, err := List(testRequest{}, idFunc, func() []string {
				return allItems
			}, WithDefaultPageSize(10))
			if err != nil {
				t.Errorf("List with default page size returned error: %v", err)
			}
			if gotSize != len(allItems) {
				t.Errorf("List with default page size got size %d, want %d", gotSize, len(allItems))
			}
			if diff := cmp.Diff(allItems[:10], gotItems); diff != "" {
				t.Errorf("List with default page size got items mismatch (-want +got):\n%s", diff)
			}
			if gotNextPageToken == "" {
				t.Errorf("List with default page size got empty next page token, want non-empty")
			}
		})

		t.Run("WithMaxPageSize", func(t *testing.T) {
			gotItems, gotSize, gotNextPageToken, err := List(testRequest{pageSize: 1000}, idFunc, func() []string {
				return allItems
			}, WithMaxPageSize(100))
			if err != nil {
				t.Errorf("List with max page size returned error: %v", err)
			}
			if gotSize != len(allItems) {
				t.Errorf("List with max page size got size %d, want %d", gotSize, len(allItems))
			}
			if diff := cmp.Diff(allItems[:100], gotItems); diff != "" {
				t.Errorf("List with max page size got items mismatch (-want +got):\n%s", diff)
			}
			if gotNextPageToken == "" {
				t.Errorf("List with max page size got empty next page token, want non-empty")
			}
		})
	})
}

type testRequest struct {
	pageToken string
	pageSize  int32
}

func (r testRequest) GetPageToken() string {
	return r.pageToken
}

func (r testRequest) GetPageSize() int32 {
	return r.pageSize
}
