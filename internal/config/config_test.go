package config

import (
	"reflect"
	"testing"
)

func TestSplitList(t *testing.T) {
	tests := map[string][]string{
		"":                      nil,
		"a":                     {"a"},
		"a,b":                   {"a", "b"},
		" a , b ,, c ":          {"a", "b", "c"},
		"http://x:1, https://y": {"http://x:1", "https://y"},
	}
	for in, want := range tests {
		if got := splitList(in); !reflect.DeepEqual(got, want) {
			t.Errorf("splitList(%q) = %q, want %q", in, got, want)
		}
	}
}
