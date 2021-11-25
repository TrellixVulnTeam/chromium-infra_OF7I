// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// moSysSkuCmd will retrieve the SKU label of the DUT.
	moSysSkuCmd = "mosys platform sku"
)

// updateDeviceSKUExec updates device's SKU label if not present in inventory
// or keep it the same if the args.DUT already has the value for SKU label.
func updateDeviceSKUExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
	skuLabelOutput, err := r(ctx, moSysSkuCmd)
	if err != nil {
		log.Debug(ctx, "Device sku label not found in the DUT.")
		return errors.Annotate(err, "update device sku label").Err()
	}
	args.DUT.DeviceSku = skuLabelOutput
	return nil
}

func init() {
	execs.Register("cros_update_device_sku", updateDeviceSKUExec)
}
