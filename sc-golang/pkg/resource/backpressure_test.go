package resource

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
)

func Test_mergeCollectionExcess(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"one", []string{"k1:foo"}, []string{"k1:foo"}},
		{"reorder", []string{"k1:foo", "k2:a", "k1:foo>bar"}, []string{"k2:a", "k1:bar"}},
		{"add,remove", []string{"k1:>foo", "k1:foo>", "k2:done"}, []string{"k2:done"}},
		{"add,update,remove", []string{"k1:>foo", "k1:foo>bar", "k1:bar>", "k2:done"}, []string{"k2:done"}},
		{"remove,add", []string{"k1:foo>", "k1:>bar"}, []string{"k1:foo>+bar"}},
		{"updates", []string{"k1:foo>bar", "k1:bar>baz", "k1:baz>que"}, []string{"k1:foo>que"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := make(chan any)
			out := mergeCollectionExcess(in)
			<-sendTo(in, tt.in)
			got := drain(out, len(tt.want))
			want := parseAllCaseChanges(tt.want...)
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mergeCollectionExcess() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	multiMergeTests := []struct {
		name string
		in   [][]string
		want [][]string
	}{
		{"one", [][]string{{"k1:foo"}, {"k1:foo>"}}, [][]string{{"k1:foo"}, {"k1:foo>"}}},
		{"independent", [][]string{{"k1:foo>bar"}, {"k1:bar>baz"}}, [][]string{{"k1:foo>bar"}, {"k1:bar>baz"}}},
	}
	for _, tt := range multiMergeTests {
		t.Run(tt.name, func(t *testing.T) {
			in := make(chan any)
			out := mergeCollectionExcess(in)
			for i := range tt.in {
				<-sendTo(in, tt.in[i])
				got := drain(out, len(tt.want[i]))
				want := parseAllCaseChanges(tt.want[i]...)
				if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
					t.Errorf("mergeCollectionExcess() mismatch (-want +got):\n%s", diff)
				}
			}

		})
	}
}

// parseCaseChange parses a string of the form "id:old>new" into a CollectionChange.
func parseCaseChange(s string) CollectionChange {
	k, v, _ := strings.Cut(s, ":")
	out := CollectionChange{
		Id: k,
	}
	o, n, found := strings.Cut(v, ">")
	switch {
	case !found: // add: "foo"
		out.ChangeType = types.ChangeType_ADD
		out.NewValue = &traits.Metadata{Name: v}
	case o == "": // add: ">foo"
		out.ChangeType = types.ChangeType_ADD
		out.NewValue = &traits.Metadata{Name: n}
	case n == "": // del: "foo>"
		out.ChangeType = types.ChangeType_REMOVE
		out.OldValue = &traits.Metadata{Name: o}
	case n[0] == '+': // replace: "foo>+bar"
		out.ChangeType = types.ChangeType_REPLACE
		out.OldValue = &traits.Metadata{Name: o}
		out.NewValue = &traits.Metadata{Name: n[1:]}
	default: // update: "foo>bar"
		out.ChangeType = types.ChangeType_UPDATE
		out.OldValue = &traits.Metadata{Name: o}
		out.NewValue = &traits.Metadata{Name: n}
	}
	return out
}

func parseAllCaseChanges(ss ...string) []CollectionChange {
	var out []CollectionChange
	for _, s := range ss {
		out = append(out, parseCaseChange(s))
	}
	return out
}

func sendTo(out chan<- any, vals []string) (sent <-chan struct{}) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, v := range vals {
			change := parseCaseChange(v)
			out <- &change
		}
	}()
	return done
}

func drain(ch <-chan any, n int) []CollectionChange {
	out := make([]CollectionChange, n)
	for i := 0; i < n; i++ {
		out[i] = *(<-ch).(*CollectionChange)
	}
	return out
}
