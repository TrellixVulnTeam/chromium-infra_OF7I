package ufspb

import "testing"

func TestValidateHostnames(t *testing.T) {
	tt := []struct {
		in     []string
		wantOK bool
	}{
		{[]string{"h1", "h2"}, true},
		{[]string{"h1", "h1"}, false},
		{[]string{"", "h1"}, false},
		{nil, true},
	}

	for _, test := range tt {
		err := validateHostnames(test.in, "")
		if test.wantOK && err != nil {
			t.Errorf("validateHostnames(%v) failed %v", test.in, err)
		}
		if !test.wantOK && err == nil {
			t.Errorf("validateHostnames(%v) succeeded but want failure", test.in)
		}
	}
}
