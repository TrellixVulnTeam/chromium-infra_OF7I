// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package args contains the logic for assembling all data required for
// creating an individual task request.
package args

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"infra/libs/skylab/inventory"
	"infra/libs/skylab/inventory/autotest/labels"
	"infra/libs/skylab/request"
	"infra/libs/skylab/worker"

	goconfig "go.chromium.org/chromiumos/config/go"
	testapi "go.chromium.org/chromiumos/config/go/test/api"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/golang/protobuf/ptypes"
	buildapi "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// Generator defines the set of inputs for creating a request.Args object.
type Generator struct {
	// Invocation describes test harness-level data and metadata.
	Invocation *steps.EnumerationResponse_AutotestInvocation
	// Params describes scheduling parameters and task-level metadata.
	Params *test_platform.Request_Params
	// WorkerConfig describes the skylab_swarming_worker-specific environment.
	WorkerConfig *config.Config_SkylabWorker
	// ParentTaskID is the Swarming ID of the CTP task.
	ParentTaskID string
	// ParentBuildID is the Buildbucket ID of the CTP build.
	ParentBuildID int64
	// ParentRequestUID is the UID of the CTP request which kicked off this
	// test run. This is needed for the analytics usage. Test execution
	// does not require this parameter.
	ParentRequestUID string
	Deadline         time.Time
	// PubSub channel to send test_runner status updates.
	StatusUpdateChannel *config.Config_PubSub
	// Experiments to pass on to test_runner builders.
	Experiments []string
	// The Gerrit Changes associated with the test_runner invocation.
	GerritChanges []*bbpb.GerritChange
}

// CheckConsistency checks the internal consistency of the various inputs to the
// argument generation logic.
func (g *Generator) CheckConsistency() error {
	el := g.enumerationInventoryLabels()

	rb := g.Params.GetSoftwareAttributes().GetBuildTarget().GetName()
	eb := el.GetBoard()
	if nonEmptyAndDifferent(rb, eb) {
		return errors.Reason("incompatible board dependency: request (%s) vs. enumeration (%s)", rb, eb).Err()
	}

	rm := g.Params.GetHardwareAttributes().GetModel()
	em := el.GetModel()
	if nonEmptyAndDifferent(rm, em) {
		return errors.Reason("incompatible model dependency: request (%s) vs. enumeration (%s)", rm, em).Err()
	}

	ud := g.getUnsupportedDependencies()
	if len(ud) > 0 {
		return errors.Reason("unsupported request dependencies: %s", strings.Join(ud, ", ")).Err()
	}

	return nil
}

func nonEmptyAndDifferent(a, b string) bool {
	return a != "" && b != "" && a != b
}

func (g *Generator) enumerationInventoryLabels() *inventory.SchedulableLabels {
	deps := g.Invocation.Test.Dependencies
	flatDims := make([]string, len(deps))
	for i, dep := range deps {
		flatDims[i] = dep.Label
	}
	return labels.Revert(flatDims)
}

func (g *Generator) getUnsupportedDependencies() []string {
	el := g.enumerationInventoryLabels()
	unsupported := stringset.New(len(g.Invocation.Test.Dependencies))
	for _, dep := range g.Invocation.Test.Dependencies {
		unsupported.Add(dep.Label)
	}
	for _, label := range labels.Convert(el) {
		unsupported.Del(label)
	}
	for _, label := range labels.IgnoredLabels() {
		unsupported.Del(label)
	}
	return unsupported.ToSlice()
}

// The interval of time during which Swarming will attempt to find a
// bot matching optional (i.e. provisionable) dimensions. After the
// expiration time Swarming will only use required dimensions for
// finding the bot.
// Has to be a multiple of a minute as per buildbucket requirements,
// see infra/appengine/cr-buildbucket/validation.py
const provisionableDimensionExpiration = time.Minute

