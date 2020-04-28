package flagx

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testDimsVarData = []struct {
	in  string
	out map[string]string
}{
	{
		"a=b",
		map[string]string{
			"a": "b",
		},
	},
	{
		"",
		map[string]string{},
	},
}

var testDimsInitialMaps = []map[string]string{
	nil,
	{},
}

func TestDimsVar(t *testing.T) {
	// copyMap copies a map, preserving nil maps
	copyMap := func(a map[string]string) map[string]string {
		if a == nil {
			return nil
		}
		out := make(map[string]string)
		for k, v := range a {
			out[k] = v
		}
		return out
	}
	t.Parallel()
	for _, initMap := range testDimsInitialMaps {
		for _, tt := range testDimsVarData {
			tt := tt
			t.Run(tt.in, func(t *testing.T) {
				t.Parallel()
				c := copyMap(initMap)
				err := Dims(&c).Set(tt.in)
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				diff := cmp.Diff(tt.out, c)
				if diff != "" {
					msg := fmt.Sprintf("unexpected diff (%s)", diff)
					t.Errorf(msg)
				}
			})
		}
	}
}

var testSplitKeyValData = []struct {
	in  string
	key string
	val string
	err string
}{
	{
		"",
		"",
		"",
		`string "" is a malformed key-value pair`,
	},
	{
		"a=",
		"a",
		"",
		"",
	},
	{
		"k=v",
		"k",
		"v",
		"",
	},
}

func TestSplitKeyVal(t *testing.T) {
	t.Parallel()
	for _, tt := range testSplitKeyValData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.in), func(t *testing.T) {
			t.Parallel()
			expected := []string{tt.key, tt.val, tt.err}
			key, val, e := splitKeyVal(tt.in)
			actual := []string{key, val, errToString(e)}
			diff := cmp.Diff(expected, actual)
			if diff != "" {
				msg := fmt.Sprintf("unexpected diff (%s)", diff)
				t.Errorf(msg)
			}
		})
	}
}

func errToString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
