// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swmbot

import (
	"reflect"
	"strings"
	"testing"

	"infra/cros/dutstate"
)

// Test that Dumping and Loading a LocalDUTState struct returns an identical struct.
func TestMarshalAndUnmarshal(t *testing.T) {
	t.Parallel()
	lds := LocalDUTState{
		HostState: dutstate.Ready,
		ProvisionableLabels: ProvisionableLabels{
			"cros-version":        "lumpy-release/R00-0.0.0.0",
			"firmware-ro-version": "Google_000",
		},
		ProvisionableAttributes: ProvisionableAttributes{
			"job_repo_url": "http://127.0.0.1",
		},
	}
	data, err := Marshal(&lds)
	if err != nil {
		t.Fatalf("Error dumping dimensions: %s", err)
	}
	if strings.Contains(string(data), string(lds.HostState)) {
		t.Fatal("Host state serialized. Field should be ignored")
	}
	var got LocalDUTState
	err = Unmarshal(data, &got)
	if err != nil {
		t.Fatalf("Error loading test file: %s", err)
	}
	if len(got.HostState) > 0 {
		t.Errorf("Got state %v, expected  to be empty", got.HostState)
	}
	lds.HostState = ""
	if !reflect.DeepEqual(got, lds) {
		t.Errorf("Got %v, expected %v", got, lds)
	}
}

func TestUnmarshalInitializesBotInfo(t *testing.T) {
	t.Parallel()
	var lds LocalDUTState
	data, err := Marshal(&lds)
	if err != nil {
		t.Fatalf("Error dumping dimensions: %s", err)
	}
	err = Unmarshal(data, &lds)
	if err != nil {
		t.Fatalf("Error loading test file: %s", err)
	}

	exp := LocalDUTState{
		ProvisionableLabels:     ProvisionableLabels{},
		ProvisionableAttributes: ProvisionableAttributes{},
	}
	if !reflect.DeepEqual(lds, exp) {
		t.Errorf("Got %v, expected %v", lds, exp)
	}
}
