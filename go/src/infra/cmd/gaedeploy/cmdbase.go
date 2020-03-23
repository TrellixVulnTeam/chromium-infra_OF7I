// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/system/signals"

	"infra/cmd/gaedeploy/cache"
	"infra/cmd/gaedeploy/source"
)

// execCb a signature of a function that executes a subcommand.
type execCb func(ctx context.Context) error

// Placeholders for some CLI flags that indicate they weren't set.
const (
	appIDPlaceholder         = "<app-id>"
	tarballSourcePlaceholder = "<path>"
)

// commandBase defines flags common to all subcommands.
type commandBase struct {
	subcommands.CommandRunBase

	extraFlags extraFlags // as passed to init(...)
	exec       execCb     // called to actually execute the command

	logConfig     logging.Config // -log-* flags
	appID         string         // -app-id flag (required)
	tarballSource string         // -tarball-source flag (required)
	tarballSHA256 string         // -tarball-sha256 flag (optional for local files)
	cacheDir      string         // -cache-dir flag (optional, has default)
	dryRun        bool           // -dry-run flag

	source source.Source // initialized in handleArgsAndFlags
	cache  *cache.Cache  // initialized in handleArgsAndFlags
}

// extraFlags indicates what CLI flags to register.
type extraFlags struct {
	appID    bool // -app-id
	tarball  bool // -tarball-*
	cacheDir bool // -cache-dir
	dryRun   bool // -dry-run
}

// init register base flags. Must be called.
func (c *commandBase) init(exec execCb, extraFlags extraFlags) {
	c.extraFlags = extraFlags
	c.exec = exec

	c.logConfig.Level = logging.Info // default logging level
	c.logConfig.AddFlags(&c.Flags)

	if extraFlags.appID {
		c.Flags.StringVar(&c.appID, "app-id", appIDPlaceholder, "GAE app ID to update.")
	}
	if extraFlags.tarball {
		c.Flags.StringVar(&c.tarballSource, "tarball-source", tarballSourcePlaceholder, "Either gs:// or local path to a tarball with app source code.")
		c.Flags.StringVar(&c.tarballSHA256, "tarball-sha256", "", "The expected tarball's SHA256 (optional for local files).")
	}
	if extraFlags.cacheDir {
		c.Flags.StringVar(&c.cacheDir, "cache-dir", "", "Directory to keep unpacked tarballs in.")
	}
	if extraFlags.dryRun {
		c.Flags.BoolVar(&c.dryRun, "dry-run", false, "Just print gcloud commands without executing them.")
	}
}

// ModifyContext implements cli.ContextModificator.
//
// Used by cli.Application.
func (c *commandBase) ModifyContext(ctx context.Context) context.Context {
	return c.logConfig.Set(ctx)
}

// Run implements the subcommands.CommandRun interface.
func (c *commandBase) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)

	logging.Infof(ctx, "Starting %s", UserAgent)

	if err := c.handleArgsAndFlags(args); err != nil {
		return handleErr(ctx, isCLIError.Apply(err))
	}

	ctx, cancel := context.WithCancel(ctx)
	signals.HandleInterrupt(cancel)

	if err := c.exec(ctx); err != nil {
		return handleErr(ctx, err)
	}
	return 0
}

// handleArgsAndFlags validates flags and substitutes defaults.
func (c *commandBase) handleArgsAndFlags(args []string) error {
	switch {
	case len(args) != 0:
		return errors.Reason("unexpected positional arguments %q", args).Err()
	case c.extraFlags.appID && c.appID == appIDPlaceholder:
		return errBadFlag("-app-id", "a value is required")
	case c.extraFlags.tarball && c.tarballSource == tarballSourcePlaceholder:
		return errBadFlag("-tarball-source", "a value is required")
	}

	// Where to grab the tarball from.
	if c.extraFlags.tarball {
		var err error
		c.source, err = source.New(c.tarballSource, c.tarballSHA256)
		if err != nil {
			return err
		}
	}

	// Where to store it.
	if c.extraFlags.cacheDir {
		if c.cacheDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return errors.Annotate(err, "failed to determine the home dir, pass -cache-dir directly").Err()
			}
			c.cacheDir = filepath.Join(home, ".gaedeploy_cache")
		}
		if err := os.MkdirAll(c.cacheDir, 0700); err != nil {
			return errors.Annotate(err, "failed to create the cache directory").Err()
		}
		c.cache = &cache.Cache{Root: c.cacheDir}
	}

	return nil
}

// isCLIError is tagged into errors caused by bad CLI flags.
var isCLIError = errors.BoolTag{Key: errors.NewTagKey("bad CLI invocation")}

// errBadFlag produces an error related to malformed or absent CLI flag
func errBadFlag(flag, msg string) error {
	return errors.Reason("bad %q: %s", flag, msg).Tag(isCLIError).Err()
}

// handleErr prints the error and returns the process exit code.
func handleErr(ctx context.Context, err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Contains(err, context.Canceled): // happens on Ctrl+C
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 3
	case isCLIError.In(err):
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		return 2
	default:
		logging.Errorf(ctx, "%s", err)
		logging.Errorf(ctx, "Full context:")
		errors.Log(ctx, err)
		return 1
	}
}
