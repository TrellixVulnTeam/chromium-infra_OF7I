// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package peripherals

import (
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"strings"
	"testing"
)

func TestCleanAndValidateFlags(t *testing.T) {
	// Test invalid flags
	errTests := []struct {
		cmd  *manageBTPsCmd
		want []string
	}{
		{
			cmd:  &manageBTPsCmd{},
			want: []string{errDUTMissing, errNoHostname},
		},
		{
			cmd:  &manageBTPsCmd{hostnames: []string{" "}},
			want: []string{errDUTMissing, errNoHostname, errEmptyHostname},
		},
		{
			cmd:  &manageBTPsCmd{hostnames: []string{"h1 ", "h1"}, dutName: "d1"},
			want: []string{errDuplicateHostname},
		},
	}

	for _, tt := range errTests {
		err := tt.cmd.cleanAndValidateFlags()
		if err == nil {
			t.Errorf("cleanAndValidateFlags = nil; want errors: %v", tt.want)
			continue
		}
		for _, errStr := range tt.want {
			if !strings.Contains(err.Error(), errStr) {
				t.Errorf("cleanAndValidateFlags = %q; want err %q included", err, errStr)
			}
		}
	}

	// Test valid flags with hostname cleanup
	c := &manageBTPsCmd{
		dutName:   "d",
		hostnames: []string{"h1", "h2"},
		mode:      actionAdd,
	}
	if err := c.cleanAndValidateFlags(); err != nil {
		t.Errorf("cleanAndValidateFlags = %v; want nil", err)
	}
	want := 2
	if n := len(c.hostnames); n != want {
		t.Errorf("len(c.hostnames) = %d; want %d", n, want)
	}
}

func TestAddBTPs(t *testing.T) {
	cmd := &manageBTPsCmd{dutName: "d", hostnames: []string{"h1", "h2"}, mode: actionAdd}
	cmd.cleanAndValidateFlags()

	// Test adding a duplicate and a valid BTP
	current := []*lab.BluetoothPeer{{Device: &lab.BluetoothPeer_RaspberryPi{RaspberryPi: &lab.RaspberryPi{Hostname: "h1"}}}}
	if _, err := cmd.addBTPs(current); err == nil {
		t.Errorf("addBTPs(%v) succeded, expect duplication failure", current)
	}

	// Test adding two valid BTPs
	current = []*lab.BluetoothPeer{{Device: &lab.BluetoothPeer_RaspberryPi{RaspberryPi: &lab.RaspberryPi{Hostname: "h3"}}}}
	out, err := cmd.addBTPs(current)
	if err != nil {
		t.Errorf("addBTPs(%v) = %v, expect success", current, err)
	}
	want := 3
	if len(out) != want {
		t.Errorf("addBTPs(%v) = %v, want total %d", current, out, want)
	}
}

func TestDeleteBTPs(t *testing.T) {
	cmd := &manageBTPsCmd{dutName: "d", hostnames: []string{"h1", "h2"}, mode: actionDelete}
	cmd.cleanAndValidateFlags()

	// Test deleting two non-existent BTPs
	current := []*lab.BluetoothPeer{{Device: &lab.BluetoothPeer_RaspberryPi{RaspberryPi: &lab.RaspberryPi{Hostname: "h3"}}}}
	if _, err := cmd.deleteBTPs(current); err == nil {
		t.Errorf("deleteBTPs(%v) succeeded, expected non-existent delete failure", current)
	}

	// Test deleting 2 of 3 BTPs
	current = []*lab.BluetoothPeer{
		createBTP("h1"),
		createBTP("h2"),
		createBTP("h3"),
	}
	out, err := cmd.deleteBTPs(current)
	if err != nil {
		t.Errorf("deleteBTPs(%v) = %v, expect success", current, err)
	}
	want := "h3"
	if len(out) != 1 || out[0].GetRaspberryPi().GetHostname() != want {
		t.Fatalf("deleteBTPs(%v) = %v, want %s", current, out, want)
	}
}
