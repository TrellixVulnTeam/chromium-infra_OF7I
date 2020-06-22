// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Program skylab_swarming_worker executes a Skylab task via Lucifer.
//
// skylab_swarming_worker uses lucifer_run_job to actually run the autotest
// job. Once lucifer_run_job is kicked off, skylab_swarming_worker handles Lucifer
// events, translating them to task updates and runtime status updates of the
// swarming bot. If the swarming task is canceled, lucifer_swarming_worker aborts
// the Lucifer run.
//
// The following environment variables control skylab_swarming_worker
// execution.
//
// Per-bot variables:
//
//   ADMIN_SERVICE: Admin service host, e.g. foo.appspot.com.
//   AUTOTEST_DIR: Path to the autotest checkout on server.
//   LUCIFER_TOOLS_DIR: Path to the lucifer installation.
//   PARSER_PATH: Path to the autotest_status_parser installation.
//   SKYLAB_DUT_ID: skylab_inventory id of the DUT that belongs to this bot.
//
// Per-task variables:
//
//   SWARMING_TASK_ID: task id of the swarming task being serviced.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	protoCommon "go.chromium.org/chromiumos/infra/proto/go/test_platform/common"
	lflag "go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/logging/gologger"

	"infra/cmd/skylab_swarming_worker/internal/annotations"
	"infra/cmd/skylab_swarming_worker/internal/autotest/constants"
	"infra/cmd/skylab_swarming_worker/internal/fifo"
	"infra/cmd/skylab_swarming_worker/internal/lucifer"
	"infra/cmd/skylab_swarming_worker/internal/parser"
	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness"
)

const repairTaskName = "repair"
const deployTaskName = "deploy"
const auditTaskName = "audit"
const setStateNeedsRepairTaskName = "set_needs_repair"

const gcpProject = "chromeos-skylab"

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(os.Args[0])))
	log.Printf("Starting with args: %s", os.Args)
	a := parseArgs()
	if err := mainInner(a); err != nil {
		log.Fatalf("Error: %s", err)
	}
	log.Printf("Exited successfully")
}

type args struct {
	adminService        string
	deadline            time.Time
	actions             string
	forceFreshInventory bool
	isolatedOutdir      string
	logdogAnnotationURL string
	sideEffectsConfig   string
	taskName            string
	xClientTest         bool
	xKeyvals            map[string]string
	xProvisionLabels    []string
	xTestArgs           string
}

func parseArgs() *args {
	a := &args{}

	flag.StringVar(&a.taskName, "task-name", "",
		"Name of the task to run. For autotest, this is the NAME attribute in control file")
	flag.StringVar(&a.logdogAnnotationURL, "logdog-annotation-url", "",
		"LogDog annotation URL, like logdog://HOST/PROJECT/PREFIX/+/annotations")
	flag.StringVar(&a.adminService, "admin-service", "",
		"Admin service host, e.g. foo.appspot.com")
	flag.BoolVar(&a.forceFreshInventory, "force-fresh", false,
		"Use fresh inventory information. This flag can increase task runtime.")
	flag.BoolVar(&a.xClientTest, "client-test", false,
		"This is a client side test")
	flag.Var(lflag.CommaList(&a.xProvisionLabels), "provision-labels",
		"Labels to provision, comma separated")
	flag.Var(lflag.JSONMap(&a.xKeyvals), "keyvals",
		"JSON string of job keyvals")
	flag.StringVar(&a.xTestArgs, "test-args", "",
		"Test args (meaning depends on test)")
	flag.StringVar(&a.actions, "actions", "",
		"Actions to execute for a task")
	flag.StringVar(&a.isolatedOutdir, "isolated-outdir", "",
		"Directory to place isolated output into. Generate no isolated output if not set.")
	flag.StringVar(&a.sideEffectsConfig, "side-effect-config", "",
		"JSONpb string of side_effects.Config to be dropped into the results directory. No file is created if empty.")
	flag.Var(lflag.Time(&a.deadline), "deadline",
		"Soft deadline for completion, formatted as stiptime. Wrap-up actions may outlive this deadline.")
	flag.Parse()

	return a
}

func mainInner(a *args) error {
	ctx := context.Background()
	// Set up Go logger for LUCI libraries.
	ctx = gologger.StdConfig.Use(ctx)
	b := swmbot.GetInfo()
	log.Printf("Swarming bot config: %#v", b)
	annotWriter, err := openLogDogWriter(ctx, a.logdogAnnotationURL)
	if err != nil {
		return err
	}
	defer annotWriter.Close()
	i, err := harness.Open(ctx, b, harnessOptions(a)...)
	log.Printf("mainInner: harness info object (%#v)", i)
	if err != nil {
		return err
	}
	defer i.Close()

	var luciferErr error

	switch a.taskName {
	case setStateNeedsRepairTaskName:
		i.BotInfo.HostState = swmbot.HostNeedsRepair
	default:
		luciferErr = luciferFlow(ctx, a, i, annotWriter)
	}

	if err := i.Close(); err != nil {
		return err
	}

	return luciferErr
}

