// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hostinfo

import (
	"reflect"
	"strings"
	"testing"
)

func TestMarshalRoundTrip(t *testing.T) {
	t.Parallel()
	var his = []*HostInfo{
		{},
		{
			Labels: []string{
				"cros-version:lumpy-release/R66-0.0.0.0",
			},
			Attributes: map[string]string{
				"job_repo_url": "http://127.0.0.1",
			},
		},
	}
	for _, hi := range his {
		d, err := Marshal(hi)
		if err != nil {
			t.Errorf("Error writing HostInfo %#v: %s", hi, err)
			continue
		}
		got, err := Unmarshal(d)
		if err != nil {
			t.Errorf("Error reading HostInfo: %#v: %s", hi, err)
		}
		if !reflect.DeepEqual(got, hi) {
			t.Errorf("Write/Read roundtrip of %#v does not match, got %#v", hi, got)
		}
	}
}

// TestWriteFormat validates the marshaled HostInfo format, which is part of autoserv API.
func TestWriteFormat(t *testing.T) {
	t.Parallel()
	hi := HostInfo{
		Labels: []string{
			"cros-version:lumpy-release/R66-0.0.0.0",
			"another-label",
		},
		Attributes: map[string]string{
			"job_repo_url": "http://127.0.0.1",
		},
	}
	got, err := Marshal(&hi)
	if err != nil {
		t.Fatalf("Error writing HostInfo %#v: %s", hi, err)
	}
	blob := `
	{
		"labels": [
			"cros-version:lumpy-release/R66-0.0.0.0",
			"another-label"
		],
		"attributes": {
			"job_repo_url": "http://127.0.0.1"
		},
		"serializer_version": 1
	}`
	gotNoSpace := strings.Join(strings.Fields(string(got)), "")
	blobNoSpace := strings.Join(strings.Fields(blob), "")
	if gotNoSpace != blobNoSpace {
		t.Errorf("Incorrect format for dumped HostInfo, want: %s, got: %s.",
			blobNoSpace, gotNoSpace)
	}
}

func TestReadIncorrectSerializerVersion(t *testing.T) {
	t.Parallel()
	blob := []byte(`{"serializer_version": 0}`)
	got, err := Unmarshal(blob)
	if err != nil {
		t.Errorf("Unmarshal should have ignored the serializer version but didn't. Parsed `%s`to %#v", blob, got)
	}
}

var TestUnMarshalTests = []struct {
	in   string
	out  *HostInfo
	name string
}{
	{
		`{}`,
		&HostInfo{},
		`{}`,
	},
	{
		`{"serializer_version": 0}`,
		&HostInfo{},
		`serializer_version=0`,
	},
	{
		`{"serializer_version": 1}`,
		&HostInfo{},
		`serializer_version=1`,
	},
	{
		`{ "labels": [ "7" ] }`,
		&HostInfo{Labels: []string{"7"}},
		`nonempty labels`,
	},
	{
		`{ "attributes": { "a" : "b" } }`,
		&HostInfo{Attributes: map[string]string{"a": "b"}},
		`nonempty attributes`,
	},
}

func TestUnMarshal(t *testing.T) {
	for _, tt := range TestUnMarshalTests {
		t.Run(tt.name, func(t *testing.T) {
			unmarshalled, err := Unmarshal([]byte(tt.in))
			if err != nil {
				t.Errorf("failed to unmarshal: %s", err)
			}
			if !reflect.DeepEqual(unmarshalled, tt.out) {
				t.Errorf("wanted: (%s) got: (%s)", tt.out, unmarshalled)
			}
		})
	}
}
