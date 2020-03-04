// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package args contains the logic for assembling all data required for
// creating an individual task request.
package args

import (
	"context"
	"fmt"
	"infra/libs/skylab/inventory"
	"infra/libs/skylab/inventory/autotest/labels"
	"infra/libs/skylab/request"
	"infra/libs/skylab/worker"
	"strings"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"

	"github.com/golang/protobuf/ptypes"
	build_api "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"
)

// Generator defines the set of inputs for creating a request.Args object.
type Generator struct {
	// invocation describes test harness-level data and metadata.
	invocation *steps.EnumerationResponse_AutotestInvocation
	// params describes scheduling parameters and task-level metadata.
	params *test_platform.Request_Params
	// workerConfig describes the skylab_swarming_worker-specific environment.
	workerConfig *config.Config_SkylabWorker
	// parentTaskID is the Swarming ID of the CTP task.
	parentTaskID string
	deadline     time.Time
}

// NewGenerator constructs an args Generator.
func NewGenerator(invocation *steps.EnumerationResponse_AutotestInvocation, params *test_platform.Request_Params, workerConfig *config.Config_SkylabWorker, parentTaskID string, deadline time.Time) *Generator {
	return &Generator{
		invocation:   invocation,
		params:       params,
		workerConfig: workerConfig,
		parentTaskID: parentTaskID,
		deadline:     deadline,
	}
}

// CheckConsistency checks the internal consistency of the various inputs to the
// argument generation logic.
func (g *Generator) CheckConsistency() error {
	el := g.enumerationInventoryLabels()

	rb := g.params.GetSoftwareAttributes().GetBuildTarget().GetName()
	eb := el.GetBoard()
	if nonEmptyAndDifferent(rb, eb) {
		return errors.Reason("incompatible board dependency: request (%s) vs. enumeration (%s)", rb, eb).Err()
	}

	rm := g.params.GetHardwareAttributes().GetModel()
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
	deps := g.invocation.Test.Dependencies
	flatDims := make([]string, len(deps))
	for i, dep := range deps {
		flatDims[i] = dep.Label
	}
	return labels.Revert(flatDims)
}

