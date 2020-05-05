// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/google/subcommands"

	"infra/cros/cmd/lucifer/internal/api"
	"infra/cros/cmd/lucifer/internal/autotest/atutil"
	"infra/cros/cmd/lucifer/internal/event"
	"infra/cros/cmd/lucifer/internal/flagx"
	"infra/cros/cmd/lucifer/internal/metrics"
)

type testCmd struct {
	commonOpts
	clientTest        bool
	controlFile       string
	executionTag      string
	hosts             []string
	jobName           string
	jobOwner          string
	keyvals           map[string]string
	level             string
	localOnlyHostInfo bool
	parentJobID       int
	parseOnly         bool
	prejobTask        atutil.AdminTaskType
	provisionFail     bool
	provisionLabels   []string
	rebootAfter       string
	requireSSP        bool
	runReset          bool
	taskName          string
	testArgs          string
	testSourceBuild   string
}

func (testCmd) Name() string {
	return "test"
}
func (testCmd) Synopsis() string {
	return "Run a test"
}
func (testCmd) Usage() string {
	return `test [FLAGS]

lucifer test implements all parts of running an Autotest job.
Updating the status of a running job is delegated to the calling
process.  Status update events are printed to stdout, and the calling
process should perform the necessary updates.
`
}

func (c *testCmd) SetFlags(f *flag.FlagSet) {
	c.commonOpts.Register(f)

	// Options starting with an x are specific to Autotest.
	f.Var(flagx.CommaList(&c.hosts), "hosts",
		"DUT hostnames, comma separated")
	f.StringVar(&c.taskName, "task-name", "",
		"Name of the lucifer task to trigger")

	f.BoolVar(&c.clientTest, "x-client-test", false,
		"This is a client test")
	f.StringVar(&c.controlFile, "x-control-file", "",
		"Path to control file")
	f.StringVar(&c.executionTag, "x-execution-tag", "",
		"Execution tag passed to autoserv")
	f.Var(flagx.JSONMap(&c.keyvals), "x-keyvals",
		"JSON string of job keyvals")
	f.StringVar(&c.level, "x-level", "STARTING",
		"lucifer rollout level")
	f.StringVar(&c.jobName, "x-job-name", "",
		"Job name")
	f.StringVar(&c.jobOwner, "x-job-owner", "",
		"Job owner")
	f.BoolVar(&c.localOnlyHostInfo, "x-local-only-host-info", false,
		"If set, do not reflect HostInfo updates back to AFE")
	f.IntVar(&c.parentJobID, "x-parent-job-id", 0,
		"Autotest parent job id")
	f.BoolVar(&c.parseOnly, "x-parse-only", false,
		"Do parsing only (STARTING level only)")
	f.Var(flagx.TaskType(&c.prejobTask, flagx.RejectRepair), "x-prejob-task",
		"Prejob task to run (Skylab only, ignored when provisioning).")
	f.BoolVar(&c.provisionFail, "x-provision-fail", false,
		"Act as if provisioning failed")
	f.Var(flagx.CommaList(&c.provisionLabels), "x-provision-labels",
		"Labels to provision, comma separated")
	f.StringVar(&c.rebootAfter, "x-reboot-after", "never",
		"frontend.afe.models.Job.reboot_after from Autotest")
	f.BoolVar(&c.requireSSP, "x-require-ssp", false,
		"Require SSP")
	f.BoolVar(&c.runReset, "x-run-reset", false,
		"Run reset")
	f.StringVar(&c.testArgs, "x-test-args", "",
		"Test args (meaning depends on test)")
	f.StringVar(&c.testSourceBuild, "x-test-source-build", "",
		"Autotest test source build")
}

