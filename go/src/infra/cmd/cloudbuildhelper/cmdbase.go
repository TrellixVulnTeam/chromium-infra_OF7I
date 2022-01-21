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
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/maruel/subcommands"
	"golang.org/x/oauth2"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag/stringmapflag"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/system/signals"

	"infra/cmd/cloudbuildhelper/manifest"
)

// execCb a signature of a function that executes a subcommand.
type execCb func(ctx context.Context) error

// commandBase defines flags common to all subcommands.
type commandBase struct {
	subcommands.CommandRunBase

	exec       execCb     // called to actually execute the command
	extraFlags extraFlags // set in init
	posArgs    []*string  // will be filled in by positional arguments

	minVersion     string                // -cloudbuildhelper-min-version
	logConfig      logging.Config        // -log-* flags
	authFlags      authcli.Flags         // -auth-* flags
	infra          string                // -infra flag
	restrictions   manifest.Restrictions // -restrict-* flags
	canonicalTag   string                // -canonical-tag flag
	labels         stringmapflag.Value   // -label flags
	buildID        string                // -build-id flag
	jsonOutput     string                // -json-output flag
	renderToStdout string                // -render-to-stdout flag

	tempDirs []string // directories created via newTempDir
}

// extraFlags tells `commandBase.init` what additional CLI flags to register.
type extraFlags struct {
	auth         bool // -auth-* flags
	infra        bool // -infra flag
	restrictions bool // -restrict-* flags
	canonicalTag bool // -canonical-tag flag
	labels       bool // -label flags
	buildID      bool // -build-id flag
	jsonOutput   bool // -json-output and -render-to-stdout
}

// loadedManifest is returned by loadManifest.
type loadedManifest struct {
	Manifest   *manifest.Manifest
	Infra      *manifest.Infra
	CloudBuild *manifest.CloudBuildBuilder
}

// baseOutput is a shared portion of -json-output for `upload` and `build`
// commands.
type baseOutput struct {
	Name       string                  `json:"name"`                  // artifacts name from the manifest YAML
	ContextDir string                  `json:"context_dir,omitempty"` // absolute path to a context directory used to build the artifact
	Sources    []string                `json:"sources,omitempty"`     // absolute paths to directories declared as `sources` in the manifest YAML
	Notify     []manifest.NotifyConfig `json:"notify,omitempty"`      // copied from the manifest YAML
}

