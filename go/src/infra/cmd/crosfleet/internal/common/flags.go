// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"flag"
	"github.com/maruel/subcommands"
	"infra/cmd/crosfleet/internal/site"
)

// Flags contains flags common to all crosfleet commands.
type Flags struct {
	subcommands.CommandRunBase
}

// Init initializes the flags.
func (c *Flags) Init() {
}

// Parse parses the flags.
func (c *Flags) Parse() error {
	return nil
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
