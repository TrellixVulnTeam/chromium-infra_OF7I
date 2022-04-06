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
	"infra/cmd/crosfleet/internal/ufs"
	"infra/cmdsupport/cmdlib"
	"math"
	"strings"
	"sync"
	"time"

	ufsapi "infra/unifiedfleet/api/v1/rpc"

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
	board                string
	secondaryBoards      []string
	models               []string
	secondaryModels      []string
	pool                 string
	image                string
	secondaryImages      []string
	release              string
	qsAccount            string
	maxRetries           int
	repeats              int
	priority             int64
	timeoutMins          int
	addedDims            map[string]string
	provisionLabels      map[string]string
	addedTags            map[string]string
	keyvals              map[string]string
	exitEarly            bool
	lacrosPath           string
	secondaryLacrosPaths []string
}

type fleetValidationResults struct {
	anyValidTests        bool
	validTests           []string
	validModels          []string
	testValidationErrors []string
}

// Registers run command-specific flags
func (c *testCommonFlags) register(f *flag.FlagSet) {
	f.StringVar(&c.image, "image", "", `Optional fully specified image name to run test against, e.g. octopus-release/R89-13609.0.0.
If no value for image or release is passed, test will run against the latest green postsubmit build for the given board.`)
	f.Var(luciflag.CommaList(&c.secondaryImages), "secondary-images", "Comma-separated list of image name(or 'skip' if no provision needed for a secondary dut) for secondary DUTs to run tests against, it need to align with boards in secondary-boards args.")
	f.StringVar(&c.release, "release", "", `Optional ChromeOS release branch to run test against, e.g. R89-13609.0.0.
If no value for image or release is passed, test will run against the latest green postsubmit build for the given board.`)
	f.StringVar(&c.board, "board", "", "Board to run tests on.")
	f.Var(luciflag.CommaList(&c.secondaryBoards), "secondary-boards", "Comma-separated list of boards for secondary DUTs to run tests on, a.k.a multi-DUTs testing.")
	f.Var(luciflag.StringSlice(&c.models), "model", fmt.Sprintf(`Model to run tests on; may be specified multiple times.
A maximum of %d tests may be launched per "crosfleet run" command.`, maxCTPRunsPerCmd))
	f.Var(luciflag.CommaList(&c.models), "models", "Comma-separated list of models to run tests on in same format as -model.")
	f.Var(luciflag.CommaList(&c.secondaryModels), "secondary-models", "Comma-separated list of models for secondary DUTs to run tests on, if provided it need to align with boards in secondary-boards args.")
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
	f.Var(luciflag.CommaList(&c.secondaryLacrosPaths), "secondary-lacros-paths", "Comma-separated list of lacros paths for secondary DUTs to run tests against, it need to align with boards in secondary-boards args.")
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
	// For multi-DUTs result reporting purpose we need board info, so even if
	// explicit secondary models request, we need to ensure board info is also
	// provided and the count matches.
	if len(c.secondaryModels) > 0 && len(c.secondaryBoards) != len(c.secondaryModels) {
		errors = append(errors, fmt.Sprintf("number of requested secondary-boards: %d does not match with number of requested secondary-models: %d", len(c.secondaryBoards), len(c.secondaryModels)))
	}
	// Check if image name provided for each secondary devices.
	if len(c.secondaryBoards) != len(c.secondaryImages) {
		errors = append(errors, fmt.Sprintf("number of requested secondary-boards: %d does not match with number of requested secondary-images: %d", len(c.secondaryBoards), len(c.secondaryImages)))
	}

	// If lacros provision required for secondary DUTs, then we require provide a path for each secondary DUT.
	if len(c.secondaryLacrosPaths) > 0 && len(c.secondaryLacrosPaths) != len(c.secondaryBoards) {
		errors = append(errors, fmt.Sprintf("number of requested secondary-boards: %d does not match with number of requested secondary-lacros-paths: %d", len(c.secondaryBoards), len(c.secondaryLacrosPaths)))
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

	request := &test_platform.Request{
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
	}
	// Handling multi-DUTs use case if secondaryBoards provided.
	if len(l.cliFlags.secondaryBoards) > 0 {
		request.Params.SecondaryDevices = l.cliFlags.secondaryDevices()
	}
	return request, nil
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

// secondaryDevices constructs secondary devices data for a test platform
// request from test run flags.
func (c *testCommonFlags) secondaryDevices() []*test_platform.Request_Params_SecondaryDevice {
	var secondary_devices []*test_platform.Request_Params_SecondaryDevice
	for i, b := range c.secondaryBoards {
		sd := &test_platform.Request_Params_SecondaryDevice{
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{Name: b},
			},
		}
		if strings.ToLower(c.secondaryImages[i]) != "skip" {
			sd.SoftwareDependencies = append(sd.SoftwareDependencies, &test_platform.Request_Params_SoftwareDependency{
				Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: c.secondaryImages[i]},
			})
		}
		if len(c.secondaryModels) > 0 {
			sd.HardwareAttributes = &test_platform.Request_Params_HardwareAttributes{
				Model: c.secondaryModels[i],
			}
		}
		if len(c.secondaryLacrosPaths) > 0 {
			sd.SoftwareDependencies = append(sd.SoftwareDependencies, &test_platform.Request_Params_SoftwareDependency{
				Dep: &test_platform.Request_Params_SoftwareDependency_LacrosGcsPath{LacrosGcsPath: c.secondaryLacrosPaths[i]},
			})
		}
		secondary_devices = append(secondary_devices, sd)
	}
	return secondary_devices
}

