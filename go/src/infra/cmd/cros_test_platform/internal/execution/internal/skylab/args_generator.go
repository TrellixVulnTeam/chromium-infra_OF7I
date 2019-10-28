// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/internal/common"
	"infra/libs/skylab/inventory"
	"infra/libs/skylab/inventory/autotest/labels"
	"infra/libs/skylab/request"
	"infra/libs/skylab/worker"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	build_api "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

type argsGenerator struct {
	invocation   *steps.EnumerationResponse_AutotestInvocation
	params       *test_platform.Request_Params
	workerConfig *config.Config_SkylabWorker
	parentTaskID string
}

// CheckConsistency checks the internal consistency of the various inputs to the
// argument generation logic.
func (a *argsGenerator) CheckConsistency() error {
	el := a.enumerationInventoryLabels()

	rb := a.params.GetSoftwareAttributes().GetBuildTarget().GetName()
	eb := el.GetBoard()
	if nonEmptyAndDifferent(rb, eb) {
		return errors.Reason("incompatible board dependency: request (%s) vs. enumeration (%s)", rb, eb).Err()
	}

	rm := a.params.GetHardwareAttributes().GetModel()
	em := el.GetModel()
	if nonEmptyAndDifferent(rm, em) {
		return errors.Reason("incompatible model dependency: request (%s) vs. enumeration (%s)", rm, em).Err()
	}

	ud := a.getUnsupportedDependencies()
	if len(ud) > 0 {
		return errors.Reason("unsupported request dependencies: %s", strings.Join(ud, ", ")).Err()
	}

	return nil
}

func nonEmptyAndDifferent(a, b string) bool {
	return a != "" && b != "" && a != b
}

func (a *argsGenerator) enumerationInventoryLabels() *inventory.SchedulableLabels {
	deps := a.invocation.Test.Dependencies
	flatDims := make([]string, len(deps))
	for i, dep := range deps {
		flatDims[i] = dep.Label
	}
	return labels.Revert(flatDims)
}

func (a *argsGenerator) getUnsupportedDependencies() []string {
	el := a.enumerationInventoryLabels()
	unsupported := stringset.New(len(a.invocation.Test.Dependencies))
	for _, dep := range a.invocation.Test.Dependencies {
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
func (a *argsGenerator) GenerateArgs(ctx context.Context) (request.Args, error) {
	isClient, err := a.isClientTest()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	provisionableDimensions, err := a.provisionableDimensions()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	timeout, err := a.timeout()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	kv := a.baseKeyvals()
	a.updateWithInvocationKeyvals(kv)
	a.addKeyvalsForDisplayName(ctx, kv)

	cmd := &worker.Command{
		TaskName:        a.invocation.Test.Name,
		ClientTest:      isClient,
		OutputToIsolate: true,
		TestArgs:        a.invocation.TestArgs,
		Keyvals:         kv,
	}
	cmd.Config(wrap(a.workerConfig))

	labels, err := a.inventoryLabels()
	if err != nil {
		return request.Args{}, errors.Annotate(err, "create request args").Err()
	}

	return request.Args{
		Cmd:                     *cmd,
		SchedulableLabels:       *labels,
		Dimensions:              a.params.GetFreeformAttributes().GetSwarmingDimensions(),
		ParentTaskID:            a.parentTaskID,
		Priority:                a.params.GetScheduling().GetPriority(),
		ProvisionableDimensions: provisionableDimensions,
		StatusTopic:             a.params.GetNotification().GetPubsubTopic(),
		SwarmingTags:            a.swarmingTags(cmd),
		Timeout:                 timeout,
	}, nil

}

func (a *argsGenerator) isClientTest() (bool, error) {
	switch a.invocation.Test.ExecutionEnvironment {
	case build_api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT:
		return true, nil
	case build_api.AutotestTest_EXECUTION_ENVIRONMENT_SERVER:
		return false, nil
	default:
		return false, errors.Reason("unknown exec environment %s", a.invocation.Test.ExecutionEnvironment).Err()
	}
}

func (a *argsGenerator) inventoryLabels() (*inventory.SchedulableLabels, error) {
	inv := a.enumerationInventoryLabels()
	if a.params.GetSoftwareAttributes().GetBuildTarget() != nil {
		*inv.Board = a.params.SoftwareAttributes.BuildTarget.Name
	}
	if a.params.GetHardwareAttributes().GetModel() != "" {
		*inv.Model = a.params.HardwareAttributes.Model
	}

	if p := a.params.GetScheduling().GetPool(); p != nil {
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

func (a *argsGenerator) provisionableDimensions() ([]string, error) {
	deps := a.params.SoftwareDependencies
	builds, err := common.ExtractBuilds(deps)
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

func (a *argsGenerator) timeout() (time.Duration, error) {
	if a.params.Time == nil {
		return 0, errors.Reason("get timeout: nil params.time").Err()
	}
	duration, err := ptypes.Duration(a.params.Time.MaximumDuration)
	if err != nil {
		return 0, errors.Annotate(err, "get timeout").Err()
	}
	return duration, nil
}

func (a *argsGenerator) addKeyvalsForDisplayName(ctx context.Context, kv map[string]string) {
	const displayNameKey = "label"

	if a.invocation.DisplayName != "" {
		kv[displayNameKey] = a.invocation.DisplayName
		return
	}
	kv[displayNameKey] = a.constructDisplayNameFromRequestParams(ctx, kv)
}

const (
	suiteKey         = "suite"
	defaultSuiteName = "cros_test_platform"
)

// This is a hack to satisfy tko/parse's insistence on parsing the display name
// (aka "label") keyval to obtain semantic information about the request.
// TODO(crbug.com/1003490): Drop this once result reporting is updated to stop
// parsing the "label" keyval.
func (a *argsGenerator) constructDisplayNameFromRequestParams(ctx context.Context, kv map[string]string) string {
	testName := a.invocation.GetTest().GetName()
	builds, err := common.ExtractBuilds(a.params.SoftwareDependencies)
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

func (a *argsGenerator) updateWithInvocationKeyvals(kv map[string]string) {
	for k, v := range a.invocation.GetResultKeyvals() {
		if _, ok := kv[k]; !ok {
			kv[k] = v
		}
	}
}

func (a *argsGenerator) baseKeyvals() map[string]string {
	keyvals := make(map[string]string)
	for k, v := range a.params.GetDecorations().GetAutotestKeyvals() {
		keyvals[k] = v
	}
	if a.parentTaskID != "" {
		// This keyval is inspected by some downstream results consumers such as
		// goldeneye and stainless.
		// TODO(akeshet): Consider whether parameter-specified parent_job_id
		// should be respected if it was specified.
		keyvals["parent_job_id"] = a.parentTaskID
	}
	// These build related keyvals are used by gs_offlaoder's CTS results
	// offload hook.
	for _, sd := range a.params.GetSoftwareDependencies() {
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

func (a *argsGenerator) swarmingTags(cmd *worker.Command) []string {
	tags := []string{
		"luci_project:" + a.workerConfig.LuciProject,
		"log_location:" + cmd.LogDogAnnotationURL,
	}
	if qa := a.params.GetScheduling().GetQuotaAccount(); qa != "" {
		tags = append(tags, "qs_account:"+qa)
	}
	// TODO(akeshet): Consider whether to ban qs_account, luci_project, log_location,
	// and other "special tags" from being client-specified here.
	tags = append(tags, a.params.GetDecorations().GetTags()...)
	return tags
}