// GenerateArgs generates request.Args, combining all the inputs to
// argsGenerator.
func (g *Generator) GenerateArgs(ctx context.Context) (request.Args, error) {
	isClient, err := g.isClientTest()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	// TODO(crbug.com/1162347) Delete after test_runner starts using structured
	// dependencies.
	provisionableDimensions, err := g.provisionableDimensions()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	timeout, err := g.timeout()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	kv := g.keyvals(ctx)

	cmd := &worker.Command{
		ClientTest:      isClient,
		Deadline:        g.Deadline,
		Keyvals:         kv,
		OutputToIsolate: true,
		TaskName:        g.Invocation.Test.Name,
		TestArgs:        g.Invocation.TestArgs,
	}
	cmd.Config(wrap(g.WorkerConfig))

	labels, err := g.inventoryLabels()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	buildTarget := g.Params.GetSoftwareAttributes().GetBuildTarget().GetName()
	containers := g.Params.GetExecutionParam().GetContainerMetadata().GetContainers()
	cftIsEnabled := false
	if containers != nil {
		_, cftIsEnabled = containers[buildTarget]
	}

	trr := &skylab_test_runner.Request{}
	cft_trr := &skylab_test_runner.CFTTestRequest{}
	if cftIsEnabled {
		cft_trr, err = g.cftTestRunnerRequest(ctx)
		if err != nil {
			return request.Args{}, errors.Annotate(err, "create request args").Err()
		}
	} else {
		trr, err = g.testRunnerRequest(ctx)
		if err != nil {
			return request.Args{}, errors.Annotate(err, "create request args").Err()
		}
	}

	return request.Args{
		Cmd:                              *cmd,
		SchedulableLabels:                labels,
		SecondaryDevicesLabels:           g.secondaryDevicesInventoryLabels(),
		Dimensions:                       g.Params.GetFreeformAttributes().GetSwarmingDimensions(),
		ParentTaskID:                     g.ParentTaskID,
		ParentRequestUID:                 g.ParentRequestUID,
		Priority:                         g.Params.GetScheduling().GetPriority(),
		ProvisionableDimensions:          provisionableDimensions,
		ProvisionableDimensionExpiration: provisionableDimensionExpiration,
		StatusTopic:                      pubSubTopicFullName(g.StatusUpdateChannel),
		SwarmingTags:                     g.swarmingTags(ctx, kv, cmd),
		TestRunnerRequest:                trr,
		CFTTestRunnerRequest:             cft_trr,
		CFTIsEnabled:                     cftIsEnabled,
		Timeout:                          timeout,
		Experiments:                      g.Experiments,
		GerritChanges:                    g.GerritChanges,
	}, nil
}

func pubSubTopicFullName(c *config.Config_PubSub) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("projects/%s/topics/%s", c.Project, c.Topic)
}

func (g *Generator) isClientTest() (bool, error) {
	switch g.Invocation.Test.ExecutionEnvironment {
	case buildapi.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT:
		return true, nil
	case buildapi.AutotestTest_EXECUTION_ENVIRONMENT_SERVER:
		return false, nil
	default:
		return false, errors.Reason("unknown exec environment %s", g.Invocation.Test.ExecutionEnvironment).Err()
	}
}

var poolMap = map[test_platform.Request_Params_Scheduling_ManagedPool]inventory.SchedulableLabels_DUTPool{
	test_platform.Request_Params_Scheduling_MANAGED_POOL_ARC_PRESUBMIT: inventory.SchedulableLabels_DUT_POOL_ARC_PRESUBMIT,
	test_platform.Request_Params_Scheduling_MANAGED_POOL_BVT:           inventory.SchedulableLabels_DUT_POOL_BVT,
	test_platform.Request_Params_Scheduling_MANAGED_POOL_CONTINUOUS:    inventory.SchedulableLabels_DUT_POOL_CONTINUOUS,
	test_platform.Request_Params_Scheduling_MANAGED_POOL_CQ:            inventory.SchedulableLabels_DUT_POOL_CQ,
	test_platform.Request_Params_Scheduling_MANAGED_POOL_CTS_PERBUILD:  inventory.SchedulableLabels_DUT_POOL_CTS_PERBUILD,
	test_platform.Request_Params_Scheduling_MANAGED_POOL_CTS:           inventory.SchedulableLabels_DUT_POOL_CTS,
	// TODO(akeshet): This mapping is inexact. Requests that specify a quota account should not
	// specify a pool, and should go routed to the quota pool automatically.
	test_platform.Request_Params_Scheduling_MANAGED_POOL_QUOTA:  inventory.SchedulableLabels_DUT_POOL_QUOTA,
	test_platform.Request_Params_Scheduling_MANAGED_POOL_SUITES: inventory.SchedulableLabels_DUT_POOL_SUITES,
}

