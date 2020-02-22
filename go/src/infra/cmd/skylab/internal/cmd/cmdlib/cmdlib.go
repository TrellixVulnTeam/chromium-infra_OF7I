// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmdlib

import (
	"flag"
	"strings"
	"time"

	"infra/cmd/skylab/internal/site"

	lflag "go.chromium.org/luci/common/flag"
)

// DefaultTaskPriority is the default priority for a swarming task.
var DefaultTaskPriority = 140

type commonFlags struct {
	debug bool
}

// EnvFlags controls selection of the environment: either prod (default) or dev.
type EnvFlags struct {
	dev bool
}

// Register sets up the -dev argument.
func (f *EnvFlags) Register(fl *flag.FlagSet) {
	fl.BoolVar(&f.dev, "dev", false, "Run in dev environment.")
}

// Env returns the environment, either dev or prod.
func (f EnvFlags) Env() site.Environment {
	if f.dev {
		return site.Dev
	}
	return site.Prod
}

// RemovalReason is the reason that a DUT has been removed from the inventory.
// Removal requires a buganizer or monorail bug and possibly a comment and
// expiration time.
type RemovalReason struct {
	Bug     string
	Comment string
	Expire  time.Time
}

// Register sets up the command line arguments for specifying a removal reason.
func (rr *RemovalReason) Register(f *flag.FlagSet) {
	f.StringVar(&rr.Bug, "bug", "", "Bug link for why DUT is being removed.  Required.")
	f.StringVar(&rr.Comment, "comment", "", "Short comment about why DUT is being removed.")
	f.Var(lflag.RelativeTime{T: &rr.Expire}, "expires-in", "Expire removal reason in `days`.")
}

// FixSuspiciousHostname checks whether a hostname is suspicious and potentially indicates a bug of some kind.
// The prefix "crossk-" is suspicious and should be removed. The suffix ".cros" is also suspicious and should be removed.
func FixSuspiciousHostname(hostname string) string {
	if strings.HasPrefix(hostname, "crossk-") {
		return strings.TrimPrefix(hostname, "crossk-")
	}
	if strings.HasSuffix(hostname, ".cros") {
		return strings.TrimSuffix(hostname, ".cros")
	}
	return hostname
}
