// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"context"
	"flag"
	"fmt"
	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/flagx"
	crosfleetpb "infra/cmd/crosfleet/internal/proto"
	"infra/cmdsupport/cmdlib"
	"math"
	"strings"
	"sync"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/auth/client/authcli"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// DefaultSwarmingPriority is the default priority for a Swarming task.
	DefaultSwarmingPriority = int64(140)
	// MinSwarmingPriority is the lowest-allowed priority for a Swarming task.
	MinSwarmingPriority = int64(50)
	// MaxSwarmingPriority is the highest-allowed priority for a Swarming task.
	MaxSwarmingPriority = int64(255)
	// imageArchiveBaseURL is the base url for the ChromeOS image archive.
	imageArchiveBaseURL = "gs://chromeos-image-archive/"
	// ctpExecuteStepName is the name of the test-execution step in any
	// cros_test_platform Buildbucket build. This step is not started until
	// all request-validation and setup steps are passed.
	ctpExecuteStepName = "execute"
	// How long build tags are allowed to be before we trigger a Swarming API
	// error due to how they store tags in a datastore. Most tags shouldn't be
	// anywhere close to this limit, but tags that could potentially be very
	// long we should crop them to this limit.
	maxSwarmingTagLength = 300
	// Maximum number of CTP builds that can be run from one "crosfleet run ..."
	// command.
	maxCTPRunsPerCmd = 12
)

// testCommonFlags contains parameters common to the "run
// test", "run suite", and "run testplan" subcommands.
type testCommonFlags struct {
	board           string
	models          []string
	pool            string
	image           string
	release         string
	qsAccount       string
	maxRetries      int
	repeats         int
	priority        int64
	timeoutMins     int
	addedDims       map[string]string
	provisionLabels map[string]string
	addedTags       map[string]string
	keyvals         map[string]string
	exitEarly       bool
	lacrosPath      string
}