func (g *Generator) inventoryLabels() (*inventory.SchedulableLabels, error) {
	inv := g.enumerationInventoryLabels()
	if g.Params.GetSoftwareAttributes().GetBuildTarget() != nil {
		*inv.Board = g.Params.SoftwareAttributes.BuildTarget.Name
	}
	if g.Params.GetHardwareAttributes().GetModel() != "" {
		*inv.Model = g.Params.HardwareAttributes.Model
	}

	priority := g.Params.GetScheduling().GetPriority()
	qs := g.Params.GetScheduling().GetQsAccount()
	if priority > 0 && qs != "" {
		panic(fmt.Sprintf("Priority and QsAccount should not both be set. Got Priority: %d and QsAccount: %s", priority, qs))
	}

	if p := g.Params.GetScheduling().GetPool(); p != nil {
		switch v := p.(type) {
		case *test_platform.Request_Params_Scheduling_ManagedPool_:
			pool, ok := poolMap[v.ManagedPool]
			if !ok {
				return nil, errors.Reason("unknown managed pool %s", v.ManagedPool.String()).Err()
			}
			inv.CriticalPools = append(inv.CriticalPools, pool)
		case *test_platform.Request_Params_Scheduling_UnmanagedPool:
			inv.SelfServePools = append(inv.SelfServePools, v.UnmanagedPool)
		default:
			panic(fmt.Sprintf("unhandled scheduling type %#v", p))
		}
	}

	if g.Params.GetHardwareAttributes().GetRequireStableDevice() {
		*inv.Stability = true
	}

	return inv, nil
}

// secondaryDevicesInventoryLabels construct SchedulableLabels for secondary devices
// based on test_platform.Request_Params.SecondaryDevices
func (g *Generator) secondaryDevicesInventoryLabels() []*inventory.SchedulableLabels {
	sds := g.Params.GetSecondaryDevices()
	var sInvLabels []*inventory.SchedulableLabels
	for _, sd := range sds {
		il := inventory.NewSchedulableLabels()
		if sd.GetSoftwareAttributes().GetBuildTarget() != nil {
			*il.Board = sd.SoftwareAttributes.BuildTarget.Name
		}
		if sd.GetHardwareAttributes().GetModel() != "" {
			*il.Model = sd.HardwareAttributes.Model
		}
		sInvLabels = append(sInvLabels, il)
	}
	return sInvLabels
}

const (
	// Names of swarming dimensions corresponding to cached provision state of
	// various Chrome OS components.
	// These names must stay consistent with the names used in
	// cros/phosphorus/internal/botcache
	dimChromeOS      = "provisionable-cros-version"
	dimFirmwareRO    = "provisionable-fwro-version"
	dimFirmwareRW    = "provisionable-fwrw-version"
	dimLacrosGCSPath = "provisionable-lacros-gcs-path"
)

func (g *Generator) provisionableDimensions() ([]string, error) {
	deps := g.Params.SoftwareDependencies
	builds, err := extractBuilds(deps)
	if err != nil {
		return nil, errors.Annotate(err, "get provisionable dimensions").Err()
	}

	var dims []string
	if b := builds.ChromeOS; b != "" {
		dims = append(dims, dimChromeOS+":"+b)
	}
	if b := builds.FirmwareRO; b != "" {
		dims = append(dims, dimFirmwareRO+":"+b)
	}
	if b := builds.FirmwareRW; b != "" {
		dims = append(dims, dimFirmwareRW+":"+b)
	}
	if b := builds.LacrosGCSPath; b != "" {
		dims = append(dims, dimLacrosGCSPath+":"+b)
	}
	return dims, nil
}

func (g *Generator) timeout() (time.Duration, error) {
	if g.Params.Time == nil {
		return 0, errors.Reason("get timeout: nil params.time").Err()
	}
	duration, err := ptypes.Duration(g.Params.Time.MaximumDuration)
	if err != nil {
		return 0, errors.Annotate(err, "get timeout").Err()
	}
	return duration, nil
}

func (g *Generator) displayName(ctx context.Context, kv map[string]string) string {
	if g.Invocation.DisplayName != "" {
		return g.Invocation.DisplayName
	}
	return g.constructDisplayNameFromRequestParams(ctx, kv)
}

