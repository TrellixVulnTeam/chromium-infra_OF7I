// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"context"
	"flag"
	"fmt"
	"infra/cmdsupport/cmdlib"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"go.chromium.org/luci/auth/client/authcli"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"

	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/flagx"

	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/common/errors"
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
)

// testCommonFlags contains parameters common to the "run
// test", "run suite", and "run testplan" subcommands.
type testCommonFlags struct {
	board           string
	model           string
	pool            string
	image           string
	release         string
	qsAccount       string
	maxRetries      int
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
	f.StringVar(&c.model, "model", "", "Model to run tests on.")
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

// buildTags combines test metadata tags with user-added tags.
func (c *testCommonFlags) buildTags(crosfleetTool string, mainArg string) map[string]string {
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
	if c.model != "" {
		tags["label-model"] = c.model
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
	printer   common.CLIPrinter
	cmdName   string
	bbClient  *buildbucket.Client
	testPlan  *test_platform.Request_TestPlan
	buildTags map[string]string
	cliFlags  *testCommonFlags
	exitEarly bool
}

// launchAndValidateTestPlan requests a run of the given CTP run launcher's
// test plan, and returns the ID of the launched cros_test_platform Buildbucket
// build. Unless the exitEarly arg is passed as true, the function waits to
// return until the build passes request-validation and setup steps.
func (l *ctpRunLauncher) launchAndValidateTestPlan(ctx context.Context) error {
	ctpBuild, err := l.launchCTPBuild(ctx)
	if err != nil {
		return err
	}
	l.printer.WriteTextStderr("Requesting %s run at %s", l.cmdName, l.bbClient.BuildURL(ctpBuild.Id))
	if !l.exitEarly {
		l.printer.WriteTextStderr("Waiting to confirm %s run request validation...\n(To skip this step, pass the -exit-early flag on future %s run commands)", l.cmdName, l.cmdName)
		ctpBuild, err = l.bbClient.WaitForBuildStepStart(ctx, ctpBuild.Id, ctpExecuteStepName)
		if err != nil {
			return err
		}
		l.printer.WriteTextStdout("Successfully started %s run", l.cmdName)
	}
	l.printer.WriteJSONStdout(ctpBuild)
	return nil
}

// launchCTPBuild uses the given Buildbucket client to launch a
// cros_test_platform Buildbucket build for the CTP run launcher's test plan,
// build tags, and command line flags, and returns the ID of the launched build.
func (l *ctpRunLauncher) launchCTPBuild(ctx context.Context) (*buildbucketpb.Build, error) {
	ctpRequest, err := l.testPlatformRequest()
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
	return l.bbClient.ScheduleBuild(ctx, buildProps, nil, l.buildTags, 0)
}

// testPlatformRequest constructs a cros_test_platform.Request from the given
// test plan, build tags, and command line flags.
func (l *ctpRunLauncher) testPlatformRequest() (*test_platform.Request, error) {
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
				Model: l.cliFlags.model,
			},
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{Name: l.cliFlags.board},
			},
			SoftwareDependencies: softwareDependencies,
			Scheduling:           l.cliFlags.schedulingParams(),
			Decorations: &test_platform.Request_Params_Decorations{
				AutotestKeyvals: l.cliFlags.keyvals,
				Tags:            common.ToKeyvalSlice(l.buildTags),
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

// testOrSuiteNamesLabel formats a label for the given test/suite names, to be
// added to the build tags of a cros_test_platform build launched for the given
// tests/suites.
func testOrSuiteNamesLabel(names []string) string {
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