func (c *testCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ctx, res, err := commonSetup(ctx, c.commonOpts)
	if err != nil {
		log.Print(err)
		return subcommands.ExitFailure
	}
	defer res.Close()
	c.prepareFlags()
	if err := runJob(ctx, c, res.apiClient()); err != nil {
		log.Printf("Error running test: %s", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func (c *testCmd) prepareFlags() {
	c.ensureJobName()
}

// EnsureJobName generate job names for Skylab.  This gets inserted as
// label into keyval and tko/parse depends on it to parse build info.
func (c *testCmd) ensureJobName() {
	if c.jobName != "" {
		return
	}
	// The purpose of the jobName is to be added to job keyvals by
	// autoserv.  This is passed as a separate flag even though
	// autoserv already accepts job keyvals.  The label flag value
	// will override the label keyval passed to autoserv.
	if label := c.keyvals["label"]; label != "" {
		c.jobName = label
		return
	}
	c.jobName = fmt.Sprintf("%s/%s/%s", c.keyvals["build"], c.keyvals["suite"], c.taskName)
}

func (c *testCmd) mainJob() *atutil.MainJob {
	return &atutil.MainJob{
		AutotestConfig:   c.autotestConfig(),
		ResultsDir:       c.resultsDir,
		UseLocalHostInfo: c.localOnlyHostInfo,
	}
}

func (c *testCmd) isHostless() bool {
	return len(c.hosts) == 0
}

func (c *testCmd) adminTask(host string) *atutil.AdminTask {
	return &atutil.AdminTask{
		Host:       host,
		ResultsDir: filepath.Join(c.resultsDir, fmt.Sprintf("prejob_%s", host)),
	}
}

func (c *testCmd) provisionJob(host string) *atutil.Provision {
	return &atutil.Provision{
		Host:              host,
		Labels:            c.provisionLabels,
		LocalOnlyHostInfo: c.localOnlyHostInfo,
		ResultsDir:        filepath.Join(c.resultsDir, fmt.Sprintf("provision_%s", host)),
	}
}

func (c *testCmd) hostlessTest() *atutil.HostlessTest {
	return &atutil.HostlessTest{
		Args:         c.testArgs,
		ClientTest:   c.clientTest,
		ControlFile:  c.controlFile,
		ControlName:  c.taskName,
		ExecutionTag: c.executionTag,
		Keyvals:      c.keyvals,
		Name:         c.jobName,
		Owner:        c.jobOwner,
		ResultsDir:   c.resultsDir,
	}
}

func (c *testCmd) hostTest() *atutil.HostTest {
	hlt := c.hostlessTest()
	t := &atutil.HostTest{
		HostlessTest:      *hlt,
		Hosts:             c.hosts,
		LocalOnlyHostInfo: c.localOnlyHostInfo,
		ParentJobID:       c.parentJobID,
		RequireSSP:        c.requireSSP,
		TestSourceBuild:   c.testSourceBuild,
	}
	// TODO(crbug.com/830912): Temporary for STARTING -> SKYLAB_PROVISION
	if c.level == "SKYLAB_PROVISION" {
		t.ResultsDir = filepath.Join(c.resultsDir, "autoserv_test")
	}
	return t
}

// tkoLevel returns the correct level argument to pass to tkoParse.
func (c *testCmd) tkoLevel() int {
	// For Autotest jobs, the results directory suffix of
	// relevance is of the form <job_id>-<user>/<host_id>.
	// tko/parse should include the last two directories in the
	// parsed jobname.
	//
	// For Skylab tasks, the results directory suffix of relevance is one
	// of the following forms:
	//   - swarming-<task_id>/<run_id>/autoserv_test
	//   - swarming-<task_id>/<run_id>/prejob_<host_id>
	//   - swarming-<task_id>/<run_id>/provision_<host_id>
	//
	// tko/parse should include the last three directories in the
	// parsed jobname.
	switch c.level {
	case "SKYLAB_PROVISION":
		return 3
	case "STARTING":
		return 2
	default:
		panic(fmt.Sprintf("Invalid lucifer level: %s", c.level))
	}
}

// runJob runs a job from beginning to end.
//
// runJob returns an error if there is an error.  This does not
// include failed tests, as tests results are uploaded using a
// separate channel.  Returned errors should be considered
// infrastructure issues (as opposed to product or test issues).
//
// A job may include tests, pre-test tasks, and post-test tasks.
// This contains all logic, excluding argument parsing and logging setup.
func runJob(ctx context.Context, c *testCmd, ac *api.Client) error {
	metrics.StartCounter.Add(ctx, 1)
	event.Send(event.Starting)
	defer event.Send(event.Completed)
	switch c.level {
	case "SKYLAB_PROVISION":
		return runSkylabProvisionJob(ctx, c, ac)
	case "STARTING":
		if c.parseOnly {
			return runParsingJob(ctx, c, ac)
		}
		return runStartingJob(ctx, c, ac)
	default:
		return fmt.Errorf("Invalid lucifer level: %s", c.level)
	}
}

// runSkylabProvisionJob is the logic path for lucifer running at
// SKYLAB_PROVISION.  This function returns the first infrastructure
// error encountered that prevented the job from running successfully.
// No error is returned if all tests are run (whether or not they
// pass) and test results are uploaded.
func runSkylabProvisionJob(ctx context.Context, c *testCmd, ac *api.Client) error {
	if len(c.provisionLabels) > 0 {
		if err := doProvisioningStep(ctx, c, ac); err != nil {
			return err
		}
	} else if c.prejobTask != atutil.NoTask {
		if err := doPrejobStep(ctx, c, ac); err != nil {
			return err
		}
	}
	return runStartingJob(ctx, c, ac)
}

// runStartingJob is the logic path for lucifer running at STARTING.
// This function returns the first infrastructure error encountered
// that prevented the job from running successfully.  No error is
// returned if all tests are run (whether or not they pass) and test
// results are uploaded.
func runStartingJob(ctx context.Context, c *testCmd, ac *api.Client) error {
	var err error
	if ctx.Err() == nil {
		err = doRunningStep(ctx, c, ac)
	}
	if err2 := doParsingStep(ctx, c, ac); err == nil {
		err = err2
	}
	if ctx.Err() != nil {
		event.Send(event.Aborted)
		if err == nil {
			err = fmt.Errorf("aborted")
		}
	}
	return err
}

// runParsingJob is the logic path for lucifer running at STARTING
// with -x-parse-only.  This function returns the last error
// encountered that prevented the job from running successfully.  The
// job is successful if all tests are run (whether or not they pass)
// and test results are uploaded.
func runParsingJob(ctx context.Context, c *testCmd, ac *api.Client) (last error) {
	if err := doParsingStep(ctx, c, ac); err != nil {
		last = err
	}
	if ctx.Err() != nil {
		event.Send(event.Aborted)
		last = fmt.Errorf("aborted")
	}
	return
}