func luciferFlow(ctx context.Context, a *args, i *harness.Info, annotWriter writeCloser) error {
	ta := lucifer.TaskArgs{
		AbortSock:  filepath.Join(i.ResultsDir, "abort_sock"),
		GCPProject: gcpProject,
		ResultsDir: i.ResultsDir,
	}
	if a.logdogAnnotationURL != "" {
		// Set up FIFO, pipe, and goroutines like so:
		//
		//        worker -> LogDog pipe
		//                      ^
		// lucifer -> FIFO -go-/
		//
		// Both the worker and Lucifer need to write to LogDog.
		fifoPath := filepath.Join(i.ResultsDir, "logdog.fifo")
		fc, err := fifo.NewCopier(annotWriter, fifoPath)
		if err != nil {
			return err
		}
		defer fc.Close()
		ta.LogDogFile = fifoPath
	}

	gsBucket := "chromeos-autotest-results"

	if a.sideEffectsConfig != "" {
		sec, err := parseSideEffectsConfig(a.sideEffectsConfig, annotWriter)
		if err != nil {
			return err
		}
		if err = dropSideEffectsConfig(sec, i.ResultsDir, annotWriter); err != nil {
			return err
		}
		gsBucket = sec.GetGoogleStorage().GetBucket()
	}

	luciferErr := runLuciferTask(ctx, i, a, ta)

	if luciferErr != nil {
		// Attempt to parse results regardless of lucifer errors.
		luciferErr = errors.Wrap(luciferErr, "run lucifer task")
		log.Printf("Encountered error, continuing with result parsing anyway."+
			"Error: %s", luciferErr)
	}

	switch {
	case isAdminTask(a) || isDeployTask(a) || isAuditTask(a):
		// Show Stainless links here for admin tasks.
		// For test tasks they are bundled with results.json.
		annotations.BuildStep(annotWriter, "Epilog")
		annotations.StepLink(annotWriter, "Task results (Stainless)", i.Info.Task.StainlessURL())
		annotations.StepClosed(annotWriter)
	default:
		pa := i.ParserArgs()
		pa.Failed = luciferErr != nil

		r, err := parser.GetResults(pa, annotWriter)
		if err != nil {
			return errors.Wrap(err, "results parsing")
		}

		r.LogData = &protoCommon.TaskLogData{
			GsUrl: i.Task.GsURL(gsBucket),
		}

		err = writeResultsFile(a.isolatedOutdir, r, annotWriter)

		if err != nil {
			return errors.Wrap(err, "writing results to isolated output file")
		}
	}
	return luciferErr
}

func harnessOptions(a *args) []harness.Option {
	var ho []harness.Option
	if updatesInventory(a) {
		ho = append(ho, harness.UpdateInventory(getTaskName(a)))
	}
	if a.forceFreshInventory {
		ho = append(ho, harness.WaitForFreshInventory)
	}
	return ho
}

// updatesInventory returns true if the task(repair/deploy/audit)
// should update the inventory else false.
func updatesInventory(a *args) bool {
	if isRepairTask(a) || isDeployTask(a) || isAuditTask(a) {
		return true
	}
	return false
}

// getTaskName returns the task name(repair/deploy/audit) for the task.
func getTaskName(a *args) string {
	switch {
	case isRepairTask(a):
		return repairTaskName
	case isDeployTask(a):
		return deployTaskName
	case isAuditTask(a):
		return auditTaskName
	default:
		return ""
	}
}

func runLuciferTask(ctx context.Context, i *harness.Info, a *args, ta lucifer.TaskArgs) error {
	if !a.deadline.IsZero() {
		var c context.CancelFunc
		ctx, c = context.WithDeadline(ctx, a.deadline)
		defer c()
	}
	switch {
	case isAuditTask(a):
		return runAuditTask(ctx, i, a.actions, ta)
	case isAdminTask(a):
		n, _ := getAdminTask(a.taskName)
		return runAdminTask(ctx, i, n, ta)
	case isDeployTask(a):
		return runDeployTask(ctx, i, a.actions, ta)
	default:
		return runTest(ctx, i, a, ta)
	}
}

// getAdminTask returns the admin task name if the given task is an
// admin task.  If the given task is not an admin task, ok will be
// false.
func getAdminTask(name string) (task string, ok bool) {
	if strings.HasPrefix(name, "admin_") {
		return strings.TrimPrefix(name, "admin_"), true
	}
	return "", false
}

// isAdminTask determines whether the args specify an admin task
func isAdminTask(a *args) bool {
	_, isAdmin := getAdminTask(a.taskName)
	return isAdmin
}

