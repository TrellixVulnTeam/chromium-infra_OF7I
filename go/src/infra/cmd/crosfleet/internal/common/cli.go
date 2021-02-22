// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"flag"
	"fmt"

	"github.com/maruel/subcommands"
	"infra/cmd/crosfleet/internal/site"
)

// PrintCrosfleetUIPrompt prints a prompt for users to visit the go/my-crosfleet PLX
// to track their crosfleet-launched tasks.
func PrintCrosfleetUIPrompt(a subcommands.Application) {
	fmt.Fprintf(a.GetOut(), "Visit http://go/my-crosfleet to track all of your crosfleet-launched tasks.\n")
}

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

// ToKeyvalSlice converts a key-val map to a slice of "key:val" strings.
func ToKeyvalSlice(keyvals map[string]string) []string {
	var s []string
	for key, val := range keyvals {
		s = append(s, fmt.Sprintf("%s:%s", key, val))
	}
	return s
}
