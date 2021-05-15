// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package harness manages the setup and teardown of various Swarming
// bot resources for running lab tasks, like results directories and
// host info.
package harness

import (
	"context"
	"log"

	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/resultsdir"
)

// closer interface to wrap Close method with providing context.
type closer interface {
	Close(ctx context.Context) error
}

// Info holds information about the Swarming bot harness.
type Info struct {
	*swmbot.Info

	TaskResultsDir *resultsdir.Dir
	DUTs           []*DUTHarness
	closers        []closer
}

// Close closes and flushes out the harness resources.  This is safe
// to call multiple times.
func (i *Info) Close(ctx context.Context) error {
	var errs []error
	for _, dh := range i.DUTs {
		if err := dh.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	for n := len(i.closers) - 1; n >= 0; n-- {
		if err := i.closers[n].Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Annotate(errors.MultiError(errs), "close harness").Err()
	}
	return nil
}

// Open opens and sets up the bot and task harness needed for Autotest
// jobs.  An Info struct is returned with necessary fields, which must
// be closed.
func Open(ctx context.Context, b *swmbot.Info, o ...Option) (i *Info, err error) {
	i = &Info{
		Info: b,
	}
	defer func(i *Info) {
		if err != nil {
			_ = i.Close(ctx)
		}
	}(i)
	// Make result dir for swarming bot, which will be uploaded to GS once the
	// task completes.
	if err := i.makeTaskResultsDir(); err != nil {
		return nil, errors.Annotate(err, "create task result directory").Err()
	}
	i.loadDUTHarness(ctx)
	for _, dh := range i.DUTs {
		for _, o := range o {
			// There could be options that configure only Info,
			// so only use the options that are specifically for DUT harness.
			if o, ok := o.(dutHarnessOption); ok {
				o.configureDutHarness(dh)
			}
		}
		// Load DUT's info(e.g. labels, attributes, stable_versions) from UFS/inventory.
		d, sv := dh.loadDUTInfo(ctx)
		// Load DUT's info from bot state file on drone.
		dh.loadLocalState(ctx)
		// Convert DUT's info into autotest/labpack friendly format, a.k.a host_info_store.
		hi := dh.makeHostInfo(d, sv)
		dh.addBotInfoToHostInfo(hi)
		// Make a sub-dir for each DUT, which will be consumed by lucifer later.
		dh.makeDUTResultsDir(i.TaskResultsDir)
		// Copying host_info_store file into DUT's result dir.
		dh.exposeHostInfo(hi)
		if dh.err != nil {
			return nil, errors.Annotate(dh.err, "open DUTharness").Err()
		}
	}
	return i, nil
}

func (i *Info) makeTaskResultsDir() error {
	path := i.Info.ResultsDir()
	rd, err := resultsdir.Open(path)
	if err != nil {
		return err
	}
	log.Printf("Created task results directory %s", path)
	i.closers = append(i.closers, rd)
	i.TaskResultsDir = rd
	return nil
}

func (i *Info) loadDUTHarness(ctx context.Context) {
	d := makeDUTHarness(i.Info)
	d.DUTID = i.Info.DUTID
	i.DUTs = append(i.DUTs, d)
}
