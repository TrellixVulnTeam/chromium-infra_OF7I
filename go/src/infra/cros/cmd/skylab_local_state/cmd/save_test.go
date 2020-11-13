// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_local_state"
	"go.chromium.org/luci/auth/client/authcli"
)

// Fake state loader that always returns a protected state.
func loadProtectedState(ctx context.Context, authFlags *authcli.Flags, crosUfsService, dutName string) (string, error) {
	return "needs_replacement", nil
}

// Fake state loader that never returns a protected state.
func loadUnprotectedState(ctx context.Context, authFlags *authcli.Flags, crosUfsService, dutName string) (string, error) {
	return "i_am_an_unprotected_state", nil
}

func TestNewDutStateFromHostInfo(t *testing.T) {
	Convey("When a DUT state is updated only provisionable labels and attributes are changed.", t, func() {
		i := &skylab_local_state.AutotestHostInfo{
			Attributes: map[string]string{
				"dummy-attribute": "dummy-value",
				"job_repo_url":    "dummy-url",
				"outlet_changed":  "true",
			},
			Labels: []string{
				"dummy-label:dummy-value",
				"cros-version:dummy-os-version",
				"fwro-version:dummy-fwro-version",
				"fwrw-version:dummy-fwrw-version",
			},
			SerializerVersion: 1,
		}

		state := newDutStateFromHostInfo(i)

		want := &lab_platform.DutState{
			ProvisionableAttributes: map[string]string{
				"job_repo_url":   "dummy-url",
				"outlet_changed": "true",
			},
			ProvisionableLabels: map[string]string{
				"cros-version": "dummy-os-version",
				"fwro-version": "dummy-fwro-version",
				"fwrw-version": "dummy-fwrw-version",
			},
		}

		So(want, ShouldResemble, state)
	})
}

func TestOverwriteRequestedDUTStateIfProtected(t *testing.T) {
	Convey("When current DUT state is in protected list, requested DUT state in the save request should be overwritten with current state.", t, func() {
		c := &saveRun{}
		saveRequest := &skylab_local_state.SaveRequest{
			Config:   &skylab_local_state.Config{AutotestDir: "dummy_autotest_dir"},
			DutId:    "dummy_dut_id",
			DutState: "dummy_state",
		}
		got, _ := c.ensureNoProtectedStateOverwrite(context.Background(), saveRequest, loadProtectedState)

		want := skylab_local_state.SaveRequest{
			Config:   &skylab_local_state.Config{AutotestDir: "dummy_autotest_dir"},
			DutId:    "dummy_dut_id",
			DutState: "needs_replacement",
		}

		So(want, ShouldResemble, got)
	})
}

func TestDontOverwriteRequestedDUTStateIfNotProtected(t *testing.T) {
	Convey("When current DUT state is not in protected list, requested DUT state in the save request should not be overwritten with current state.", t, func() {
		c := &saveRun{}
		saveRequest := &skylab_local_state.SaveRequest{
			Config:   &skylab_local_state.Config{AutotestDir: "dummy_autotest_dir"},
			DutId:    "dummy_dut_id",
			DutState: "dummy_state",
		}
		got, _ := c.ensureNoProtectedStateOverwrite(context.Background(), saveRequest, loadUnprotectedState)

		want := skylab_local_state.SaveRequest{
			Config:   &skylab_local_state.Config{AutotestDir: "dummy_autotest_dir"},
			DutId:    "dummy_dut_id",
			DutState: "dummy_state",
		}

		So(want, ShouldResemble, got)
	})
}
