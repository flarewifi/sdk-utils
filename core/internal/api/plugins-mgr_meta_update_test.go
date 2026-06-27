//go:build !mono

package api

import (
	"reflect"
	"testing"
)

// droppedMembers underpins the on-device "remove a member on the next release"
// behavior: it is the set difference (old members − new members) that feeds the
// orphan/uninstall cascade in RepinMetaRecordsToLatest.
func TestDroppedMembers(t *testing.T) {
	cases := []struct {
		name    string
		old     []string
		current []string
		want    []string
	}{
		{"one dropped", []string{"a", "b", "c"}, []string{"a", "c"}, []string{"b"}},
		{"none dropped", []string{"a", "b"}, []string{"a", "b"}, nil},
		{"all dropped", []string{"a", "b"}, nil, []string{"a", "b"}},
		{"added only", []string{"a"}, []string{"a", "b"}, nil},
		{"empty old", nil, []string{"a"}, nil},
		{"preserves order of old", []string{"x", "y", "z"}, []string{"y"}, []string{"x", "z"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := droppedMembers(tc.old, tc.current)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("droppedMembers(%v, %v) = %v, want %v", tc.old, tc.current, got, tc.want)
			}
		})
	}
}
