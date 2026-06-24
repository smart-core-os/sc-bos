package ocitag

import (
	"strings"
	"testing"
)

func TestValid(t *testing.T) {
	valid := []string{
		"v1.2.3",
		"1.2.3",
		"latest",
		"V2",
		"a",
		"1",
		"_x",
		"v1_2-3.4",
		strings.Repeat("a", 128), // max length
	}
	for _, s := range valid {
		if !Valid(s) {
			t.Errorf("Valid(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",                       // empty
		"bad/tag",                // slash
		"1.2.3+build",            // plus
		"-leading",               // leading dash not allowed
		".leading",               // leading dot not allowed
		"with space",             // space
		"café",                   // non-ASCII
		strings.Repeat("a", 129), // one over max length
	}
	for _, s := range invalid {
		if Valid(s) {
			t.Errorf("Valid(%q) = true, want false", s)
		}
	}
}