// isDeployTask determines if the given task name corresponds to a deploy task.
func isDeployTask(a *args) bool {
	return a.taskName == deployTaskName
}

// isAuditTask determines if the given task name corresponds to a audit task.
func isAuditTask(a *args) bool {
	task, _ := getAdminTask(a.taskName)
	return task == auditTaskName
}

// isRepairTask determines if the given task name corresponds to a repair task.
func isRepairTask(a *args) bool {
	task, _ := getAdminTask(a.taskName)
	return task == repairTaskName
}

// runTest runs a test.
func runTest(ctx context.Context, i *harness.Info, a *args, ta lucifer.TaskArgs) (err error) {
	// TODO(ayatane): Always reboot between each test for now.
	tc := prejobTaskControl{
		runReset:     true,
		rebootBefore: RebootAlways,
	}
	r := lucifer.TestArgs{
		TaskArgs:           ta,
		Hosts:              []string{i.DUTName},
		TaskName:           a.taskName,
		XTestArgs:          a.xTestArgs,
		XClientTest:        a.xClientTest,
		XKeyvals:           a.xKeyvals,
		XLevel:             lucifer.LuciferLevelSkylabProvision,
		XLocalOnlyHostInfo: true,
		// TODO(ayatane): hostDirty, hostProtected not implemented
		XPrejobTask:      choosePrejobTask(tc, true, false),
		XProvisionLabels: a.xProvisionLabels,
	}

	cmd := lucifer.TestCommand(i.LuciferConfig(), r)
	lr, err := runLuciferCommand(ctx, cmd, i, r.AbortSock)
	switch {
	case err != nil:
		return errors.Wrap(err, "run lucifer failed")
	case lr.TestsFailed > 0:
		return errors.Errorf("%d tests failed", lr.TestsFailed)
	default:
		return nil
	}
}

type rebootBefore int

// Reboot type values.
const (
	RebootNever rebootBefore = iota
	RebootIfDirty
	RebootAlways
)

// prejobTaskControl groups values used to control whether to run
// prejob tasks for tests.  Note that there are subtle interactions
// between these values, e.g., runReset may run verify as part of
// reset even if runVerify is false, but runReset will fail if
// rebootBefore is RebootNever because that restricts cleanup, which
// runs as part of reset.
type prejobTaskControl struct {
	runVerify    bool
	runReset     bool
	rebootBefore rebootBefore
}

func choosePrejobTask(tc prejobTaskControl, hostDirty, hostProtected bool) constants.AdminTaskType {
	willVerify := (tc.runReset || tc.runVerify) && !hostProtected

	var willReboot bool
	switch tc.rebootBefore {
	case RebootAlways:
		willReboot = true
	case RebootIfDirty:
		willReboot = hostDirty || (tc.runReset && willVerify)
	case RebootNever:
		willReboot = false
	}

	switch {
	case willReboot && willVerify:
		return constants.Reset
	case willReboot:
		return constants.Cleanup
	case willVerify:
		return constants.Verify
	default:
		return constants.NoTask
	}
}

// runAdminTask runs an admin task.  name is the name of the task.
func runAdminTask(ctx context.Context, i *harness.Info, name string, ta lucifer.TaskArgs) (err error) {
	r := lucifer.AdminTaskArgs{
		TaskArgs: ta,
		Host:     i.DUTName,
		Task:     name,
	}

	cmd := lucifer.AdminTaskCommand(i.LuciferConfig(), r)
	if _, err := runLuciferCommand(ctx, cmd, i, r.AbortSock); err != nil {
		return errors.Wrap(err, "run admin task")
	}
	return nil
}

// runDeployTask runs a deploy task using lucifer.
//
// actions is a possibly empty comma separated list of deploy actions to run
func runDeployTask(ctx context.Context, i *harness.Info, actions string, ta lucifer.TaskArgs) error {
	r := lucifer.DeployTaskArgs{
		TaskArgs: ta,
		Host:     i.DUTName,
		Actions:  actions,
	}

	cmd := lucifer.DeployTaskCommand(i.LuciferConfig(), r)
	if _, err := runLuciferCommand(ctx, cmd, i, r.AbortSock); err != nil {
		return errors.Wrap(err, "run deploy task")
	}
	return nil
}

// runAuditTask runs an audit task using lucifer.
//
// actions is a possibly empty comma separated list of deploy actions to run
func runAuditTask(ctx context.Context, i *harness.Info, actions string, ta lucifer.TaskArgs) error {
	r := lucifer.AuditTaskArgs{
		TaskArgs: ta,
		Host:     i.DUTName,
		Actions:  actions,
	}

	cmd := lucifer.AuditTaskCommand(i.LuciferConfig(), r)
	if _, err := runLuciferCommand(ctx, cmd, i, r.AbortSock); err != nil {
		return errors.Wrap(err, "run audit task")
	}
	return nil
}
