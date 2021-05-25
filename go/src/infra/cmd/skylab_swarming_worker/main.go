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
	luciErrors "go.chromium.org/luci/common/errors"
	lflag "go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/logging/gologger"

	"infra/cmd/skylab_swarming_worker/internal/annotations"
	"infra/cmd/skylab_swarming_worker/internal/fifo"
	"infra/cmd/skylab_swarming_worker/internal/lucifer"
	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness"
	"infra/cros/dutstate"
)

// Task names.
const (
	repairTaskName                    = "repair"
	deployTaskName                    = "deploy"
	auditTaskName                     = "audit"
	setStateNeedsRepairTaskName       = "set_needs_repair"
	setStateReservedTaskName          = "set_reserved"
	setStateManualRepairTaskName      = "set_manual_repair"
	setStateNeedsReplacementTaskName  = "set_needs_replacement"
	setStateNeedsManualRepairTaskName = "set_needs_manual_repair"
)

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
	defer i.Close(ctx)

	var luciferErr error

	switch {
	case a.taskName == setStateNeedsRepairTaskName:
		setStateForDUTs(i, dutstate.NeedsRepair)
	case a.taskName == setStateReservedTaskName:
		setStateForDUTs(i, dutstate.Reserved)
	case a.taskName == setStateManualRepairTaskName:
		setStateForDUTs(i, dutstate.ManualRepair)
	case a.taskName == setStateNeedsReplacementTaskName:
		setStateForDUTs(i, dutstate.NeedsReplacement)
	case a.taskName == setStateNeedsManualRepairTaskName:
		setStateForDUTs(i, dutstate.NeedsManualRepair)
	case isSupportedLuciferTask(a):
		luciferErr = luciferFlow(ctx, a, i, annotWriter)
	default:
		luciferErr = errors.New("skylab_swarming_worker failed to recognize task type")
	}

	if err := i.Close(ctx); err != nil {
		return err
	}
	return luciferErr
}

func setStateForDUTs(i *harness.Info, state dutstate.State) {
	for _, dh := range i.DUTs {
		dh.LocalState.HostState = state
	}
}

func luciferFlow(ctx context.Context, a *args, i *harness.Info, annotWriter writeCloser) error {
	var fifoPath string
	if a.logdogAnnotationURL != "" {
		// Set up FIFO, pipe, and goroutines like so:
		//
		//        worker -> LogDog pipe
		//                      ^
		// lucifer -> FIFO -go-/
		//
		// Both the worker and Lucifer need to write to LogDog.
		fifoPath = filepath.Join(i.TaskResultsDir.Path, "logdog.fifo")
		fc, err := fifo.NewCopier(annotWriter, fifoPath)
		if err != nil {
			return err
		}
		defer fc.Close()
	}
	var errs []error
	for _, dh := range i.DUTs {
		ta := lucifer.TaskArgs{
			AbortSock:  filepath.Join(dh.ResultsDir, "abort_sock"),
			GCPProject: gcpProject,
			ResultsDir: dh.ResultsDir,
			LogDogFile: fifoPath,
		}
		luciferErr := runLuciferTask(ctx, dh, a, ta)
		if luciferErr != nil {
			// Attempt to parse results regardless of lucifer errors.
			luciferErr = errors.Wrap(luciferErr, "run lucifer task")
			log.Printf("Encountered error on %s. Error: %s", dh.DUTHostname, luciferErr)
			errs = append(errs, luciferErr)
		}
	}
	annotations.BuildStep(annotWriter, "Epilog")
	annotations.StepLink(annotWriter, "Task results (Stainless)", i.Info.Task.StainlessURL())
	annotations.StepClosed(annotWriter)
	if len(errs) > 0 {
		return luciErrors.Annotate(luciErrors.MultiError(errs), "lucifer flow").Err()
	}
	return nil
}

func harnessOptions(a *args) []harness.Option {
	var ho []harness.Option
	if updatesInventory(a) {
		ho = append(ho, harness.UpdateInventory(getTaskName(a)))
	}
	return ho
}

func isSupportedLuciferTask(a *args) bool {
	return isAdminTask(a) || isDeployTask(a) || isAuditTask(a)
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

func runLuciferTask(ctx context.Context, dh *harness.DUTHarness, a *args, ta lucifer.TaskArgs) error {
	if !a.deadline.IsZero() {
		var c context.CancelFunc
		ctx, c = context.WithDeadline(ctx, a.deadline)
		defer c()
	}
	switch {
	case isAuditTask(a):
		return runAuditTask(ctx, dh, a.actions, ta)
	case isAdminTask(a):
		n, _ := getAdminTask(a.taskName)
		return runAdminTask(ctx, dh, n, ta)
	case isDeployTask(a):
		return runDeployTask(ctx, dh, a.actions, ta)
	default:
		panic("Unsupported task type")
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

// runAdminTask runs an admin task.  name is the name of the task.
func runAdminTask(ctx context.Context, dh *harness.DUTHarness, name string, ta lucifer.TaskArgs) (err error) {
	r := lucifer.AdminTaskArgs{
		TaskArgs: ta,
		Host:     dh.DUTHostname,
		Task:     name,
	}

	cmd := lucifer.AdminTaskCommand(dh.BotInfo.LuciferConfig(), r)
	if _, err := runLuciferCommand(ctx, cmd, dh, r.AbortSock); err != nil {
		return errors.Wrap(err, "run admin task")
	}
	return nil
}

// runDeployTask runs a deploy task using lucifer.
//
// actions is a possibly empty comma separated list of deploy actions to run
func runDeployTask(ctx context.Context, dh *harness.DUTHarness, actions string, ta lucifer.TaskArgs) error {
	r := lucifer.DeployTaskArgs{
		TaskArgs: ta,
		Host:     dh.DUTHostname,
		Actions:  actions,
	}

	cmd := lucifer.DeployTaskCommand(dh.BotInfo.LuciferConfig(), r)
	if _, err := runLuciferCommand(ctx, cmd, dh, r.AbortSock); err != nil {
		return errors.Wrap(err, "run deploy task")
	}
	return nil
}

// runAuditTask runs an audit task using lucifer.
//
// actions is a possibly empty comma separated list of deploy actions to run
func runAuditTask(ctx context.Context, dh *harness.DUTHarness, actions string, ta lucifer.TaskArgs) error {
	r := lucifer.AuditTaskArgs{
		TaskArgs: ta,
		Host:     dh.DUTHostname,
		Actions:  actions,
	}

	cmd := lucifer.AuditTaskCommand(dh.BotInfo.LuciferConfig(), r)
	if _, err := runLuciferCommand(ctx, cmd, dh, r.AbortSock); err != nil {
		return errors.Wrap(err, "run audit task")
	}
	return nil
}
