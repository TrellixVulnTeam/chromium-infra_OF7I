// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package wifirouter

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/tlw"
)

// setStateBrokenExec sets state as BROKEN.
func setStateBrokenExec(ctx context.Context, info *execs.ExecInfo) error {
	if h, err := activeHost(info.RunArgs); err != nil {
		return errors.Annotate(err, "set state broken").Err()
	} else {
		h.State = tlw.WifiRouterHost_BROKEN
	}
	return nil
}

// setStateWorkingExec sets state as WORKING.
func setStateWorkingExec(ctx context.Context, info *execs.ExecInfo) error {
	if h, err := activeHost(info.RunArgs); err != nil {
		return errors.Annotate(err, "set state working").Err()
	} else {
		h.State = tlw.WifiRouterHost_WORKING
	}
	return nil
}

func matchWifirouterBoardAndModelExec(ctx context.Context, info *execs.ExecInfo) error {
	if wifiRouterHost, err := activeHost(info.RunArgs); err != nil {
		return errors.Annotate(err, "match wifirouter board and model").Err()
	} else {
		argsMap := info.GetActionArgs(ctx)
		board := argsMap.AsString(ctx, "board", "")
		model := argsMap.AsString(ctx, "model", "")
		if (board == "" || board == wifiRouterHost.GetBoard()) && (model == "" || model == wifiRouterHost.GetModel()) {
			return nil
		}
	}
	return errors.Reason("wifirouter %q board model not matching %q", info.RunArgs.ResourceName, info.ActionArgs).Err()
}

// wifirouterPresentExec check if wifi router hosts exists
func wifirouterPresentExec(ctx context.Context, info *execs.ExecInfo) error {
	if d := info.RunArgs.DUT; d == nil || d.WifiRouterHosts == nil || len(d.WifiRouterHosts) == 0 {
		return errors.Reason("wifirouter host present: data is not present").Err()
	}
	return nil
}

func updatePeripheralWifiStateExec(ctx context.Context, info *execs.ExecInfo) error {
	pws := tlw.PeripheralWifiStateWorking
	for _, routerHost := range info.RunArgs.DUT.WifiRouterHosts {
		if routerHost.GetState() != tlw.WifiRouterHost_WORKING {
			pws = tlw.PeripheralWifiStateBroken
			break
		}
	}
	info.RunArgs.DUT.PeripheralWifiState = pws
	return nil
}

func init() {
	execs.Register("wifirouter_state_broken", setStateBrokenExec)
	execs.Register("wifirouter_state_working", setStateWorkingExec)
	execs.Register("is_wifirouter_board_model_matching", matchWifirouterBoardAndModelExec)
	execs.Register("wifi_router_host_present", wifirouterPresentExec)
	execs.Register("update_peripheral_wifi_state", updatePeripheralWifiStateExec)
}