const (
	suiteKey         = "suite"
	defaultSuiteName = "cros_test_platform"
)

// This is a hack to satisfy tko/parse's insistence on parsing the display name
// (aka "label") keyval to obtain semantic information about the request.
// TODO(crbug.com/1003490): Drop this once result reporting is updated to stop
// parsing the "label" keyval.
func (g *Generator) constructDisplayNameFromRequestParams(ctx context.Context, kv map[string]string) string {
	testName := g.Invocation.GetTest().GetName()
	builds, err := extractBuilds(g.Params.SoftwareDependencies)
	if err != nil {
		logging.Warningf(ctx,
			"Failed to get build due to error %s\n Defaulting to test name as display name: %s",
			err.Error(), testName)
		return testName
	}

	build := builds.ChromeOS
	if build == "" {
		logging.Warningf(ctx, "Build missing. Defaulting to test name as display name: %s", testName)
		return testName
	}

	suite := kv[suiteKey]
	if suite == "" {
		suite = defaultSuiteName
	}

	return build + "/" + suite + "/" + testName
}

const displayNameKey = "label"

func (g *Generator) keyvals(ctx context.Context) map[string]string {
	kv := g.baseKeyvals()
	g.updateWithInvocationKeyvals(kv)
	kv[displayNameKey] = g.displayName(ctx, kv)
	return kv
}

func (g *Generator) updateWithInvocationKeyvals(kv map[string]string) {
	for k, v := range g.Invocation.GetResultKeyvals() {
		if _, ok := kv[k]; !ok {
			kv[k] = v
		}
	}
}

func (g *Generator) baseKeyvals() map[string]string {
	keyvals := make(map[string]string)
	for k, v := range g.Params.GetDecorations().GetAutotestKeyvals() {
		keyvals[k] = v
	}
	if g.ParentTaskID != "" {
		// This keyval is inspected by some downstream results consumers such as
		// goldeneye and stainless.
		// TODO(akeshet): Consider whether parameter-specified parent_job_id
		// should be respected if it was specified.
		keyvals["parent_job_id"] = g.ParentTaskID
	}
	// These build related keyvals are used by gs_offlaoder's CTS results
	// offload hook.
	for _, sd := range g.Params.GetSoftwareDependencies() {
		if b := sd.GetChromeosBuild(); b != "" {
			keyvals["build"] = b
		}
		if b := sd.GetRwFirmwareBuild(); b != "" {
			keyvals["fwrw_build"] = b
		}
		if b := sd.GetRoFirmwareBuild(); b != "" {
			keyvals["fwro_build"] = b
		}
	}
	return keyvals
}

func (g *Generator) testargs() string {
	ta := g.Invocation.TestArgs
	// TODO(b/200273237): Remove check for experiment once uploading results to
	// resultdb is stable.
	addRdbSettings := false
	for _, x := range g.Experiments {
		if strings.EqualFold(x, "chromeos.cros_test_platform.add_resultdb_settings") {
			addRdbSettings = true
		}
	}
	for k, v := range g.Params.GetDecorations().GetTestArgs() {
		if k == "resultdb_settings" && !addRdbSettings {
			continue
		}
		ta += fmt.Sprintf(" %s=%s", k, v)
	}
	return ta
}

var reservedTags = map[string]bool{
	"qs_account":   true,
	"luci_project": true,
	"log_location": true,
}

func (g *Generator) swarmingTags(ctx context.Context, kv map[string]string, cmd *worker.Command) []string {
	tags := []string{
		"luci_project:" + g.WorkerConfig.LuciProject,
		"log_location:" + cmd.LogDogAnnotationURL,
	}
	// CTP "builds" triggered by `led` don't have a buildbucket ID.
	if g.ParentBuildID != 0 {
		tags = append(tags, "parent_buildbucket_id:"+strconv.FormatInt(g.ParentBuildID, 10))
	}
	tags = append(tags, "display_name:"+g.displayName(ctx, kv))
	if qa := g.Params.GetScheduling().GetQsAccount(); qa != "" {
		tags = append(tags, "qs_account:"+qa)
	}
	tags = append(tags, removeReservedTags(g.Params.GetDecorations().GetTags())...)
	// Add primary/secondary DUTs board/model info in swarming tags for
	// multi-DUTs result reporting purpose.
	tags = append(tags, g.multiDutsTags()...)
	return tags
}

