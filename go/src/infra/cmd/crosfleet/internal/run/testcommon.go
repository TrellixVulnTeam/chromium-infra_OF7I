// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"context"
	"flag"
	"fmt"
	"infra/cmdsupport/cmdlib"
	"io"
	"strings"
	"time"

	"go.chromium.org/luci/auth/client/authcli"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"

	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/flagx"

	"github.com/golang/protobuf/ptypes"
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
)

// testCommonFlags contains parameters common to the "run
// test", "run suite", and "run testplan" subcommands.
type testCommonFlags struct {
	board           string
	model           string
	pool            string
	image           string
	qsAccount       string
	maxRetries      int
	priority        int64
	timeoutMins     int
	addedDims       map[string]string
	provisionLabels map[string]string
	addedTags       map[string]string
	keyvals         map[string]string
	json            bool
}

// Registers run command-specific flags
func (c *testCommonFlags) register(f *flag.FlagSet) {
	f.StringVar(&c.image, "image", "", `Optional fully specified image name to run test against, e.g. octopus-release/R89-13609.0.0.
If no value is passed, test will run against the latest green build for the given board.`)
	f.StringVar(&c.board, "board", "", "Board to run tests on.")
	f.StringVar(&c.model, "model", "", "Model to run tests on.")
	f.StringVar(&c.pool, "pool", "", "Device pool to run tests on.")
	f.StringVar(&c.qsAccount, "qs-account", "", `Optional Quota Scheduler account to use for this task. Overrides -priority flag.
If no account is set, tests are scheduled using -priority flag.`)
	f.IntVar(&c.maxRetries, "max-retries", 0, "Maximum retries allowed. No retry if set to 0.")
	f.Int64Var(&c.priority, "priority", DefaultSwarmingPriority, `Swarming scheduling priority for tests, between 50 and 255 (lower values indicate higher priorities).
If a Quota Scheduler account is specified via -qs-account, this value is not used.`)
	f.IntVar(&c.timeoutMins, "timeout-mins", 30, "Test run timeout.")
	f.Var(flagx.KeyVals(&c.addedDims), "dim", "Additional scheduling dimension in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.addedDims), "dims", "Comma-separated additional scheduling addedDims in same format as -dim.")
	f.Var(flagx.KeyVals(&c.provisionLabels), "provision-label", "Additional provisionable label in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.provisionLabels), "provision-labels", "Comma-separated additional provisionable labels in same format as -provision-label.")
	f.Var(flagx.KeyVals(&c.addedTags), "tag", "Additional Swarming tag in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.addedTags), "tags", "Comma-separated Swarming tags in same format as -tag.")
	f.Var(flagx.KeyVals(&c.keyvals), "autotest-keyval", "Autotest keyval in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.keyvals), "autotest-keyvals", "Comma-separated Autotest keyvals in same format as -keyval.")
	f.BoolVar(&c.json, "json", false, "Format output as JSON.")
}

// validateAndAutocompleteFlags returns any errors after validating the CLI
// flags, and autocompletes the -image flag unless it was specified by the user.
func (c *testCommonFlags) validateAndAutocompleteFlags(ctx context.Context, f *flag.FlagSet, mainArgType, bbService string, authFlags authcli.Flags, writer io.Writer) error {
	if err := c.validateArgs(f, mainArgType); err != nil {
		return err
	}
	// If no image was specified, determine the latest green image for the
	// given board.
	if c.image == "" {
		latestImage, err := latestImage(ctx, c.board, bbService, authFlags)
		if err != nil {
			return fmt.Errorf("error determining the latest image for board %s: %v", c.board, err)
		}
		fmt.Fprintf(writer, "Using latest green build image %s for board %s\n", latestImage, c.board)
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
		panic(errors.Reason("must provide crosfleet-tool tag").Err())
	}
	tags["crosfleet-tool"] = crosfleetTool
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

// launchCTPBuild uses the given Buildbucket client to launch a
// cros_test_platform Buildbucket build for the given test plan, build tags,
// and command line flags, and returns the ID of the launched build.
func launchCTPBuild(ctx context.Context, bbClient *buildbucket.Client, testPlan *test_platform.Request_TestPlan, buildTags map[string]string, cliFlags *testCommonFlags) (int64, error) {
	ctpRequest, err := testPlatformRequest(testPlan, buildTags, cliFlags)
	if err != nil {
		return 0, err
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
	return bbClient.ScheduleBuild(ctx, buildProps, nil, buildTags, 0)
}

// testPlatformRequest constructs a cros_test_platform.Request from the given
// test plan, build tags, and command line flags.
func testPlatformRequest(testPlan *test_platform.Request_TestPlan, buildTags map[string]string, cliFlags *testCommonFlags) (*test_platform.Request, error) {
	softwareDependencies, err := cliFlags.softwareDependencies()
	if err != nil {
		return nil, err
	}

	return &test_platform.Request{
		TestPlan: testPlan,
		Params: &test_platform.Request_Params{
			FreeformAttributes: &test_platform.Request_Params_FreeformAttributes{
				SwarmingDimensions: common.ToKeyvalSlice(cliFlags.addedDims),
			},
			HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
				Model: cliFlags.model,
			},
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{Name: cliFlags.board},
			},
			SoftwareDependencies: softwareDependencies,
			Scheduling:           cliFlags.schedulingParams(),
			Decorations: &test_platform.Request_Params_Decorations{
				AutotestKeyvals: cliFlags.keyvals,
				Tags:            common.ToKeyvalSlice(buildTags),
			},
			Retry: cliFlags.retryParams(),
			Metadata: &test_platform.Request_Params_Metadata{
				TestMetadataUrl:        imageArchiveBaseURL + cliFlags.image,
				DebugSymbolsArchiveUrl: imageArchiveBaseURL + cliFlags.image,
			},
			Time: &test_platform.Request_Params_Time{
				MaximumDuration: ptypes.DurationProto(
					time.Duration(cliFlags.timeoutMins) * time.Minute),
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
	return label
}
