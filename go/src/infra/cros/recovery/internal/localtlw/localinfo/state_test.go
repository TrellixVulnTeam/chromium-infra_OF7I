// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package localinfo

import (
	"reflect"
	"testing"
)

// Test that Dumping and Loading a localDUTState struct returns an identical struct.
func TestMarshalAndUnmarshal(t *testing.T) {
	t.Parallel()
	lds := localDUTState{
		ProvisionableLabels: provisionableLabels{
			"cros-version":        "lumpy-release/R00-0.0.0.0",
			"firmware-ro-version": "Google_000",
		},
		ProvisionableAttributes: provisionableAttributes{
			"job_repo_url": "http://127.0.0.1",
		},
	}
	data, err := lds.marshal()
	if err != nil {
		t.Fatalf("Error marshal test localDUTState: %s", err)
	}
	var got localDUTState
	err = got.unmarshal(data)
	if err != nil {
		t.Fatalf("Error unmarshal data: %s", err)
	}
	if !reflect.DeepEqual(got, lds) {
		t.Errorf("Got %v, expected %v", got, lds)
	}
}

// Test that Dumping and Loading a localDutState struct that has empty map values for both
// the ProvisionableLabels as well as ProvisionableAttributes returns an identical struct.
func TestMarshalAndUnmarshalEmptyValue(t *testing.T) {
	t.Parallel()
	lds := localDUTState{
		ProvisionableLabels:     make(map[string]string),
		ProvisionableAttributes: make(map[string]string),
	}
	data, err := lds.marshal()
	if err != nil {
		t.Fatalf("Error marshal test localDUTState: %s", err)
	}
	var got localDUTState
	err = got.unmarshal(data)
	if err != nil {
		t.Fatalf("Error unmarshal data: %s", err)
	}
	if !reflect.DeepEqual(got, lds) {
		t.Errorf("Got %v, expected %v", got, lds)
	}
}