// multiDutsTags generate swarming tags based on primary/secondary DUTs
// board/model info from request.
func (g *Generator) multiDutsTags() []string {
	var tags []string
	if g.Params.GetSoftwareAttributes().GetBuildTarget() != nil {
		tags = append(tags, fmt.Sprintf("primary_board:%s", g.Params.SoftwareAttributes.BuildTarget.Name))
	}
	if g.Params.GetHardwareAttributes().GetModel() != "" {
		tags = append(tags, fmt.Sprintf("primary_model:%s", g.Params.HardwareAttributes.Model))
	}
	sds := g.Params.GetSecondaryDevices()
	var secondary_boards []string
	var secondary_models []string
	for _, sd := range sds {
		if sd.GetSoftwareAttributes().GetBuildTarget() != nil {
			secondary_boards = append(secondary_boards, sd.SoftwareAttributes.BuildTarget.Name)
		}
		if sd.GetHardwareAttributes().GetModel() != "" {
			secondary_models = append(secondary_models, sd.HardwareAttributes.Model)
		}
	}
	if len(secondary_boards) > 0 {
		boards := strings.Join(secondary_boards, ",")
		tags = append(tags, fmt.Sprintf("secondary_boards:%s", boards))
	}
	if len(secondary_models) > 0 {
		models := strings.Join(secondary_models, ",")
		tags = append(tags, fmt.Sprintf("secondary_models:%s", models))
	}
	return tags
}

// removeReservedTags removes the reserved tags attached by users.
func removeReservedTags(tags []string) []string {
	res := make([]string, 0, len(tags))
	for _, tag := range tags {
		keyval := strings.Split(tag, ":")
		if _, isReserved := reservedTags[keyval[0]]; isReserved {
			continue
		}
		res = append(res, tag)
	}
	return res
}

// builds describes the build names that were requested by a test_platform
// invocation.
type builds struct {
	ChromeOS       string
	ChromeOSBucket string
	FirmwareRW     string
	FirmwareRO     string
	LacrosGCSPath  string
}

// extractBuilds extracts builds that were requested by the test_platform invocation.
func extractBuilds(deps []*test_platform.Request_Params_SoftwareDependency) (*builds, error) {
	b := &builds{}
	for _, dep := range deps {
		switch d := dep.Dep.(type) {
		case *test_platform.Request_Params_SoftwareDependency_ChromeosBuild:
			if already := b.ChromeOS; already != "" {
				return nil, errors.Reason("duplicate ChromeOS builds (%s, %s)", already, d.ChromeosBuild).Err()
			}
			b.ChromeOS = d.ChromeosBuild
		case *test_platform.Request_Params_SoftwareDependency_ChromeosBuildGcsBucket:
			if already := b.ChromeOSBucket; already != "" {
				return nil, errors.Reason("duplicate ChromeOS buckets (%s, %s)", already, d.ChromeosBuildGcsBucket).Err()
			}
			b.ChromeOSBucket = d.ChromeosBuildGcsBucket
		case *test_platform.Request_Params_SoftwareDependency_RoFirmwareBuild:
			if already := b.FirmwareRO; already != "" {
				return nil, errors.Reason("duplicate RO Firmware builds (%s, %s)", already, d.RoFirmwareBuild).Err()
			}
			b.FirmwareRO = d.RoFirmwareBuild
		case *test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild:
			if already := b.FirmwareRW; already != "" {
				return nil, errors.Reason("duplicate RW Firmware builds (%s, %s)", already, d.RwFirmwareBuild).Err()
			}
			b.FirmwareRW = d.RwFirmwareBuild
		case *test_platform.Request_Params_SoftwareDependency_LacrosGcsPath:
			if already := b.LacrosGCSPath; already != "" {
				return nil, errors.Reason("duplicate Lacros paths (%s, %s)", already, d.LacrosGcsPath).Err()
			}
			b.LacrosGCSPath = d.LacrosGcsPath
		default:
			return nil, errors.Reason("unknown dep %+v", dep).Err()
		}
	}
	return b, nil
}