func (g *Generator) getUnsupportedDependencies() []string {
	el := g.enumerationInventoryLabels()
	unsupported := stringset.New(len(g.invocation.Test.Dependencies))
	for _, dep := range g.invocation.Test.Dependencies {
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

// GenerateArgs generates request.Args, combining all the inputs to
// argsGenerator.
func (g *Generator) GenerateArgs(ctx context.Context) (request.Args, error) {
	isClient, err := g.isClientTest()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

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
		Deadline:        g.deadline,
		Keyvals:         kv,
		OutputToIsolate: true,
		TaskName:        g.invocation.Test.Name,
		TestArgs:        g.invocation.TestArgs,
	}
	cmd.Config(wrap(g.workerConfig))

	labels, err := g.inventoryLabels()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	trr, err := g.testRunnerRequest(ctx)
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	return request.Args{
		Cmd:                     *cmd,
		SchedulableLabels:       *labels,
		Dimensions:              g.params.GetFreeformAttributes().GetSwarmingDimensions(),
		ParentTaskID:            g.parentTaskID,
		Priority:                g.params.GetScheduling().GetPriority(),
		ProvisionableDimensions: provisionableDimensions,
		StatusTopic:             g.params.GetNotification().GetPubsubTopic(),
		SwarmingTags:            g.swarmingTags(cmd),
		TestRunnerRequest:       trr,
		Timeout:                 timeout,
	}, nil

}

func (g *Generator) isClientTest() (bool, error) {
	switch g.invocation.Test.ExecutionEnvironment {
	case build_api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT:
		return true, nil
	case build_api.AutotestTest_EXECUTION_ENVIRONMENT_SERVER:
		return false, nil
	default:
		return false, errors.Reason("unknown exec environment %s", g.invocation.Test.ExecutionEnvironment).Err()
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
	if g.params.GetSoftwareAttributes().GetBuildTarget() != nil {
		*inv.Board = g.params.SoftwareAttributes.BuildTarget.Name
	}
	if g.params.GetHardwareAttributes().GetModel() != "" {
		*inv.Model = g.params.HardwareAttributes.Model
	}

	if p := g.params.GetScheduling().GetPool(); p != nil {
		switch v := p.(type) {
		case *test_platform.Request_Params_Scheduling_ManagedPool_:
			pool, ok := poolMap[v.ManagedPool]
			if !ok {
				return nil, errors.Reason("unknown managed pool %s", v.ManagedPool.String()).Err()
			}
			inv.CriticalPools = append(inv.CriticalPools, pool)
		case *test_platform.Request_Params_Scheduling_UnmanagedPool:
			inv.SelfServePools = append(inv.SelfServePools, v.UnmanagedPool)
		case *test_platform.Request_Params_Scheduling_QuotaAccount:
			inv.CriticalPools = append(inv.CriticalPools, inventory.SchedulableLabels_DUT_POOL_QUOTA)
		default:
			panic(fmt.Sprintf("unhandled scheduling type %#v", p))
		}
	}
	return inv, nil
}

const (
	// These prefixes are interpreted by autotest's provisioning behavior;
	// they are defined in the autotest repo, at utils/labellib.py
	prefixChromeOS   = "cros-version"
	prefixFirmwareRO = "fwro-version"
	prefixFirmwareRW = "fwrw-version"
)

func (g *Generator) provisionableDimensions() ([]string, error) {
	deps := g.params.SoftwareDependencies
	builds, err := extractBuilds(deps)
	if err != nil {
		return nil, errors.Annotate(err, "get provisionable dimensions").Err()
	}

	var dims []string
	if b := builds.ChromeOS; b != "" {
		dims = append(dims, "provisionable-"+prefixChromeOS+":"+b)
	}
	if b := builds.FirmwareRO; b != "" {
		dims = append(dims, "provisionable-"+prefixFirmwareRO+":"+b)
	}
	if b := builds.FirmwareRW; b != "" {
		dims = append(dims, "provisionable-"+prefixFirmwareRW+":"+b)
	}
	return dims, nil
}

func (g *Generator) provisionableLabels() (map[string]string, error) {
	deps := g.params.SoftwareDependencies
	builds, err := extractBuilds(deps)
	if err != nil {
		return nil, errors.Annotate(err, "get provisionable labels").Err()
	}

	dims := make(map[string]string)
	if b := builds.ChromeOS; b != "" {
		dims[prefixChromeOS] = b
	}
	if b := builds.FirmwareRO; b != "" {
		dims[prefixFirmwareRO] = b
	}
	if b := builds.FirmwareRW; b != "" {
		dims[prefixFirmwareRW] = b
	}
	return dims, nil
}

func (g *Generator) timeout() (time.Duration, error) {
	if g.params.Time == nil {
		return 0, errors.Reason("get timeout: nil params.time").Err()
	}
	duration, err := ptypes.Duration(g.params.Time.MaximumDuration)
	if err != nil {
		return 0, errors.Annotate(err, "get timeout").Err()
	}
	return duration, nil
}

func (g *Generator) displayName(ctx context.Context, kv map[string]string) string {
	if g.invocation.DisplayName != "" {
		return g.invocation.DisplayName
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
	testName := g.invocation.GetTest().GetName()
	builds, err := extractBuilds(g.params.SoftwareDependencies)
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
	for k, v := range g.invocation.GetResultKeyvals() {
		if _, ok := kv[k]; !ok {
			kv[k] = v
		}
	}
}

func (g *Generator) baseKeyvals() map[string]string {
	keyvals := make(map[string]string)
	for k, v := range g.params.GetDecorations().GetAutotestKeyvals() {
		keyvals[k] = v
	}
	if g.parentTaskID != "" {
		// This keyval is inspected by some downstream results consumers such as
		// goldeneye and stainless.
		// TODO(akeshet): Consider whether parameter-specified parent_job_id
		// should be respected if it was specified.
		keyvals["parent_job_id"] = g.parentTaskID
	}
	// These build related keyvals are used by gs_offlaoder's CTS results
	// offload hook.
	for _, sd := range g.params.GetSoftwareDependencies() {
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

func (g *Generator) swarmingTags(cmd *worker.Command) []string {
	tags := []string{
		"luci_project:" + g.workerConfig.LuciProject,
		"log_location:" + cmd.LogDogAnnotationURL,
	}
	if qa := g.params.GetScheduling().GetQuotaAccount(); qa != "" {
		tags = append(tags, "qs_account:"+qa)
	}
	// TODO(akeshet): Consider whether to ban qs_account, luci_project, log_location,
	// and other "special tags" from being client-specified here.
	tags = append(tags, g.params.GetDecorations().GetTags()...)
	return tags
}

// builds describes the build names that were requested by a test_platform
// invocation.
type builds struct {
	ChromeOS   string
	FirmwareRW string
	FirmwareRO string
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
	pl, err := g.provisionableLabels()
	if err != nil {
		return nil, errors.Annotate(err, "create test runner request").Err()
	}
	kv := g.keyvals(ctx)
	return &skylab_test_runner.Request{
		Deadline: google.NewTimestamp(g.deadline),
		Prejob: &skylab_test_runner.Request_Prejob{
			ProvisionableLabels: pl,
		},
		Test: &skylab_test_runner.Request_Test{
			Harness: &skylab_test_runner.Request_Test_Autotest_{
				Autotest: &skylab_test_runner.Request_Test_Autotest{
					DisplayName:  g.displayName(ctx, kv),
					IsClientTest: isClient,
					Name:         g.invocation.Test.Name,
					Keyvals:      kv,
					TestArgs:     g.invocation.TestArgs,
				},
			},
		},
	}, nil
}
