package tasks

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

func TestDimsVar(t *testing.T) {
	t.Parallel()
	for _, tt := range testDimsVarData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.in), func(t *testing.T) {
			t.Parallel()
			c := &leaseDutRun{}
			err := dimsVar{data: c}.Set(tt.in)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			actual := c.dims
			diff := cmp.Diff(tt.out, actual)
			if diff != "" {
				msg := fmt.Sprintf("unexpected diff (%s)", diff)
				t.Errorf(msg)
			}
		})
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
		`string () does not contain a key and value`,
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