func (g *Generator) testRunnerRequest(ctx context.Context) (*skylab_test_runner.Request, error) {
	isClient, err := g.isClientTest()
	if err != nil {
		return nil, errors.Annotate(err, "create test runner request").Err()
	}
	kv := g.keyvals(ctx)
	ta := g.testargs()

	var deadline *timestamppb.Timestamp
	if !g.Deadline.IsZero() {
		deadline = timestamppb.New(g.Deadline)
	}

	return &skylab_test_runner.Request{
		Deadline: deadline,
		Prejob: &skylab_test_runner.Request_Prejob{
			SoftwareDependencies: g.Params.SoftwareDependencies,
			SoftwareAttributes:   g.Params.SoftwareAttributes,
			HardwareAttributes:   g.Params.HardwareAttributes,
			SecondaryDevices:     g.Params.SecondaryDevices,
		},
		// The hard coded "original_test" key is ignored in test_runner builds.
		// All behavior will remain the same, until we start running multiple tests per test_runner build.
		Tests: map[string]*skylab_test_runner.Request_Test{
			"original_test": {
				Harness: &skylab_test_runner.Request_Test_Autotest_{
					Autotest: &skylab_test_runner.Request_Test_Autotest{
						DisplayName:  g.displayName(ctx, kv),
						IsClientTest: isClient,
						Name:         g.Invocation.Test.Name,
						Keyvals:      kv,
						TestArgs:     ta,
					},
				},
			},
		},
		ParentRequestUid:             g.ParentRequestUID,
		ParentBuildId:                g.ParentBuildID,
		ExecutionParam:               g.Params.ExecutionParam,
		DefaultTestExecutionBehavior: g.Params.TestExecutionBehavior,
	}, nil
}

// cftTestRunnerRequest creates test runner request for cft workflow
func (g *Generator) cftTestRunnerRequest(ctx context.Context) (*skylab_test_runner.CFTTestRequest, error) {
	kv := g.keyvals(ctx)

	var deadline *timestamppb.Timestamp
	if !g.Deadline.IsZero() {
		deadline = timestamppb.New(g.Deadline)
	}

	builds, err := extractBuilds(g.Params.GetSoftwareDependencies())
	if err != nil {
		return nil, errors.Annotate(err, "create cft test runner request: extractBuilds").Err()
	}

	imagePath := ""
	if builds.LacrosGCSPath != "" {
		imagePath = builds.LacrosGCSPath
	} else if builds.ChromeOSBucket != "" && builds.ChromeOS != "" {
		imagePath = fmt.Sprintf("gs://%s/%s", builds.ChromeOSBucket, builds.ChromeOS)
	}
	if imagePath == "" {
		return nil, errors.Annotate(err, "create cft test runner request: imagePath").Err()
	}

	provsionState := &testapi.ProvisionState{
		SystemImage: &testapi.ProvisionState_SystemImage{
			SystemImagePath: &goconfig.StoragePath{
				HostType: goconfig.StoragePath_GS,
				Path:     imagePath,
			},
		},
	}

	buildTarget := g.Params.GetSoftwareAttributes().GetBuildTarget().GetName()
	dutModel := &labapi.DutModel{
		BuildTarget: buildTarget,
		ModelName:   g.Params.GetHardwareAttributes().GetModel(),
	}

	testSuites := []*testapi.TestSuite{
		{
			Name: kv["suite"],
			Spec: &testapi.TestSuite_TestCaseIds{
				TestCaseIds: &testapi.TestCaseIdList{
					TestCaseIds: []*testapi.TestCase_Id{
						{
							Value: fmt.Sprintf("tauto.%s", g.Invocation.Test.Name),
						},
					},
				},
			},
		},
	}
	// TODO(b/220801220): Pass in companion duts info for multi-duts cases.
	return &skylab_test_runner.CFTTestRequest{
		Deadline:         deadline,
		ParentRequestUid: g.ParentRequestUID,
		ParentBuildId:    g.ParentBuildID,
		PrimaryDut: &skylab_test_runner.CFTTestRequest_Device{
			DutModel:             dutModel,
			ProvisionState:       provsionState,
			ContainerMetadataKey: buildTarget,
		},
		ContainerMetadata:            g.Params.GetExecutionParam().GetContainerMetadata(),
		TestSuites:                   testSuites,
		DefaultTestExecutionBehavior: g.Params.GetTestExecutionBehavior(),
		AutotestKeyvals:              kv,
	}, nil
}