// Registers run command-specific flags
func (c *testCommonFlags) register(f *flag.FlagSet) {
	f.StringVar(&c.image, "image", "", `Optional fully specified image name to run test against, e.g. octopus-release/R89-13609.0.0.
If no value for image or release is passed, test will run against the latest green postsubmit build for the given board.`)
	f.StringVar(&c.release, "release", "", `Optional ChromeOS release branch to run test against, e.g. R89-13609.0.0.
If no value for image or release is passed, test will run against the latest green postsubmit build for the given board.`)
	f.StringVar(&c.board, "board", "", "Board to run tests on.")
	f.Var(luciflag.StringSlice(&c.models), "model", fmt.Sprintf(`Model to run tests on; may be specified multiple times.
A maximum of %d tests may be launched per "crosfleet run" command.`, maxCTPRunsPerCmd))
	f.Var(luciflag.CommaList(&c.models), "models", "Comma-separated list of models to run tests on in same format as -model.")
	f.IntVar(&c.repeats, "repeats", 1, fmt.Sprintf(`Number of repeat tests to launch (per model specified).
A maximum of %d tests may be launched per "crosfleet run" command.`, maxCTPRunsPerCmd))
	f.StringVar(&c.pool, "pool", "", "Device pool to run tests on.")
	f.StringVar(&c.qsAccount, "qs-account", "", `Optional Quota Scheduler account to use for this task. Overrides -priority flag.
If no account is set, tests are scheduled using -priority flag.`)
	f.IntVar(&c.maxRetries, "max-retries", 0, "Maximum retries allowed. No retry if set to 0.")
	f.Int64Var(&c.priority, "priority", DefaultSwarmingPriority, `Swarming scheduling priority for tests, between 50 and 255 (lower values indicate higher priorities).
If a Quota Scheduler account is specified via -qs-account, this value is not used.`)
	f.IntVar(&c.timeoutMins, "timeout-mins", 360, "Test run timeout.")
	f.Var(flagx.KeyVals(&c.addedDims), "dim", "Additional scheduling dimension in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.addedDims), "dims", "Comma-separated additional scheduling addedDims in same format as -dim.")
	f.Var(flagx.KeyVals(&c.provisionLabels), "provision-label", "Additional provisionable label in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.provisionLabels), "provision-labels", "Comma-separated additional provisionable labels in same format as -provision-label.")
	f.Var(flagx.KeyVals(&c.addedTags), "tag", "Additional Swarming metadata tag in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.addedTags), "tags", "Comma-separated Swarming metadata tags in same format as -tag.")
	f.Var(flagx.KeyVals(&c.keyvals), "autotest-keyval", "Autotest keyval in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.keyvals), "autotest-keyvals", "Comma-separated Autotest keyvals in same format as -keyval.")
	f.BoolVar(&c.exitEarly, "exit-early", false, "Exit command as soon as test is scheduled. crosfleet will not notify on test validation failure.")
	f.StringVar(&c.lacrosPath, "lacros-path", "", "Optional GCS path pointing to a lacros artifact.")
}

// validateAndAutocompleteFlags returns any errors after validating the CLI
// flags, and autocompletes the -image flag unless it was specified by the user.
func (c *testCommonFlags) validateAndAutocompleteFlags(ctx context.Context, f *flag.FlagSet, mainArgType, bbService string, authFlags authcli.Flags, printer common.CLIPrinter) error {
	if err := c.validateArgs(f, mainArgType); err != nil {
		return err
	}
	if c.release != "" {
		// Users can specify the ChromeOS release branch via the -release flag,
		// rather than specifying a full image name. In this case, we infer the
		// full image name from the release branch.
		c.image = releaseImage(c.board, c.release)
	} else if c.image == "" {
		// If no release or image was specified, determine the latest green
		// postsubmit
		// image for the given board.
		latestImage, err := latestImage(ctx, c.board, bbService, authFlags)
		if err != nil {
			return fmt.Errorf("error determining the latest image for board %s: %v", c.board, err)
		}
		printer.WriteTextStderr("Using latest green build image %s for board %s", latestImage, c.board)
		c.image = latestImage
	}
	return nil
}

func (c *testCommonFlags) validateArgs(f *flag.FlagSet, mainArgType string) error {
	var errors []string
	if c.board == "" {
		errors = append(errors, "missing board flag")
	}
	if c.pool == "" {
		errors = append(errors, "missing pool flag")
	}
	if c.image != "" && c.release != "" {
		errors = append(errors, "cannot specify both image and release branch")
	}
	if c.priority < MinSwarmingPriority || c.priority > MaxSwarmingPriority {
		errors = append(errors, fmt.Sprintf("priority flag should be in [%d, %d]", MinSwarmingPriority, MaxSwarmingPriority))
	}
	// If no models are specified, we still schedule one test with model label
	// left blank.
	numUniqueDUTs := int(math.Max(1, float64(len(c.models))))
	if numUniqueDUTs*c.repeats > maxCTPRunsPerCmd {
		errors = append(errors, fmt.Sprintf("total number of CTP runs launched (# models specified * repeats) cannot exceed %d", maxCTPRunsPerCmd))
	}
	if f.NArg() == 0 {
		errors = append(errors, fmt.Sprintf("missing %v arg", mainArgType))
	}

	if len(errors) > 0 {
		return cmdlib.NewUsageError(*f, strings.Join(errors, "\n"))
	}
	return nil
}

// releaseImage constructs a build image name from the release builder for the
// given board and ChromeOS release branch.
func releaseImage(board, release string) string {
	return fmt.Sprintf("%s-release/%s", board, release)
}

// latestImage gets the build image from the latest green postsubmit build for
// the given board.
func latestImage(ctx context.Context, board, bbService string, authFlags authcli.Flags) (string, error) {
	postsubmitBuilder := &buildbucketpb.BuilderID{
		Project: "chromeos",
		Bucket:  "postsubmit",
		Builder: fmt.Sprintf("%s-postsubmit", board),
	}
	postsubmitBBClient, err := buildbucket.NewClient(ctx, postsubmitBuilder, bbService, authFlags)
	if err != nil {
		return "", err
	}
	latestGreenPostsubmit, err := postsubmitBBClient.GetLatestGreenBuild(ctx)
	if err != nil {
		return "", err
	}
	outputProperties := latestGreenPostsubmit.Output.Properties.GetFields()
	artifacts := outputProperties["artifacts"].GetStructValue().GetFields()
	image := artifacts["gs_path"].GetStringValue()
	if image == "" {
		buildURL := postsubmitBBClient.BuildURL(latestGreenPostsubmit.Id)
		return "", fmt.Errorf("most recent postsubmit for board %s has no corresponding build image; visit postsubmit build at %s for more details", board, buildURL)
	}
	return image, nil
}

// buildTagsForModel combines test metadata tags with user-added tags for the
// given model.
func (c *testCommonFlags) buildTagsForModel(crosfleetTool string, model string, mainArg string) map[string]string {
	tags := map[string]string{}

	// Add user-added tags.
	for key, val := range c.addedTags {
		tags[key] = val
	}

	// Add crosfleet-tool tag.
	if crosfleetTool == "" {
		panic(fmt.Errorf("must provide %s tag", common.CrosfleetToolTag))
	}
	tags[common.CrosfleetToolTag] = crosfleetTool
	if mainArg != "" {
		// Intended for `run test` and `run suite` commands. This label takes
		// the form "label-suite:SUITE_NAME" for a `run suite` command.
		tags[fmt.Sprintf("label-%s", crosfleetTool)] = mainArg
	}

	// Add metadata tags.
	if c.board != "" {
		tags["label-board"] = c.board
	}
	if model != "" {
		tags["label-model"] = model
	}
	if c.pool != "" {
		tags["label-pool"] = c.pool
	}
	if c.image != "" {
		tags["label-image"] = c.image
	}
	// Only surface the priority if Quota Account was unset.
	// NOTE: these addedTags themselves will NOT be processed by Buildbucket or
	// Swarming--they are for metadata purposes only.
	// addedTags attached here will NOT be processed by CTP.
	if c.qsAccount != "" {
		tags["label-quota-account"] = c.qsAccount
	} else if c.priority != 0 {
		tags["label-priority"] = fmt.Sprint(c.priority)
	}

	return tags
}

// testRunLauncher contains the necessary information to launch and validate a
// CTP test plan.
type ctpRunLauncher struct {
	// Tag denoting the tests or suites specified to run; left blank for custom
	// test plans.
	mainArgsTag string
	printer     common.CLIPrinter
	cmdName     string
	bbClient    *buildbucket.Client
	testPlan    *test_platform.Request_TestPlan
	cliFlags    *testCommonFlags
	exitEarly   bool
}

// launchAndOutputTests invokes the inner launchTestsAsync() function
// and handles the CLI output of the buildLaunchList JSON object, which should
// happen even in case of command failure.
func (l *ctpRunLauncher) launchAndOutputTests(ctx context.Context) error {
	buildLaunchList, err := l.launchTestsAsync(ctx)
	l.printer.WriteJSONStdout(buildLaunchList)
	return err
}

// launchTestsAsync requests a run of the given CTP run launcher's
// test plan, and returns the ID of the launched cros_test_platform Buildbucket
// build. Unless the exitEarly arg is passed as true, the function waits to
// return until the build passes request-validation and setup steps.
func (l *ctpRunLauncher) launchTestsAsync(ctx context.Context) (*crosfleetpb.BuildLaunchList, error) {
	buildLaunchList, scheduledAnyBuilds, schedulingErrors := l.scheduleCTPBuildsAsync(ctx)
	if len(schedulingErrors) > 0 {
		fullErrorMsg := fmt.Sprintf("Encountered the following errors requesting %s run(s):\n%s\n",
			l.cmdName, strings.Join(schedulingErrors, "\n"))
		if scheduledAnyBuilds {
			// Don't fail the command if we were able to request some builds.
			l.printer.WriteTextStderr(fullErrorMsg)
		} else {
			return buildLaunchList, fmt.Errorf(fullErrorMsg)
		}
	}
	if l.exitEarly {
		return buildLaunchList, nil
	}
	l.printer.WriteTextStderr(`Waiting to confirm %s run request validation...
(To skip this step, pass the -exit-early flag on future %s run commands)
`, l.cmdName, l.cmdName)
	confirmedAnyBuilds, confirmationErrors := l.confirmCTPBuildsAsync(ctx, buildLaunchList)
	if len(confirmationErrors) > 0 {
		fullErrorMsg := fmt.Sprintf("Encountered the following errors confirming %s run(s):\n%s\n",
			l.cmdName, strings.Join(confirmationErrors, "\n"))
		if confirmedAnyBuilds {
			// Don't fail the command if we were able to confirm some of the
			// requested builds as having started.
			l.printer.WriteTextStderr(fullErrorMsg)
		} else {
			return buildLaunchList, fmt.Errorf(fullErrorMsg)
		}
	}
	return buildLaunchList, nil
}

// scheduleCTPBuild uses the given Buildbucket client to request a
// cros_test_platform Buildbucket build for the CTP run launcher's test plan,
// build tags, and command line flags, and returns the ID of the pending build.
func (l *ctpRunLauncher) scheduleCTPBuild(ctx context.Context, model string) (*buildbucketpb.Build, error) {
	buildTags := l.cliFlags.buildTagsForModel(l.cmdName, model, l.mainArgsTag)
	ctpRequest, err := l.testPlatformRequest(model, buildTags)
	if err != nil {
		return nil, err
	}
	buildProps := map[string]interface{}{
		"requests": map[string]interface{}{
			// Convert to protoreflect.ProtoMessage for easier type comparison.
			"default": ctpRequest.ProtoReflect().Interface(),
		},
	}
	// Parent cros_test_platform builds run on generic GCE bots at the default
	// priority, so we pass zero values for the dimensions and priority of the
	// parent build.
	//
	// buildProps contains separate dimensions and priority values to apply to
	// the child test_runner builds that will be launched by the parent build.
	return l.bbClient.ScheduleBuild(ctx, buildProps, nil, buildTags, 0)
}

// scheduleCTPBuildsAsync schedules all builds asynchronously and returns a
// build launch list, a bool indicating whether any builds were successfully
// scheduled, and a slice of scheduling error strings. Mutex locks are used to
// avoid race conditions from concurrent writes to the return variables in the
// async loop.
func (l *ctpRunLauncher) scheduleCTPBuildsAsync(ctx context.Context) (buildLaunchList *crosfleetpb.BuildLaunchList, scheduledAnyBuilds bool, schedulingErrors []string) {
	buildLaunchList = &crosfleetpb.BuildLaunchList{}
	waitGroup := sync.WaitGroup{}
	mutex := sync.Mutex{}
	allModels := l.cliFlags.models
	if len(allModels) == 0 {
		// If no models are specified, just launch one run with a blank model.
		allModels = []string{""}
	}
	for _, model := range allModels {
		for i := 0; i < l.cliFlags.repeats; i++ {
			waitGroup.Add(1)
			model := model
			go func() {
				build, err := l.scheduleCTPBuild(ctx, model)
				mutex.Lock()
				errString := ""
				if err != nil {
					errString = fmt.Sprintf("Error requesting %s run for model %s: %s", l.cmdName, model, err.Error())
					schedulingErrors = append(schedulingErrors, errString)
				} else {
					scheduledAnyBuilds = true
					l.printer.WriteTextStderr("Requesting %s run at %s", l.cmdName, l.bbClient.BuildURL(build.Id))
				}
				buildLaunchList.Launches = append(buildLaunchList.Launches, &crosfleetpb.BuildLaunch{
					Build:      build,
					BuildError: errString,
				})
				mutex.Unlock()
				waitGroup.Done()
			}()
		}
	}
	waitGroup.Wait()
	return
}

// confirmCTPBuildsAsync waits for all builds to start asynchronously, and
// updates the details for each build it confirms has started in the given build
// launch list. The function returns a bool indicating whether any builds were
// confirmed started, and a slice of confirmation error strings. Mutex locks are
// used to avoid race conditions from concurrent writes to the return variables
// in the async loop.
func (l *ctpRunLauncher) confirmCTPBuildsAsync(ctx context.Context, buildLaunchList *crosfleetpb.BuildLaunchList) (confirmedAnyBuilds bool, confirmationErrors []string) {
	waitGroup := sync.WaitGroup{}
	mutex := sync.Mutex{}
	for _, buildLaunch := range buildLaunchList.Launches {
		buildLaunch := buildLaunch
		// Only wait for builds that were already scheduled without issues.
		if buildLaunch.Build == nil || buildLaunch.Build.GetId() == 0 || buildLaunch.BuildError != "" {
			continue
		}
		waitGroup.Add(1)
		go func() {
			updatedBuild, err := l.bbClient.WaitForBuildStepStart(ctx, buildLaunch.Build.Id, ctpExecuteStepName)
			mutex.Lock()
			if updatedBuild != nil {
				buildLaunch.Build = updatedBuild
				if updatedBuild.Status == buildbucketpb.Status_STARTED {
					confirmedAnyBuilds = true
					l.printer.WriteTextStdout("Successfully started %s run %d", l.cmdName, updatedBuild.Id)
				}
			}
			if err != nil {
				errString := fmt.Sprintf("Error waiting for build %d to start: %s", buildLaunch.Build.Id, err.Error())
				buildLaunch.BuildError = errString
				confirmationErrors = append(confirmationErrors, errString)
			}
			mutex.Unlock()
			waitGroup.Done()
		}()
	}
	waitGroup.Wait()
	return
}

// testPlatformRequest constructs a cros_test_platform.Request from the given
// test plan, build tags, and command line flags.
func (l *ctpRunLauncher) testPlatformRequest(model string, buildTags map[string]string) (*test_platform.Request, error) {
	softwareDependencies, err := l.cliFlags.softwareDependencies()
	if err != nil {
		return nil, err
	}

	return &test_platform.Request{
		TestPlan: l.testPlan,
		Params: &test_platform.Request_Params{
			FreeformAttributes: &test_platform.Request_Params_FreeformAttributes{
				SwarmingDimensions: common.ToKeyvalSlice(l.cliFlags.addedDims),
			},
			HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
				Model: model,
			},
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{Name: l.cliFlags.board},
			},
			SoftwareDependencies: softwareDependencies,
			Scheduling:           l.cliFlags.schedulingParams(),
			Decorations: &test_platform.Request_Params_Decorations{
				AutotestKeyvals: l.cliFlags.keyvals,
				Tags:            common.ToKeyvalSlice(buildTags),
			},
			Retry: l.cliFlags.retryParams(),
			Metadata: &test_platform.Request_Params_Metadata{
				TestMetadataUrl:        imageArchiveBaseURL + l.cliFlags.image,
				DebugSymbolsArchiveUrl: imageArchiveBaseURL + l.cliFlags.image,
			},
			Time: &test_platform.Request_Params_Time{
				MaximumDuration: durationpb.New(
					time.Duration(l.cliFlags.timeoutMins) * time.Minute),
			},
		},
	}, nil
}

// softwareDependencies constructs test platform software dependencies from
// test run flags.
func (c *testCommonFlags) softwareDependencies() ([]*test_platform.Request_Params_SoftwareDependency, error) {
	// Add dependencies from provision labels.
	deps, err := softwareDepsFromProvisionLabels(c.provisionLabels)
	if err != nil {
		return nil, err
	}
	// Add build image dependency.
	if c.image != "" {
		deps = append(deps, &test_platform.Request_Params_SoftwareDependency{
			Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: c.image},
		})
	}
	// Add lacros path dependency.
	if c.lacrosPath != "" {
		deps = append(deps, &test_platform.Request_Params_SoftwareDependency{
			Dep: &test_platform.Request_Params_SoftwareDependency_LacrosGcsPath{LacrosGcsPath: c.lacrosPath},
		})
	}
	return deps, nil
}

// softwareDepsFromProvisionLabels parses the given provision labels into a
// []*test_platform.Request_Params_SoftwareDependency.
func softwareDepsFromProvisionLabels(labels map[string]string) ([]*test_platform.Request_Params_SoftwareDependency, error) {
	var deps []*test_platform.Request_Params_SoftwareDependency
	for label, value := range labels {
		dep := &test_platform.Request_Params_SoftwareDependency{}
		switch label {
		// These prefixes are interpreted by autotest's provisioning behavior;
		// they are defined in the autotest repo, at utils/labellib.py
		case "cros-version":
			dep.Dep = &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{
				ChromeosBuild: value,
			}
		case "fwro-version":
			dep.Dep = &test_platform.Request_Params_SoftwareDependency_RoFirmwareBuild{
				RoFirmwareBuild: value,
			}
		case "fwrw-version":
			dep.Dep = &test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild{
				RwFirmwareBuild: value,
			}
		default:
			return nil, errors.Reason("invalid provisionable label %s", label).Err()
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

// schedulingParams constructs Swarming scheduling params from test run flags.
func (c *testCommonFlags) schedulingParams() *test_platform.Request_Params_Scheduling {
	s := &test_platform.Request_Params_Scheduling{}

	if managedPool, isManaged := managedPool(c.pool); isManaged {
		s.Pool = &test_platform.Request_Params_Scheduling_ManagedPool_{ManagedPool: managedPool}
	} else {
		s.Pool = &test_platform.Request_Params_Scheduling_UnmanagedPool{UnmanagedPool: c.pool}
	}

	// Priority and Quota Scheduler account cannot coexist in a CTP request.
	// Only attach priority if no quota account is specified.
	if c.qsAccount != "" {
		s.QsAccount = c.qsAccount
	} else {
		s.Priority = c.priority
	}

	return s
}

// managedPool returns the test_platform.Request_Params_Scheduling_ManagedPool
// matching the given pool string, and returns false if no match was found.
func managedPool(pool string) (test_platform.Request_Params_Scheduling_ManagedPool, bool) {
	// Attempt to handle common pool name format discrepancies.
	pool = strings.ToUpper(pool)
	pool = strings.Replace(pool, "-", "_", -1)
	pool = strings.Replace(pool, "DUT_POOL_", "MANAGED_POOL_", 1)

	enum, ok := test_platform.Request_Params_Scheduling_ManagedPool_value[pool]
	if !ok {
		return 0, false
	}
	return test_platform.Request_Params_Scheduling_ManagedPool(enum), true
}

// schedulingParams constructs test retry params from test run flags.
func (c *testCommonFlags) retryParams() *test_platform.Request_Params_Retry {
	return &test_platform.Request_Params_Retry{
		Max:   int32(c.maxRetries),
		Allow: c.maxRetries != 0,
	}
}

// testOrSuiteNamesTag formats a label for the given test/suite names, to be
// added to the build tags of a cros_test_platform build launched for the given
// tests/suites.
func testOrSuiteNamesTag(names []string) string {
	if len(names) == 0 {
		panic("no test/suite names given")
	}
	var label string
	if len(names) > 1 {
		label = fmt.Sprintf("%v", names)
	} else {
		label = names[0]
	}
	if len(label) > maxSwarmingTagLength {
		return label[:maxSwarmingTagLength]
	}
	return label
}
