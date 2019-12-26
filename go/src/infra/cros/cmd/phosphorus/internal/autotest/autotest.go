// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package autotest provides a Go API for interacting with Autotest.
//
// This package provides a very low level API with no business logic.
// This is to keep the bug surface small and keep the business logic
// clearly separate.
package autotest

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const autoservRelpath = "server/autoserv"

const offloadPath = "LOGDATA_DIR"

// AutoservArgs is the arguments for creating an autoserv command.
type AutoservArgs struct {
	// Args is split with shlex.split by autoserv.
	Args               string
	Cleanup            bool
	ClientTest         bool
	CollectCrashinfo   bool
	ControlName        string
	ExecutionTag       string
	HostInfoSubDir     string
	Hosts              []string
	JobLabels          []string
	JobName            string
	JobOwner           string
	Lab                bool
	LocalOnlyHostInfo  bool
	NoTee              bool
	OffloadDir         string
	ParentJobID        int
	Provision          bool
	Repair             bool
	RequireSSP         bool
	Reset              bool
	ResultsDir         string
	TestSourceBuild    string
	UseExistingResults bool
	Verbose            bool
	Verify             bool
	VerifyJobRepoURL   bool
	WritePidfile       bool

	ControlFile string
}

// AutoservCommand returns the Cmd struct to execute autoserv with the
// given arguments.
func AutoservCommand(c Config, cmd *AutoservArgs) *exec.Cmd {
	args := make([]string, 0, 20)
	if cmd.Args != "" {
		args = append(args, "--args", cmd.Args)
	}
	if cmd.Cleanup {
		args = append(args, "--cleanup")
	}
	if cmd.ClientTest {
		args = append(args, "-c")
	} else {
		// This is only used to check that it is not passed along with -c.
		args = append(args, "-s")
	}
	if cmd.CollectCrashinfo {
		args = append(args, "--collect-crashinfo")
	}
	if cmd.ControlName != "" {
		args = append(args, "--control-name", cmd.ControlName)
	}
	if cmd.ExecutionTag != "" {
		args = append(args, "-P", cmd.ExecutionTag)
	}
	if cmd.HostInfoSubDir != "" {
		args = append(args, "--host-info-subdir", cmd.HostInfoSubDir)
	}
	if len(cmd.Hosts) != 0 {
		args = append(args, "-m", strings.Join(cmd.Hosts, ","))
	}
	if len(cmd.JobLabels) != 0 {
		args = append(args, "--job-labels", strings.Join(cmd.JobLabels, ","))
	}
	if cmd.JobName != "" {
		args = append(args, "-l", cmd.JobName)
	}
	if cmd.JobOwner != "" {
		args = append(args, "-u", cmd.JobOwner)
	}
	if cmd.Lab {
		args = append(args, "--lab", "True")
	}
	if cmd.LocalOnlyHostInfo {
		// autoserv bool args require values.
		args = append(args, "--local-only-host-info", "True")
	}
	if cmd.NoTee {
		args = append(args, "-n")
	}
	if cmd.OffloadDir != "" {
		c.Environment[offloadPath] = cmd.OffloadDir
	}
	if cmd.ParentJobID != 0 {
		args = append(args, fmt.Sprintf("--parent_job_id=%d", cmd.ParentJobID))
	}
	if cmd.Provision {
		args = append(args, "--provision")
	}
	if cmd.Repair {
		args = append(args, "-R")
	}
	if cmd.RequireSSP {
		args = append(args, "--require-ssp")
	}
	if cmd.Reset {
		args = append(args, "--reset")
	}
	if cmd.ResultsDir != "" {
		args = append(args, "-r", cmd.ResultsDir)
	}
	if cmd.TestSourceBuild != "" {
		args = append(args, "--test_source_build", cmd.TestSourceBuild)
	}
	if cmd.UseExistingResults {
		args = append(args, "--use-existing-results")
	}
	if cmd.Verbose {
		args = append(args, "--verbose")
	}
	if cmd.Verify {
		args = append(args, "-v")
	}
	if cmd.VerifyJobRepoURL {
		args = append(args, "--verify_job_repo_url")
	}
	if cmd.WritePidfile {
		args = append(args, "-p")
	}

	// The reason for the following behavior is due to autoserv argument parsing.
	// autoserv will call shlex.split on the --args value and
	// append those onto the positional parameters.
	// Then, autoserv takes the first positional parameter as the control file name.
	// For all of the following, autoserv uses "foo" as the control file:
	//
	//   autoserv [...] --args "foo bar"
	//   autoserv [...] -- foo bar
	//   autoserv [...] --args "spam eggs" -- foo bar
	//
	// Thus, if ControlFile is unset but Args is Set, we need to
	// prepend a placeholder control file.
	if cmd.ControlFile != "" || cmd.Args != "" {
		args = append(args, cmd.ControlFile)
	}
	return command(c, autoservRelpath, args...)
}

const tkoRelpath = "tko/parse"

// A.k.a. "SKYLAB_PROVISION" in older code.
const skylabResultsDirNestingLevel = "3"

// ParseCommand returns the Cmd struct to execute tko/parse with the
// given arguments.
func ParseCommand(c Config, resultsDir string) *exec.Cmd {
	args := make([]string, 0, 10)
	// Levels of subdirectories to include in the job name.
	args = append(args, "-l", skylabResultsDirNestingLevel)
	// TODO(crbug.com/1012839): consider removing.
	args = append(args, "--record-duration")
	// Reparse results.
	args = append(args, "-r")
	// Parse a single results directory.
	args = append(args, "-o")
	// TODO(crbug.com/1012839): consider removing.
	args = append(args, "--suite-report")
	args = append(args, "--write-pidfile")
	args = append(args, resultsDir)
	return command(c, tkoRelpath, args...)
}

const dutPreparationRelPath = "site_utils/deployment/prepare/main.py"

// DutPreparationArgs contains arguments for DutPreparationCommand.
type DutPreparationArgs struct {
	Hostname     string
	ResultsDir   string
	HostInfoFile string
	Actions      []string
}

// DutPreparationCommand returns the Cmd struct to execute DUT preparation
// script with the given arguments.
func DutPreparationCommand(c Config, cmd *DutPreparationArgs) *exec.Cmd {
	args := make([]string, 0, 10)
	if cmd.Hostname != "" {
		args = append(args, "--hostname", cmd.Hostname)
	}
	if cmd.ResultsDir != "" {
		args = append(args, "--results-dir", cmd.ResultsDir)
	}
	if cmd.HostInfoFile != "" {
		args = append(args, "--host-info-file", cmd.HostInfoFile)
	}
	args = append(args, cmd.Actions...)
	return command(c, dutPreparationRelPath, args...)
}

// Config describes where the Autotest directory is.
type Config struct {
	AutotestDir string
	Environment map[string]string
}

// command creates an exec.Cmd for running an executable file in the
// Autotest directory.
func command(c Config, relpath string, args ...string) *exec.Cmd {
	path := filepath.Join(c.AutotestDir, relpath)
	log.Printf("Running Autotest command %s %s", path, args)
	cmd := exec.Command(path, args...)
	envVars := os.Environ()
	for k, v := range c.Environment {
		envVars = append(envVars, k+"="+v)
	}
	cmd.Env = envVars
	return cmd
}

// WriteKeyvals writes a map of keyvals in the format Autotest expects.
func WriteKeyvals(w io.Writer, m map[string]string) error {
	for k, v := range m {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}
