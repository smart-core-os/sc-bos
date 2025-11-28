package devices

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test_parsePath(t *testing.T) {
	n := func(name string) pathSegment {
		return pathSegment{Name: name}
	}
	i := func(index int) pathSegment {
		return pathSegment{Index: index, IsIndex: true}
	}

	tests := []struct {
		path    string
		want    []pathSegment
		wantErr error
	}{
		{"", []pathSegment{}, emptyPathErr},
		{".", []pathSegment{}, leadingDotErr},
		{"a", []pathSegment{n("a")}, nil},
		{"a.", []pathSegment{n("a")}, parsingNameErr},
		{".a", []pathSegment{}, leadingDotErr},
		{"abc", []pathSegment{n("abc")}, nil},
		{"世界", []pathSegment{n("世界")}, nil},
		{"a.b", []pathSegment{n("a"), n("b")}, nil},
		{"a.b.", []pathSegment{n("a"), n("b")}, parsingNameErr},
		{"a..b", []pathSegment{n("a")}, parsingNameErr},
		{"a.世界", []pathSegment{n("a"), n("世界")}, nil},
		{"世.界", []pathSegment{n("世"), n("界")}, nil},
		{"世界.b", []pathSegment{n("世界"), n("b")}, nil},
		{"a[0]", []pathSegment{n("a"), i(0)}, nil},
		{"a[10]", []pathSegment{n("a"), i(10)}, nil},
		{"a[+10]", []pathSegment{n("a"), i(10)}, nil},
		{"a[-10]", []pathSegment{n("a"), i(-10)}, nil},
		{"a[foo]", []pathSegment{n("a")}, parsingIndexErr},
		{"世界[1]", []pathSegment{n("世界"), i(1)}, nil},
		{"a[1].b", []pathSegment{n("a"), i(1), n("b")}, nil},
		{"a.b[1]", []pathSegment{n("a"), n("b"), i(1)}, nil},
		{"a[", []pathSegment{n("a")}, parsingIndexErr},
		{"a[1", []pathSegment{n("a")}, parsingIndexErr},
		{"[", []pathSegment{}, parsingIndexErr},
		{"[1]", []pathSegment{i(1)}, nil},
		{"[1.1]", []pathSegment{}, parsingIndexErr},
		{"[1][2]", []pathSegment{i(1), i(2)}, nil},
		{"a[1][2]", []pathSegment{n("a"), i(1), i(2)}, nil},
		{"a[1][2].b", []pathSegment{n("a"), i(1), i(2), n("b")}, nil},
		{"a[1].b[2].c", []pathSegment{n("a"), i(1), n("b"), i(2), n("c")}, nil},
		{"[1]b", []pathSegment{i(1)}, parsingPathErr},
		{"a[1]b", []pathSegment{n("a"), i(1)}, parsingPathErr},
		{"a[1]b[2]", []pathSegment{n("a"), i(1)}, parsingPathErr},
		{"a.[1]", []pathSegment{n("a")}, parsingNameErr},
		{"a[1].[2]", []pathSegment{n("a"), i(1)}, parsingNameErr},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			ps, err := parsePath(test.path)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("parsePath(%q) error = %v, wantErr %v", test.path, err, test.wantErr)
			}
			if diff := cmp.Diff(test.want, ps, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("parsePath(%q): -want +got:\n%s", test.path, diff)
			}
		})
	}
}
