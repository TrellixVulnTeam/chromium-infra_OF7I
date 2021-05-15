// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"

	"infra/cmd/skylab_swarming_worker/internal/event"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness"
	"infra/cros/dutstate"
)

type luciferResult struct {
	TestsFailed int
}

// runLuciferCommand runs a Lucifer exec.Cmd and processes Lucifer events.
func runLuciferCommand(ctx context.Context, cmd *exec.Cmd, dh *harness.DUTHarness, abortSock string) (*luciferResult, error) {
	log.Printf("Running %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	cmd.Stderr = os.Stderr

	af := event.ForwardAbortSignal(abortSock)
	defer af.Close()
	df := event.AbortWhenDone(ctx, abortSock)
	defer df()

	r := &luciferResult{}
	f := func(e event.Event, m string) {
		switch {
		case e == event.TestFailed && m != "autoserv":
			r.TestsFailed++
		case isHostStatus(e):
			s := hostStateUpdates[e]
			log.Printf("Got host event '%s', set host state to %s", e, s)
			dh.LocalState.HostState = s
		default:
		}
	}
	err := event.RunCommand(cmd, f)
	return r, err
}

// hostStateUpdates maps Events to the target runtime state of the
// host.  Host events that don't need to be handled are left as
// comment placeholders to aid cross-referencing.
var hostStateUpdates = map[event.Event]dutstate.State{
	event.HostClean:             dutstate.Ready,
	event.HostNeedsRepair:       dutstate.NeedsRepair,
	event.HostNeedsReset:        dutstate.NeedsReset,
	event.HostReady:             dutstate.Ready,
	event.HostFailedRepair:      dutstate.RepairFailed,
	event.HostNeedsDeploy:       dutstate.NeedsDeploy,
	event.HostNeedsManualRepair: dutstate.NeedsManualRepair,
	event.HostNeedsReplacement:  dutstate.NeedsReplacement,
}

func isHostStatus(e event.Event) bool {
	_, ok := hostStateUpdates[e]
	return ok
}
