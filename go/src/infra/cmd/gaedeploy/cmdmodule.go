// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag/stringmapflag"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/gaedeploy/gcloud"
	"infra/cmd/gaedeploy/module"
)

// Placeholders for some CLI flags that indicate they weren't set.
const (
	moduleNamePlaceholder    = "<name>"
	moduleVersionPlaceholder = "<version>"
	moduleYAMLPlaceholder    = "<path>"
)

var cmdModule = &subcommands.Command{
	UsageLine: "module [...]",
	ShortDesc: "deploys a single GAE module (aka service)",
	LongDesc: `Deploys a single GAE module (aka service).

Fetches and unpacks the tarball, reads and potentially modifies the module
YAML there (by injecting site-specific configuration provided via -var), and
then calls gcloud to actually deploy it:
	gcloud app deploy --project <app-id> --version <version> <yaml path>

Does nothing at all if such version (based on -version-name) already exists,
unless -force flag is passed.

Process the YAML before deployment by removing some unused deprecated fields
and by interpreting non-standard "luci_gae_vars" section which can be used to
parametrize the YAML. The section may look like this:

  luci_gae_vars:
    example-app-id-dev:
      AUTH_SERVICE_HOST: auth-service-dev.appspot.com
    example-app-id-prod:
      AUTH_SERVICE_HOST: auth-service-prod.appspot.com

Such variables can appear in the YAML (inside various values, but not keys)
as e.g. ${AUTH_SERVICE_HOST} and they'll be substituted with values provided via
e.g. "-var AUTH_SERVICE_HOST=..." CLI flag or, if there's no such flag, ones
specified in the "luci_gae_vars" section in the YAML.

It is recommended to put some sample values in the YAML (to act as a
documentation) and store real production configuration elsewhere, and provide it
to gaedeploy dynamically via -var flags.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdModuleRun{}
		c.init()
		return c
	},
}

type cmdModuleRun struct {
	commandBase

	moduleName    string              // -module-name flag, required
	moduleYAML    string              // -module-yaml flag, require
	moduleVersion string              // -module-version flag, required
	vars          stringmapflag.Value // -var flags
	force         bool                // -force flag
}

func (c *cmdModuleRun) init() {
	c.commandBase.init(c.exec, extraFlags{
		appID:    true,
		tarball:  true,
		cacheDir: true,
		dryRun:   true,
	})
	c.Flags.StringVar(&c.moduleName, "module-name", moduleNamePlaceholder,
		"Name of the module to deploy (must match what's in the YAML).")
	c.Flags.StringVar(&c.moduleYAML, "module-yaml", moduleYAMLPlaceholder,
		"Path within the tarball to a module YAML to deploy.")
	c.Flags.StringVar(&c.moduleVersion, "module-version", moduleVersionPlaceholder,
		"Version name for the deployed code. Does nothing if such version already exists, unless -force is also given.")
	c.Flags.Var(&c.vars, "var", "A KEY=VALUE pair that defines a variable used when rendering module's YAML. May be repeated.")
	c.Flags.BoolVar(&c.force, "force", false,
		"Deploy the module even if such version already exists")
}

func (c *cmdModuleRun) exec(ctx context.Context) error {
	switch {
	case c.moduleName == moduleNamePlaceholder:
		return errBadFlag("-module-name", "a value is required")
	case c.moduleYAML == moduleYAMLPlaceholder:
		return errBadFlag("-module-yaml", "a value is required")
	case c.moduleVersion == moduleVersionPlaceholder:
		return errBadFlag("-module-version", "a value is required")
	}

	logging.Infof(ctx, "App ID:  %s", c.appID)
	logging.Infof(ctx, "Tarball: %s", c.tarballSource)
	logging.Infof(ctx, "Cache:   %s", c.cacheDir)
	logging.Infof(ctx, "Module:  %s", c.moduleName)
	logging.Infof(ctx, "YAML:    %s", c.moduleYAML)
	logging.Infof(ctx, "Version: %s", c.moduleVersion)

	if !c.force {
		logging.Infof(ctx, "Checking if such version already exists...")
		mods, err := gcloud.List(ctx, c.appID, c.moduleName)
		if err != nil {
			return errors.Annotate(err, "failed to check whether such version already exists").Err()
		}
		if _, yes := mods[c.moduleName][c.moduleVersion]; yes {
			logging.Infof(ctx, "Version %q of %q already exists, skipping the deployment!", c.moduleVersion, c.moduleName)
			return nil
		}
		logging.Infof(ctx, "No such version, will deploy it.")
	}

	return c.cache.WithTarball(ctx, c.source, func(path string) error {
		// Read the original YAML to inject site-specific configuration into it.
		logging.Infof(ctx, "Loading %s...", filepath.Join(path, c.moduleYAML))
		mod, err := module.ReadYAML(filepath.Join(path, c.moduleYAML))
		if err != nil {
			return errors.Annotate(err, "failed to read module YAML").Err()
		}
		if mod.Name != c.moduleName {
			return errors.Reason("module name in the yaml %q doesn't match -module-name flag %q", mod.Name, c.moduleName).Err()
		}

		// Convert it to something that gcloud actually understands.
		consumedVars, err := mod.Process(c.appID, map[string]string(c.vars))
		if err != nil {
			return errors.Annotate(err, "failed to process module's config").Err()
		}

		// Pretty print the final YAML to the console.
		blob, err := mod.DumpYAML()
		if err != nil {
			return errors.Annotate(err, "failed to serialize processed module config").Err()
		}
		logging.Infof(ctx, "Processed module YAML:\n\n%s\n", blob)

		// Loudly warn about supplied but unused variables.
		sortedVars := make([]string, 0, len(c.vars))
		for key := range c.vars {
			sortedVars = append(sortedVars, key)
		}
		sort.Strings(sortedVars)
		for _, key := range sortedVars {
			if !consumedVars.Has(key) {
				logging.Warningf(ctx, "Variable %q was passed via -var flag but not referenced in the YAML", key)
			}
		}

		// Need to save the YAML on disk in the same directory as the original one,
		// so that gcloud resolves all paths in it correctly. Keep it hanging there
		// afterwards to aid in debugging, it is harmless.
		dir, base := filepath.Dir(c.moduleYAML), filepath.Base(c.moduleYAML)
		modPath := filepath.Join(dir, ".gaedeploy_"+base)
		if err := ioutil.WriteFile(filepath.Join(path, modPath), blob, 0600); err != nil {
			return errors.Annotate(err, "failed to save processed module config").Err()
		}

		// Perform the actual deployment.
		return gcloud.Run(ctx, []string{
			"app", "deploy",
			"--project", c.appID,
			"--quiet", // disable interactive prompts
			"--no-promote",
			"--no-stop-previous-version",
			"--version", c.moduleVersion,
			modPath,
		}, path, c.dryRun)
	})
}