// verifyFleetTestsPolicy validate tests based on fleet-side permission check.
//
// This method calls UFS CheckFleetTestsPolicy RPC for each testName, board, image and model combination.
// The test run stops if an invalid board or image is specified.
// After this validation only valid models and tests will be used in the run command.
func (c *testCommonFlags) verifyFleetTestsPolicy(ctx context.Context, ufsClient ufs.Client, cmdName string,
	testNames []string, allowPublicUserAccount bool) (*fleetValidationResults, error) {
	validTestNamesMap := map[string]bool{}
	validModelsMap := map[string]bool{}
	results := &fleetValidationResults{}

	// Calling the UFS CheckFleetTestsPolicy with empty test params.
	// For a user account which runs private tests there is no validation on the UFS side so the CheckFleetTestsPolicy will return a valid test response for empty test params.
	// If UFS returns an OK status for this RPC then it means that the service account is not something that is used to run public tests so we can skip further validation.
	// This check is to avoid unnecessary RPC calls to UFS for tests run by service accounts meant for private tests.
	isPublicTestResponse, err := ufsClient.CheckFleetTestsPolicy(ctx, &ufsapi.CheckFleetTestsPolicyRequest{})
	if err != nil && !allowPublicUserAccount {
		return nil, fmt.Errorf("Public user service accounts are not allowed to run %s run(s)",
			cmdName)
	}
	if isPublicTestResponse != nil && isPublicTestResponse.TestStatus.Code == ufsapi.TestStatus_OK {
		results.anyValidTests = true
		results.validModels = c.models
		results.validTests = testNames
		return results, nil
	}

	if len(c.models) == 0 {
		return nil, fmt.Errorf("model is required for public users' crosfleet %s run(s)",
			cmdName)
	}
	for _, model := range c.models {
		for _, testName := range testNames {
			resp, err := ufsClient.CheckFleetTestsPolicy(ctx, &ufsapi.CheckFleetTestsPolicyRequest{
				TestName: testName,
				Board:    c.board,
				Model:    model,
				Image:    c.image,
			})
			if err != nil {
				results.testValidationErrors = append(results.testValidationErrors, err.Error())
				continue
			}
			if resp.TestStatus.Code == ufsapi.TestStatus_OK {
				results.anyValidTests = true
				validModelsMap[model] = true
				validTestNamesMap[testName] = true
				continue
			}
			if resp.TestStatus.Code == ufsapi.TestStatus_NOT_A_PUBLIC_BOARD || resp.TestStatus.Code == ufsapi.TestStatus_NOT_A_PUBLIC_IMAGE {
				// No tests can be run with Invalid Board or Image so returning early to avoid unnecessary calls to UFS
				// results.anyValidTests = false
				return nil, fmt.Errorf(resp.TestStatus.Message)
			}
			results.testValidationErrors = append(results.testValidationErrors, resp.TestStatus.Message)
		}
	}

	for test := range validTestNamesMap {
		results.validTests = append(results.validTests, test)
	}
	for model := range validModelsMap {
		results.validModels = append(results.validModels, model)
	}

	return results, nil
}

func checkAndPrintFleetValidationErrors(results fleetValidationResults, printer common.CLIPrinter, cmdName string) error {
	if len(results.testValidationErrors) > 0 {
		fullErrorMsg := fmt.Sprintf("Encountered the following errors requesting %s run(s):\n%s\n",
			cmdName, strings.Join(results.testValidationErrors, "\n"))
		if results.anyValidTests {
			// Don't fail the command if we were able to request some runs.
			printer.WriteTextStderr(fullErrorMsg)
		} else {
			return fmt.Errorf(fullErrorMsg)
		}
	}
	return nil
}
