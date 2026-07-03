package hubpage

import (
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
)

func nodes(names ...string) []*hubpb.HubNode {
	out := make([]*hubpb.HubNode, len(names))
	for i, n := range names {
		out[i] = &hubpb.HubNode{Name: n}
	}
	return out
}

func names(ns []*hubpb.HubNode) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = n.GetName()
	}
	return out
}

func TestPaginate_SinglePage(t *testing.T) {
	page, next, total, err := Paginate(nodes("c", "a", "b"), 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if next != "" {
		t.Errorf("next = %q, want empty (no more pages)", next)
	}
	// results are sorted by name
	got := names(page)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("page = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("page = %v, want %v", got, want)
		}
	}
}

func TestPaginate_TraversesAllPagesInOrderWithoutGapsOrDuplicates(t *testing.T) {
	all := nodes("e", "b", "g", "a", "f", "d", "c") // 7 nodes, unsorted
	want := []string{"a", "b", "c", "d", "e", "f", "g"}

	var got []string
	token := ""
	iterations := 0
	for {
		iterations++
		if iterations > 100 {
			t.Fatal("pagination did not terminate")
		}
		page, next, total, err := Paginate(all, 3, token)
		if err != nil {
			t.Fatal(err)
		}
		if total != int32(len(all)) {
			t.Errorf("total = %d, want %d", total, len(all))
		}
		got = append(got, names(page)...)
		if next == "" {
			break
		}
		token = next
	}

	if len(got) != len(want) {
		t.Fatalf("traversed %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("traversed %v, want %v", got, want)
		}
	}
}

func TestPaginate_Empty(t *testing.T) {
	page, next, total, err := Paginate(nil, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 0 || next != "" || total != 0 {
		t.Errorf("got page=%v next=%q total=%d, want empty", names(page), next, total)
	}
}

func TestPaginate_PageSizeCapAndDefault(t *testing.T) {
	all := nodes("a", "b", "c")
	// pageSize 0 falls back to the default (50), so all fit on one page.
	page, next, _, err := Paginate(all, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 3 || next != "" {
		t.Errorf("default page size: got %d nodes next=%q, want 3 and empty", len(page), next)
	}
}

func TestPaginate_BadToken(t *testing.T) {
	if _, _, _, err := Paginate(nodes("a"), 10, "!!!not-base64!!!"); err == nil {
		t.Error("expected error for malformed page token, got nil")
	}
}
