// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"

	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/state"
)

// GetDutState returns the DutState for the ChromeOS device.
func GetDutState(ctx context.Context, id, hostname string) (*chromeosLab.DutState, error) {
	return nil, nil
}

// ListDutStates lists the DutStates in datastore.
func ListDutStates(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*chromeosLab.DutState, string, error) {
	return nil, "", nil
}

// UpdateDutState updates the dut state for a ChromeOS DUT
func UpdateDutState(ctx context.Context, ds *chromeosLab.DutState) (*chromeosLab.DutState, error) {
	f := func(ctx context.Context) error {
		// It's not ok that no such DUT (machine lse) exists in UFS.
		_, err := inventory.GetMachineLSE(ctx, ds.GetHostname())
		if err != nil {
			return err
		}
		hc := &HistoryClient{}
		// It's ok that no old dut state for this DUT exists before.
		oldDS, _ := state.GetDutState(ctx, ds.GetId().GetValue())

		if _, err := state.UpdateDutStates(ctx, []*chromeosLab.DutState{ds}); err != nil {
			return errors.Annotate(err, "Unable to update dut state for %s", ds.GetId().GetValue()).Err()
		}
		hc.LogDutStateChanges(oldDS, ds)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "UpdateDutState (%s, %s) - %s", ds.GetId().GetValue(), ds.GetHostname(), err)
		return nil, err
	}
	return ds, nil
}
