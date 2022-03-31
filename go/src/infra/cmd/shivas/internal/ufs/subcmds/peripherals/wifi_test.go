// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package peripherals

import (
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"strings"
	"testing"
)

func TestWifiCleanAndValidateFlags(t *testing.T) {
	// Test invalid flags
	errTests := []struct {
		cmd  *manageWifiCmd
		want []string
	}{
		{
			cmd:  &manageWifiCmd{},
			want: []string{errDUTMissing, errNoRouterAndNoFeature},
		},
		{
			cmd:  &manageWifiCmd{routers: [][]string{{"hostname: "}}},
			want: []string{errDUTMissing, errNoRouterAndNoFeature, errEmptyHostname},
		},
		{
			cmd:  &manageWifiCmd{routers: [][]string{{"hostname:h1 "}, {"hostname:h1"}}, dutName: "d1"},
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
	c := &manageWifiCmd{
		dutName:      "d",
		wifiFeatures: []string{"unknown"},
		routers: [][]string{
			{
				"hostname:h1",
				"model:test",
				"feature:unknown",
			},
			{
				"hostname:h2",
			},
		},
		mode: actionAdd,
	}
	if err := c.cleanAndValidateFlags(); err != nil {
		t.Errorf("cleanAndValidateFlags = %v; want nil", err)
	}
	wantFeatures := 1
	if n := len(c.wifiFeatures); n != wantFeatures {
		t.Errorf("len(c.wifiFeatures) = %d; want %d", n, wantFeatures)
	}
	wantRouters := 2
	if n := len(c.routers); n != wantRouters {
		t.Errorf("len(c.routers) = %d; want %d", n, wantRouters)
	}

}

func TestAddWifi(t *testing.T) {
	cmd := &manageWifiCmd{
		dutName:      "d",
		wifiFeatures: []string{"unknown"},
		routers: [][]string{
			{
				"hostname:h1",
				"model:test",
				"feature:unknown",
			},
			{
				"hostname:h2",
			},
		},
		mode: actionAdd,
	}
	cmd.cleanAndValidateFlags()

	// Test adding a duplicate and a valid BTP
	current := &lab.Wifi{
		WifiRouters: []*lab.WifiRouter{
			{
				Hostname: "h1",
			},
		},
	}

	if _, err := cmd.addWifi(current); err == nil {
		t.Errorf("addWifi(%v) succeded, expect duplication failure", current)
	}

	// Test adding two valid routers and one wifi feature
	current = &lab.Wifi{
		WifiRouters: []*lab.WifiRouter{
			{
				Hostname: "h3",
			},
		},
	}
	out, err := cmd.addWifi(current)
	if err != nil {
		t.Errorf("addWifi(%v) = %v, expect success", current, err)
	}
	wantRouters := 3
	if len(out.GetWifiRouters()) != wantRouters {
		t.Errorf("addWifi(%v) = %v, want total wifirouters %d", current, out.GetWifiRouters(), wantRouters)
	}
	wantFeatures := 1
	if len(out.GetFeatures()) != wantFeatures {
		t.Errorf("addWifi(%v) = %v, want total features %d", current, out, wantFeatures)
	}
}

func TestDeleteWifi(t *testing.T) {
	cmd := &manageWifiCmd{
		dutName: "d",
		routers: [][]string{
			{
				"hostname:h1",
				"model:test",
				"feature:unknown",
			},
			{
				"hostname:h2",
			},
		},
		mode: actionDelete,
	}
	cmd.cleanAndValidateFlags()

	// Test deleting two non-existent BTPs
	current := &lab.Wifi{
		WifiRouters: []*lab.WifiRouter{
			{
				Hostname: "h3",
			},
		},
	}

	if _, err := cmd.deleteWifi(current); err == nil {
		t.Errorf("deleteWifi(%v) succeeded, expected non-existent delete failure", current)
	}

	// Test deleting 2 of 3 routers
	current = &lab.Wifi{
		WifiRouters: []*lab.WifiRouter{
			{
				Hostname: "h1",
			},
			{
				Hostname: "h2",
			},
			{
				Hostname: "h3",
			},
		},
	}

	out, err := cmd.deleteWifi(current)
	if err != nil {
		t.Errorf("deleteWifi(%v) = %v, expect success", current, err)
	}
	want := "h3"
	if len(out.GetWifiRouters()) != 1 || out.GetWifiRouters()[0].GetHostname() != want {
		t.Fatalf("deleteWifi(%v) = %v, want %s", current, out.GetWifiRouters(), want)
	}
}

func TestReplaceWifi(t *testing.T) {
	cmd := &manageWifiCmd{
		dutName: "d",
		routers: [][]string{
			{
				"hostname:h1",
				"model:test",
				"feature:unknown",
			},
			{"hostname:h2"},
		},
		mode: actionReplace,
	}
	cmd.cleanAndValidateFlags()

	// Test replace two non-existent BTPs
	current := &lab.Wifi{
		WifiRouters: []*lab.WifiRouter{
			{
				Hostname: "h3",
			},
		},
	}
	want := 2
	if out, _ := cmd.replaceWifi(current); len(out.GetWifiRouters()) != want {
		t.Errorf("replaceWifi(%v) = %v, want %d replacing failure", current, out, want)
	}

}
