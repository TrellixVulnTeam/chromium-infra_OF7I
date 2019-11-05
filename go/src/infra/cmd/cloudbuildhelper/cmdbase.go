// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/maruel/subcommands"
	"golang.org/x/oauth2"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/system/signals"
)

// execCb a signature of a function that executes a subcommand.
type execCb func(ctx context.Context) error

// commandBase defines flags common to all subcommands.
type commandBase struct {
	subcommands.CommandRunBase

	exec     execCb    // called to actually execute the command
	needAuth bool      // set in init, true if we have auth flags registered
	posArgs  []*string // will be filled in by positional arguments

	minVersion     string         // -cloudbuildhelper-min-version
	logConfig      logging.Config // -log-* flags
	authFlags      authcli.Flags  // -auth-* flags
	jsonOutput     string         // -json-output flag
	renderToStdout string         // -render-to-stdout flag
}

// init register base flags. Must be called.
func (c *commandBase) init(exec execCb, needAuth, needJSONOutput bool, posArgs []*string) {
	c.exec = exec
	c.needAuth = needAuth
	c.posArgs = posArgs

	c.Flags.StringVar(
		&c.minVersion, "cloudbuildhelper-min-version", "",
		"Min expected version of cloudbuildhelper tool")

	c.logConfig.Level = logging.Info // default logging level
	c.logConfig.AddFlags(&c.Flags)

	if c.needAuth {
		c.authFlags.Register(&c.Flags, authOptions()) // see main.go
	}
	if needJSONOutput {
		c.Flags.StringVar(&c.jsonOutput, "json-output", "", "Where to write JSON file with the outcome (\"-\" for stdout).")
		c.Flags.StringVar(&c.renderToStdout, "render-to-stdout", "", "Text template with fields to print to stdout. It takes -json-output JSON as an input.")
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

	if len(args) != len(c.posArgs) {
		if len(c.posArgs) == 0 {
			return handleErr(ctx, errors.Reason("unexpected positional arguments %q", args).Tag(isCLIError).Err())
		}
		return handleErr(ctx, errors.Reason(
			"expected %d positional argument(s), got %d",
			len(c.posArgs), len(args)).Tag(isCLIError).Err())
	}

	for i, arg := range args {
		*c.posArgs[i] = arg
	}

	// For now just make sure we can compile it.
	if c.renderToStdout != "" {
		if _, err := template.New("").Parse(c.renderToStdout); err != nil {
			return handleErr(ctx, errBadFlag("-render-to-stdout", err.Error()))
		}
		if c.jsonOutput == "-" {
			return handleErr(ctx, errors.Reason("-render-to-stdout and -json-output='-' can't be used together").Tag(isCLIError).Err())
		}
	}

	if c.minVersion != "" {
		if err := checkVersion(c.minVersion); err != nil {
			return handleErr(ctx, err)
		}
	}
	logging.Infof(ctx, "Starting %s", UserAgent)

	ctx, cancel := context.WithCancel(ctx)
	signals.HandleInterrupt(cancel)

	if err := c.exec(ctx); err != nil {
		return handleErr(ctx, err)
	}
	return 0
}

// tokenSource returns a source of OAuth2 tokens (based on CLI flags) or
// auth.ErrLoginRequired if the user needs to login first.
//
// This error is sniffed by Run(...) and converted into a comprehensible error
// message, so no need to handle it specially.
//
// Panics if the command was not configured to use auth in c.init(...).
func (c *commandBase) tokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if !c.needAuth {
		panic("needAuth is false")
	}
	opts, err := c.authFlags.Options()
	if err != nil {
		return nil, errors.Annotate(err, "bad auth options").Tag(isCLIError).Err()
	}
	authn := auth.NewAuthenticator(ctx, auth.SilentLogin, opts)
	if email, err := authn.GetEmail(); err == nil {
		logging.Infof(ctx, "Running as %s", email)
	}
	return authn.TokenSource()
}

// writeJSONOutput writes the result to -json-output file (if was given).
//
// Handles -render-to-stdout as well.
func (c *commandBase) writeJSONOutput(r interface{}) error {
	// Need to round-trip though JSON to "activate" all `json:...` annotations.
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return errors.Annotate(err, "failed to marshal to JSON: %v", r).Err()
	}
	var asMap interface{}
	if err := json.Unmarshal(b, &asMap); err != nil {
		return errors.Annotate(err, "generated bad JSON output").Err()
	}

	if c.renderToStdout != "" {
		// We already verified it can be compiled in Run.
		tmpl := template.Must(template.New("").Parse(c.renderToStdout))

		// Render it.
		buf := bytes.Buffer{}
		if err = tmpl.Execute(&buf, asMap); err != nil {
			return errors.Annotate(err, "failed to render %q", c.renderToStdout).Err()
		}
		buf.WriteRune('\n')

		// Emit it.
		if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
			return errors.Annotate(err, "failed to write to stdout").Err()
		}
	}

	switch c.jsonOutput {
	case "":
		return nil
	case "-":
		fmt.Printf("%s\n", b)
		return nil
	default:
		return errors.Annotate(ioutil.WriteFile(c.jsonOutput, b, 0600), "failed to write %q", c.jsonOutput).Err()
	}
}

// checkVersion returns an error if the version of cloudbuildhelper is older
// than minVer.
func checkVersion(minVer string) error {
	min, err := parseVersion(minVer)
	if err != nil {
		return errBadFlag("-cloudbuildhelper-min-version", err.Error())
	}
	cur, err := parseVersion(Version)
	if err != nil {
		panic("impossible")
	}
	for i := range min {
		if cur[i] < min[i] {
			return errors.Reason("the caller wants cloudbuildhelper >=v%s but the running executable is v%s", minVer, Version).Tag(isCLIError).Err()
		}
	}
	return nil
}

// parseVersion parses "a.b.c" into [a, b, c] slice.
func parseVersion(v string) (out [3]int64, err error) {
	chunks := strings.Split(v, ".")
	if len(chunks) > 3 {
		err = fmt.Errorf("version string is allowed to have at most 3 components, got %d", len(chunks))
	} else {
		for i, s := range chunks {
			if out[i], err = strconv.ParseInt(s, 10, 32); err != nil {
				err = fmt.Errorf("bad version string %q", v)
				break
			}
		}
	}
	return
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
		return 4
	case errors.Contains(err, auth.ErrLoginRequired):
		fmt.Fprintf(os.Stderr, "Need to login first by running:\n  $ %s login\n", os.Args[0])
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