// init register base flags. Must be called.
func (c *commandBase) init(exec execCb, extraFlags extraFlags, posArgs []*string) {
	c.exec = exec
	c.extraFlags = extraFlags
	c.posArgs = posArgs

	c.Flags.StringVar(
		&c.minVersion, "cloudbuildhelper-min-version", "",
		"Min expected version of cloudbuildhelper tool.")

	c.logConfig.Level = logging.Info // default logging level
	c.logConfig.AddFlags(&c.Flags)

	if c.extraFlags.auth {
		c.authFlags.Register(&c.Flags, authOptions()) // see main.go
	}
	if c.extraFlags.infra {
		c.Flags.StringVar(&c.infra, "infra", "dev", "What section to pick from 'infra' field in the manifest YAML.")
	}
	if c.extraFlags.restrictions {
		c.restrictions.AddFlags(&c.Flags)
	}
	if c.extraFlags.canonicalTag {
		c.Flags.StringVar(&c.canonicalTag, "canonical-tag", "", "Tag to apply to an artifact if it's the first time we built it.")
	}
	if c.extraFlags.labels {
		c.Flags.Var(&c.labels, "label", "Labels to attach to the docker image, in k=v form.")
	}
	if c.extraFlags.buildID {
		c.Flags.StringVar(&c.buildID, "build-id", "", "Identifier of the CI build that calls this tool (used in various metadata).")
	}
	if c.extraFlags.jsonOutput {
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

	logging.Infof(ctx, "Starting %s", UserAgent)
	if c.minVersion != "" {
		if err := checkVersion(c.minVersion); err != nil {
			return handleErr(ctx, err)
		}
	}

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

	ctx, cancel := context.WithCancel(ctx)
	signals.HandleInterrupt(cancel)

	defer c.cleanup(ctx)
	if err := c.exec(ctx); err != nil {
		return handleErr(ctx, err)
	}
	return 0
}

// cleanup is called after the command execution.
func (c *commandBase) cleanup(ctx context.Context) {
	for _, p := range c.tempDirs {
		if err := os.RemoveAll(p); err != nil {
			logging.Warningf(ctx, "Leaking temp directory %s: %s", p, err)
		}
	}
	c.tempDirs = nil
}

// newTempDir returns a path to a new empty temp directory.
//
// It will be removed at the end of the command execution.
func (c *commandBase) newTempDir() (string, error) {
	path, err := ioutil.TempDir("", "cbh_")
	if err == nil {
		c.tempDirs = append(c.tempDirs, path)
	}
	return path, err
}

// loadManifest loads manifest from the given path, returning CLI errors.
//
// If the command requires `-infra` flag (as indicated by extraFlags in init),
// checks it was passed and picks the corresponding infra section from the
// manifest and checks it doesn't violate any of the restrictions supplied by
// `-restrict-*` flags. Populates Infra field of loadedManifest on success.
// If the command doesn't require `-infra` flag, the returned Infra is nil.
//
// If needCloudBuild is true, verifies Cloud Build configuration specified in
// the manifest and populates CloudBuild field of loadedManifest with the final
// resolved and validated configuration.
func (c *commandBase) loadManifest(ctx context.Context, path string, needStorage, needCloudBuild bool) (m *loadedManifest, output *baseOutput, err error) {
	m = &loadedManifest{}

	m.Manifest, err = manifest.Load(path)
	if err != nil {
		return nil, nil, errors.Annotate(err, "when loading manifest").Tag(isCLIError).Err()
	}

	// If contextdir isn't set, replace it with an empty temp directory.
	if m.Manifest.ContextDir == "" {
		m.Manifest.ContextDir, err = c.newTempDir()
		if err != nil {
			return nil, nil, errors.Annotate(err, "failed to create temp directory to act as a context dir").Err()
		}
	}

	// Now that all paths are initialized, render "${dir}/..." strings.
	if err = m.Manifest.Finalize(); err != nil {
		return nil, nil, errors.Annotate(err, "when loading manifest").Tag(isCLIError).Err()
	}

	if c.extraFlags.infra {
		if c.infra == "" {
			return nil, nil, errBadFlag("-infra", "a value is required")
		}
		section, ok := m.Manifest.Infra[c.infra]
		switch {
		case !ok:
			return nil, nil, errBadFlag("-infra", fmt.Sprintf("no %q infra specified in the manifest", c.infra))
		case needStorage && section.Storage == "":
			return nil, nil, errors.Reason("in %q: infra[...].storage is required when using remote build", path).Tag(isCLIError).Err()
		case needCloudBuild:
			m.CloudBuild, err = section.ResolveCloudBuildConfig(m.Manifest.CloudBuild)
			if err != nil {
				return nil, nil, errors.Annotate(err, "in %q", path).Err()
			}
		}
		m.Infra = &section
	}

	// Enforce restrictions specified via -restrict-* flags.
	if c.extraFlags.restrictions {
		var violations []string
		violations = append(violations, c.restrictions.CheckTargetName(m.Manifest.Name)...)
		violations = append(violations, c.restrictions.CheckBuildSteps(m.Manifest.Build)...)
		if c.extraFlags.infra {
			violations = append(violations, c.restrictions.CheckInfra(m.Infra)...)
		}
		if needCloudBuild {
			violations = append(violations, c.restrictions.CheckCloudBuild(m.CloudBuild)...)
		}
		if len(violations) != 0 {
			for _, msg := range violations {
				logging.Errorf(ctx, "Restriction violation: %s", msg)
			}
			return nil, nil, errors.Reason("restrictions violation detected, see logs").Err()
		}
	}

	// Prepare -json-output portion that depends on the manifest.
	if output, err = prepBaseOutput(m.Manifest, m.Infra); err != nil {
		return nil, nil, errors.Annotate(err, "failed to prepare values for -json-output").Err()
	}

	return
}

// prepBaseOutput populates baseOutput based on the manifest and -infra flag.
//
// Note: `infra` may be nil if -infra flag is optional and was omitted.
func prepBaseOutput(m *manifest.Manifest, infra *manifest.Infra) (*baseOutput, error) {
	contextDir := m.ContextDir
	if contextDir != "" {
		var err error
		if contextDir, err = filepath.Abs(contextDir); err != nil {
			return nil, errors.Annotate(err, "bad context directory path").Err()
		}
	}
	sources := make([]string, len(m.Sources))
	for i, path := range m.Sources {
		var err error
		if sources[i], err = filepath.Abs(path); err != nil {
			return nil, errors.Annotate(err, "bad path in `sources`").Err()
		}
	}
	var notify []manifest.NotifyConfig
	if infra != nil {
		notify = infra.Notify
	}
	return &baseOutput{
		Name:       m.Name,
		ContextDir: contextDir,
		Sources:    sources,
		Notify:     notify,
	}, nil
}

// tokenSource returns a source of OAuth2 tokens (based on CLI flags) or
// auth.ErrLoginRequired if the user needs to login first.
//
// This error is sniffed by Run(...) and converted into a comprehensible error
// message, so no need to handle it specially.
//
// Panics if the command was not configured to use auth in c.init(...).
func (c *commandBase) tokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if !c.extraFlags.auth {
		panic("auth flags weren't requested")
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
		switch {
		case cur[i] > min[i]:
			return nil
		case cur[i] < min[i]:
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
