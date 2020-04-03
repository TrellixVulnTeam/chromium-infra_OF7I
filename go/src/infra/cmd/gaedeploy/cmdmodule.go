// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag/stringmapflag"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/system/environ"

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
	logging.Infof(ctx, "Cache:   %s", c.cache.Root)
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

	return c.cache.WithTarball(ctx, c.source, func(root string) error {
		// Read the original YAML to inject site-specific configuration into it.
		logging.Infof(ctx, "Loading %s...", filepath.Join(root, c.moduleYAML))
		mod, err := module.ReadYAML(filepath.Join(root, c.moduleYAML))
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
		modDir, yamlBaseName := filepath.Dir(c.moduleYAML), filepath.Base(c.moduleYAML)
		yamlName := ".gaedeploy_" + yamlBaseName
		if err := ioutil.WriteFile(filepath.Join(root, filepath.Join(modDir, yamlName)), blob, 0600); err != nil {
			return errors.Annotate(err, "failed to save processed module config").Err()
		}

		// If this is a tarball with Go code, need to setup GOPATH and deploy
		// from within it to make sure *.go paths in GAE app's stack traces are
		// correct.
		var env environ.Env
		if strings.HasPrefix(mod.Runtime, "go") {
			if modDir, env, err = prepareForGoDeploy(ctx, root, modDir); err != nil {
				return errors.Annotate(err, "failed to prepare for Go deployment").Err()
			}
		}

		// Perform the actual deployment.
		return gcloud.Run(ctx, []string{
			"app", "deploy",
			"--project", c.appID,
			"--quiet", // disable interactive prompts
			"--no-promote",
			"--no-stop-previous-version",
			"--version", c.moduleVersion,
			yamlName,
		}, filepath.Join(root, modDir), env, c.dryRun)
	})
}

// prepareForGoDeploy prepares Go environment variables and finds the module
// in GOPATH.
//
// `root` is a path to where the tarball is checked out.
// `modDir` is a path within the tarball to the directory with module's YAML.
//
// Uses the presence of "<root>/_gopath" as indicator that the tarball was
// built by cloudbuildhelper (using "go_gae_bundle" build step). If it's
// absent, assumes the tarball uses Go modules and lets "gcloud app deploy"
// deal with it.
//
// Returns:
//   `newModDir`: a path within the tarball to use as new "directory with
//       module's YAML" (may be same as `modDir` if no changes are needed).
//   `env`: a environ to pass to "gcloud app deploy".
//   `err`: if something is not right.
func prepareForGoDeploy(ctx context.Context, root, modDir string) (newModDir string, env environ.Env, err error) {
	// Scrub the existing Go environ. This scrubs a bit more, but gcloud should
	// not depend on env vars that start with GO or CGO anyway.
	env = environ.System()
	env.RemoveMatch(func(k, v string) bool {
		return strings.HasPrefix(k, "GO") || strings.HasPrefix(k, "CGO")
	})

	// Setup GOPATH if the tarball has "_gopath" directory.
	goPathAbs, err := filepath.Abs(filepath.Join(root, "_gopath"))
	if err != nil {
		return "", nil, err
	}
	if _, err := os.Stat(goPathAbs); err == nil {
		logging.Infof(ctx, "Found _gopath, using it as GOPATH")
		env.Set("GOPATH", goPathAbs)
		env.Set("GO111MODULE", "off")
	}

	// Detect when `modDir` is a symlink to a _gopath/... and follow it. This is
	// how tarballs built by cloudbuildhelper look like. By following the symlink
	// we make the deployed *.go files have paths matching their absolute import
	// paths. They eventually surface in stack traces in error messages, etc.
	dest, err := filepath.EvalSymlinks(filepath.Join(root, modDir))
	if err != nil {
		return "", nil, errors.Annotate(err, "failed to evaluate %q as a symlink", modDir).Err()
	}
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return "", nil, errors.Annotate(err, "failed to calculate rel(%q, %q)", root, dest).Err()
	}
	if strings.HasPrefix(rel, filepath.Join("_gopath", "src")+string(filepath.Separator)) {
		logging.Infof(ctx, "Following symlink %q to its destination in _gopath %q", modDir, rel)
		return rel, env, nil
	}

	// Not a cloudbuildhelper tarball, feed the module directory to
	// "gcloud app deploy" as is. This can potentially work with apps that use
	// go.mod but it hasn't been tested.
	return modDir, env, nil
}
