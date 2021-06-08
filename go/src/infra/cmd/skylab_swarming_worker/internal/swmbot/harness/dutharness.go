// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package harness

import (
	"context"
	"log"

	"go.chromium.org/luci/common/errors"

	"infra/libs/skylab/inventory"

	"infra/libs/skylab/autotest/hostinfo"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	h_hostinfo "infra/cmd/skylab_swarming_worker/internal/swmbot/harness/hostinfo"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/labelupdater"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/localdutinfo"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/resultsdir"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/ufsdutinfo"
)

// DUTHarness holds information about a DUT's harness
type DUTHarness struct {
	BotInfo      *swmbot.Info
	DUTID        string
	DUTHostname  string
	ResultsDir   string
	LocalState   *swmbot.LocalDUTState
	labelUpdater *labelupdater.LabelUpdater
	// err tracks errors during setup to simplify error handling logic.
	err     error
	closers []closer
}

// Close closes and flushes out the harness resources.  This is safe
// to call multiple times.
func (dh *DUTHarness) Close(ctx context.Context) error {
	log.Printf("Wrapping up harness for %s", dh.DUTHostname)
	var errs []error
	for n := len(dh.closers) - 1; n >= 0; n-- {
		if err := dh.closers[n].Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Annotate(errors.MultiError(errs), "close harness").Err()
	}
	return nil
}

func makeDUTHarness(b *swmbot.Info) *DUTHarness {
	return &DUTHarness{
		BotInfo: b,
		labelUpdater: &labelupdater.LabelUpdater{
			BotInfo: b,
		},
	}
}

func (dh *DUTHarness) loadLocalDUTInfo(ctx context.Context) {
	if dh.err != nil {
		return
	}
	if dh.DUTHostname == "" {
		dh.err = errors.Reason("DUTHostname cannot be blank").Err()
		return
	}
	ldi, err := localdutinfo.Open(ctx, dh.BotInfo, dh.DUTHostname)
	if err != nil {
		dh.err = err
		return
	}
	dh.closers = append(dh.closers, ldi)
	dh.LocalState = &ldi.LocalDUTState
}

func (dh *DUTHarness) loadUFSDUTInfo(ctx context.Context) (*inventory.DeviceUnderTest, map[string]string) {
	if dh.err != nil {
		return nil, nil
	}
	var s *ufsdutinfo.Store
	if dh.DUTID != "" {
		s, dh.err = ufsdutinfo.LoadByID(ctx, dh.BotInfo, dh.DUTID, dh.labelUpdater.Update)
	} else if dh.DUTHostname != "" {
		s, dh.err = ufsdutinfo.LoadByHostname(ctx, dh.BotInfo, dh.DUTHostname, dh.labelUpdater.Update)
	} else {
		dh.err = errors.Reason("Both DUTID and DUTHostname field is empty.").Err()
	}
	if dh.err != nil {
		return nil, nil
	}
	// We overwrite both DUTHostname and DUTID based on UFS data because in
	// single DUT tasks we don't have DUTHostname when we start, and in the
	// scheduling_unit (multi-DUT) tasks we don't have DUTID when we start.
	dh.DUTHostname = s.DUT.GetCommon().GetHostname()
	dh.DUTID = s.DUT.GetCommon().GetId()
	dh.closers = append(dh.closers, s)
	return s.DUT, s.StableVersions
}

func (dh *DUTHarness) makeHostInfo(d *inventory.DeviceUnderTest, stableVersion map[string]string) *hostinfo.HostInfo {
	if dh.err != nil {
		return nil
	}
	hip := h_hostinfo.FromDUT(d, stableVersion)
	dh.closers = append(dh.closers, hip)
	return hip.HostInfo
}

func (dh *DUTHarness) addLocalStateToHostInfo(hi *hostinfo.HostInfo) {
	if dh.err != nil {
		return
	}
	hib := h_hostinfo.BorrowLocalDUTState(hi, dh.LocalState)
	dh.closers = append(dh.closers, hib)
}

func (dh *DUTHarness) makeDUTResultsDir(d *resultsdir.Dir) {
	if dh.err != nil {
		return
	}
	path, err := d.OpenSubDir(dh.DUTHostname)
	if err != nil {
		dh.err = err
		return
	}
	log.Printf("Created DUT level results sub-dir %s", path)
	dh.ResultsDir = path
}

func (dh *DUTHarness) exposeHostInfo(hi *hostinfo.HostInfo) {
	if dh.err != nil {
		return
	}
	hif, err := h_hostinfo.Expose(hi, dh.ResultsDir, dh.DUTHostname)
	if err != nil {
		dh.err = err
		return
	}
	dh.closers = append(dh.closers, hif)
}
