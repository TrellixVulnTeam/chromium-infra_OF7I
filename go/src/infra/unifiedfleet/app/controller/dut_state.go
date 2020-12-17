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
	"infra/unifiedfleet/app/util"
)

// UpdateDutState updates the dut state for a ChromeOS DUT
func UpdateDutState(ctx context.Context, ds *chromeosLab.DutState) (*chromeosLab.DutState, error) {
	newCtx, err := util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		logging.Debugf(newCtx, "Failed to set os namespace in context: %s", err.Error())
		return nil, err
	}

	f := func(ctx context.Context) error {
		// It's not ok that no such DUT (machine lse) exists in UFS.
		_, err = inventory.GetMachineLSE(newCtx, ds.GetHostname())
		if err != nil {
			return errors.Annotate(err, "UpdateDutState").Err()
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

	if err := datastore.RunInTransaction(newCtx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update dut state: %s", err)
		return nil, err
	}
	return ds, nil
}
