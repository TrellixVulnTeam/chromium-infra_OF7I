// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inventory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/clients"
	"infra/appengine/crosskylabadmin/app/config"
	"infra/appengine/crosskylabadmin/app/frontend/internal/datastore/deploy"
	"infra/appengine/crosskylabadmin/app/frontend/internal/swarming"
	"infra/appengine/crosskylabadmin/app/frontend/internal/worker"
	"infra/appengine/crosskylabadmin/app/gitstore"
	"infra/cros/lab_inventory/utils"
	protos "infra/libs/fleet/protos"
	"infra/libs/skylab/inventory"

	"go.chromium.org/chromiumos/infra/proto/go/device"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	validateDeviceConfigsFunc = validateDeviceconfig
)

// DeployDut implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) DeployDut(ctx context.Context, req *fleet.DeployDutRequest) (resp *fleet.DeployDutResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err = req.Validate(); err != nil {
		return nil, err
	}

	allSpecs, err := parseManyDUTSpecs(req.GetNewSpecs())
	if err != nil {
		return nil, err
	}
	for _, specs := range allSpecs {
		if specs.GetHostname() == "" {
			return nil, status.Error(codes.InvalidArgument, "DUT hostname not set in new_specs")
		}
	}

	ic, err := is.newInventoryClient(ctx)
	if err != nil {
		return nil, err
	}

	if err = validateDeviceConfigsFunc(ctx, ic, allSpecs); err != nil {
		return nil, err
	}

	sc, err := is.newSwarmingClient(ctx, config.Get(ctx).Swarming.Host)
	if err != nil {
		return nil, err
	}

	attemptID, err := initializeDeployAttempt(ctx)
	if err != nil {
		return nil, err
	}

	actions := req.GetActions()
	options := req.GetOptions()

	ds := deployManyDUTs(ctx, ic, sc, attemptID, allSpecs, actions, options)
	updateDeployStatusIgnoringErrors(ctx, attemptID, ds)
	return &fleet.DeployDutResponse{DeploymentId: attemptID}, nil
}

// validateDeviceconfig validates if the device config is available in the inventoryV2
func validateDeviceconfig(ctx context.Context, ic inventoryClient, nds []*inventory.CommonDeviceSpecs) error {
	devCfgIds := make([]*device.ConfigId, 0, len(nds))
	for _, d := range nds {
		devConfigID := &device.ConfigId{
			PlatformId: &device.PlatformId{
				Value: d.GetLabels().GetBoard(),
			},
			ModelId: &device.ModelId{
				Value: d.GetLabels().GetModel(),
			},
		}
		devCfgIds = append(devCfgIds, devConfigID)
	}
	exists, err := ic.deviceConfigsExists(ctx, devCfgIds)
	if err != nil {
		return err
	}
	for i := range devCfgIds {
		if !exists[int32(i)] {
			return status.Errorf(codes.NotFound, "Device Config Not found for %s", devCfgIds[i].String())
		}
	}
	return nil
}

// RedeployDut implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) RedeployDut(ctx context.Context, req *fleet.RedeployDutRequest) (resp *fleet.RedeployDutResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err = req.Validate(); err != nil {
		return nil, err
	}

	oldSpecs, err := parseDUTSpecs(req.GetOldSpecs())
	if err != nil {
		return nil, err
	}
	newSpecs, err := parseAndEnsureDUTSpecs(req.GetNewSpecs())
	if err != nil {
		return nil, err
	}
	if oldSpecs.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "empty ID in old_specs")
	}
	if newSpecs.GetId() != oldSpecs.GetId() {
		return nil, status.Errorf(codes.InvalidArgument, "new_specs ID %s does not match old_specs ID %s",
			newSpecs.GetId(), oldSpecs.GetId())
	}

	ic, err := is.newInventoryClient(ctx)
	if err != nil {
		return nil, err
	}
	sc, err := is.newSwarmingClient(ctx, config.Get(ctx).Swarming.Host)
	if err != nil {
		return nil, err
	}

	attemptID, err := initializeDeployAttempt(ctx)
	if err != nil {
		return nil, err
	}
	ds := redeployDUT(ctx, ic, sc, attemptID, oldSpecs, newSpecs, req.GetActions(), req.GetOptions())
	updateDeployStatusIgnoringErrors(ctx, attemptID, ds)
	return &fleet.RedeployDutResponse{DeploymentId: attemptID}, nil
}

// GetDeploymentStatus implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) GetDeploymentStatus(ctx context.Context, req *fleet.GetDeploymentStatusRequest) (resp *fleet.GetDeploymentStatusResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err = req.Validate(); err != nil {
		return nil, err
	}

	sc, err := is.newSwarmingClient(ctx, config.Get(ctx).Swarming.Host)
	if err != nil {
		return nil, err
	}

	ds, err := deploy.GetStatus(ctx, req.DeploymentId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no deployment attempt with ID %s", req.DeploymentId)
	}
	if !ds.IsFinal {
		if err = refreshDeployStatus(ctx, sc, ds, req.DeploymentId); err != nil {
			return nil, err
		}
		if err := deploy.UpdateStatus(ctx, req.DeploymentId, ds); err != nil {
			return nil, err
		}
	}

	resp = &fleet.GetDeploymentStatusResponse{
		Status:    ds.Status,
		ChangeUrl: ds.ChangeURL,
		Message:   ds.Reason,
	}
	if len(ds.TaskIDs) > 0 {
		resp.TaskUrl = swarming.URLForTags(ctx, getDeployTags(req.DeploymentId))
	}
	return resp, nil
}

// DeleteDuts implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) DeleteDuts(ctx context.Context, req *fleet.DeleteDutsRequest) (resp *fleet.DeleteDutsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err = req.Validate(); err != nil {
		return nil, err
	}
	ic, err := is.newInventoryClient(ctx)
	if err != nil {
		return nil, err
	}
	changeURL, removedIDs, err := ic.deleteDUTsFromFleet(ctx, req.Hostnames)
	if err != nil {
		return nil, err
	}

	return &fleet.DeleteDutsResponse{
		ChangeUrl: changeURL,
		Ids:       removedIDs,
	}, nil
}

// initializeDeployAttempt initializes internal state for a deployment attempt.
//
// This function returns a new ID for this deployment attempt.
func initializeDeployAttempt(ctx context.Context) (string, error) {
	attemptID, err := deploy.PutStatus(ctx, &deploy.Status{
		Status: fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_FAILED,
		Reason: "unknown",
	})
	if err != nil {
		return "", errors.Annotate(err, "initialize deploy attempt").Err()
	}
	return attemptID, nil
}

// deploy many DUTs simultaneously.
func deployManyDUTs(ctx context.Context, ic inventoryClient, sc clients.SwarmingClient, attemptID string, nds []*inventory.CommonDeviceSpecs, a *fleet.DutDeploymentActions, o *fleet.DutDeploymentOptions) *deploy.Status {
	// TODO(gregorynisbet): consider policy for amalgamating many errors into a coherent description
	// of how deployManyDUTs failed.
	ds := &deploy.Status{Status: fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_IN_PROGRESS}
	for _, nd := range nds {
		id := uuid.New().String()
		nd.Id = &id
	}
	// Add DUTs to fleet first synchronously, don't proceed with scheduling tasks
	// unless we've succeeded in adding the DUTs to the inventory.
	url, newDevices, err := ic.addManyDUTsToFleet(ctx, nds, o.GetAssignServoPortIfMissing())
	ds.ChangeURL = url
	if err != nil {
		failDeployStatus(ctx, ds, fmt.Sprintf("failed to add DUT(s) to fleet: %s", err))
		return ds
	}

	// Update registration datastore
	updateAssets(ctx, ic, newDevices)

	if a.GetSkipDeployment() {
		ds.IsFinal = true
		ds.Status = fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_SUCCEEDED
		return ds
	}
	for _, nd := range newDevices {
		logging.Infof(ctx, "schedule deploy task for %s", nd.GetHostname())
		taskID, err := scheduleDUTPreparationTask(ctx, sc, nd.GetId(), attemptID, a)
		// We deliberately keep going after encountering an error. We try to
		// schedule a preparation for every DUT once regardless of whether an earlier
		// DUT fails.
		if err != nil {
			logging.Errorf(ctx, "failed to create deploy task: %s", err)
		}
		ds.TaskIDs = append(ds.TaskIDs, taskID)
	}
	return ds
}

func updateAssets(ctx context.Context, ic inventoryClient, newDevices []*inventory.CommonDeviceSpecs) {
	if len(newDevices) <= 0 {
		return
	}
	var existingAssetsIDs = make([]string, 0, len(newDevices))
	var existingAssets = make([]*protos.ChopsAsset, 0, len(newDevices))
	for _, newDevice := range newDevices {
		existingAssetsIDs = append(existingAssetsIDs, newDevice.GetId())
		existingAssets = append(existingAssets,
			&protos.ChopsAsset{
				Id:       newDevice.GetId(),
				Location: utils.GetLocation(newDevice.GetHostname()),
			})
	}
	assetResponse, err := ic.getAssetsFromRegistration(ctx, &api.AssetIDList{Id: existingAssetsIDs})
	if err == nil {
		for _, assetResult := range assetResponse.Passed {
			logging.Infof(ctx, "AssetId: %s , Old Location: %s", assetResult.GetAsset().GetId(), assetResult.GetAsset().GetLocation().String())
		}
		for _, assetResult := range assetResponse.Failed {
			logging.Infof(ctx, "failed to get asset from registration for %s : %s", assetResult.Asset.GetId(), assetResult.GetErrorMsg())
		}
	}
	assetResponse, err = ic.updateAssetsInRegistration(ctx, &api.AssetList{Asset: existingAssets})
	if err == nil {
		for _, assetResult := range assetResponse.Passed {
			logging.Infof(ctx, "AssetId: %s, New Location: %s", assetResult.GetAsset().GetId(), assetResult.GetAsset().GetLocation().String())
		}
		for _, assetResult := range assetResponse.Failed {
			logging.Infof(ctx, "failed to update in registration for %s : %s", assetResult.GetAsset().GetId(), assetResult.GetErrorMsg())
		}
	}
}

const (
	servoHostAttributeKey = "servo_host"
	servoPortAttributeKey = "servo_port"
)

// assignNewServoPort adds a valid servo port attribute to d
//
// This function returns an error if d does not have the servo host attribute.
// The assigned servo port value is unique among all the duts with the same
// servo host attribute.
func assignNewServoPort(duts []*inventory.DeviceUnderTest, d *inventory.CommonDeviceSpecs) error {
	if looksLikeLabstation(d.GetHostname()) {
		return nil
	}
	servoHost, found := getAttributeByKey(d, servoHostAttributeKey)
	if !found {
		return errors.Reason("no servo_host attribute in specs").Err()
	}

	used := usedServoPorts(duts, servoHost)
	p, err := findFreePort(used)
	if err != nil {
		return err
	}
	k := servoPortAttributeKey
	v := strconv.Itoa(p)
	d.Attributes = append(d.Attributes, &inventory.KeyValue{Key: &k, Value: &v})
	return nil
}

// usedServoPorts finds the servo ports used by duts with the given servo host
// attribute.
func usedServoPorts(duts []*inventory.DeviceUnderTest, servoHost string) []int {
	used := []int{}
	for _, d := range duts {
		c := d.GetCommon()
		if sh, found := getAttributeByKey(c, servoHostAttributeKey); found && sh == servoHost {
			if p, found := getAttributeByKey(c, servoPortAttributeKey); found {
				ip, err := strconv.ParseInt(p, 10, 32)
				if err != nil {
					// This is not the right place to enforce servo_port correctness for other DUTs.
					// All we care about is that the port we pick will not conflict with
					// this corrupted entry.
					continue
				}
				used = append(used, int(ip))
			}
		}
	}
	return used
}

// findFreePort finds a valid port that is not in use.
//
// This function returns an error if no free port is found.
// This function modifies the slice of ports provided.
func findFreePort(used []int) (int, error) {
	sort.Sort(sort.Reverse(sort.IntSlice(used)))
	// This range is consistent with the range of ports generated by servod:
	// https://chromium.googlesource.com/chromiumos/third_party/hdctools/+/cf5f8027b9d3015db75df4853e37ea7a2f1ac538/servo/servod.py#36
	for p := 9999; p > 9900; p-- {
		if len(used) == 0 || p != used[0] {
			return p, nil
		}
		used = used[1:]
	}
	return -1, errors.Reason("no free valid ports").Err()
}

// addDUTToStore adds a new DUT with the given specs to the store.
//
// This function returns the ID for the added DUT.
func addDUTToStore(s *gitstore.InventoryStore, nd *inventory.CommonDeviceSpecs) string {
	// TODO(crbug/912977) DUTs under deployment are not marked specially in the
	// inventory yet. This causes two problems:
	// - Another admin task (say repair) may get scheduled on the new bot
	//   before the deploy task we create.
	// - If the deploy task fails, the DUT will still enter the fleet, but may
	//   not be ready for use.
	s.Lab.Duts = append(s.Lab.Duts, &inventory.DeviceUnderTest{
		Common: nd,
	})
	return *nd.Id
}

// scheduleDUTPreparationTask schedules a Skylab DUT preparation task.
func scheduleDUTPreparationTask(ctx context.Context, sc clients.SwarmingClient, dutID string, attemptID string, a *fleet.DutDeploymentActions) (string, error) {
	if a.GetSkipDeployment() {
		return "", errors.New("no DUT preparation task should be scheduled if deployment is skipped")
	}
	taskCfg := config.Get(ctx).GetEndpoint().GetDeployDut()
	tags := swarming.AddCommonTags(ctx, fmt.Sprintf("deploy_task:%s", dutID))
	for _, t := range getDeployTags(attemptID) {
		tags = append(tags, t)
	}
	at := worker.DeployTaskWithActions(ctx, deployActionArgs(a))
	tags = append(tags, at.Tags...)
	return sc.CreateTask(ctx, at.Name, swarming.SetCommonTaskArgs(ctx, &clients.SwarmingCreateTaskArgs{
		Cmd:                  at.Cmd,
		DutID:                dutID,
		ExecutionTimeoutSecs: taskCfg.GetTaskExecutionTimeout().GetSeconds(),
		ExpirationSecs:       taskCfg.GetTaskExpirationTimeout().GetSeconds(),
		Priority:             taskCfg.GetTaskPriority(),
		Tags:                 tags,
	}))
}

// redeployDUT kicks off a redeployment of an existing DUT.
//
// Errors are communicated via returned deploy.Status
func redeployDUT(ctx context.Context, ic inventoryClient, sc clients.SwarmingClient, attemptID string, oldSpecs, newSpecs *inventory.CommonDeviceSpecs, a *fleet.DutDeploymentActions, o *fleet.DutDeploymentOptions) *deploy.Status {
	var err error
	ds := &deploy.Status{Status: fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_IN_PROGRESS}

	if !proto.Equal(oldSpecs, newSpecs) {
		ds.ChangeURL, err = ic.updateDUTSpecs(ctx, oldSpecs, newSpecs, o.GetAssignServoPortIfMissing())
		if err != nil {
			failDeployStatus(ctx, ds, fmt.Sprintf("failed to update DUT specs: %s", err))
			return ds
		}
		// Update registration datastore
		newSpecsArr := []*inventory.CommonDeviceSpecs{newSpecs}
		updateAssets(ctx, ic, newSpecsArr)
	}

	taskID, err := scheduleDUTPreparationTask(ctx, sc, oldSpecs.GetId(), attemptID, a)
	ds.TaskIDs = append(ds.TaskIDs, taskID)
	if err != nil {
		failDeployStatus(ctx, ds, fmt.Sprintf("failed to create deploy task: %s", err))
		return ds
	}
	return ds
}

// refreshDeployStatus refreshes the status of given deployment attempt from
// Swarming.
func refreshDeployStatus(ctx context.Context, sc clients.SwarmingClient, ds *deploy.Status, attemptID string) error {
	if len(ds.TaskIDs) < 1 {
		failDeployStatus(ctx, ds, "missing deploy task IDs in deploy request entry")
		return nil
	}

	tags := getDeployTags(attemptID)
	results, err := sc.ListRecentTasks(ctx, tags, "", len(ds.TaskIDs))
	if err != nil {
		return errors.Annotate(err, "refresh deploy status").Err()
	}

	if len(results) != len(ds.TaskIDs) {
		logging.Warningf(ctx, "%d tasks expected, only found %d tasks", len(ds.TaskIDs), len(results))
	}

	isFinal := true
	status := fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_SUCCEEDED
	for _, tr := range results {
		switch tr.State {
		case "COMPLETED":
			if tr.Failure || tr.InternalFailure {
				status = fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_FAILED
			}
		case "PENDING", "RUNNING":
			isFinal = false
		default:
			status = fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_FAILED
			logging.Warningf(ctx, "unhandled deploy task state: %s", tr.State)
		}
	}
	ds.IsFinal = isFinal
	if !isFinal {
		ds.Status = fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_IN_PROGRESS
	} else {
		ds.Status = status
	}
	return nil
}

// failDeployStatus updates ds to correspond to a failed deploy with the given
// reason.
func failDeployStatus(ctx context.Context, ds *deploy.Status, reason string) {
	ds.IsFinal = true
	ds.Status = fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_FAILED
	ds.Reason = reason
	logging.Errorf(ctx, "Failed deploy: %#v", ds)
}

func updateDeployStatusIgnoringErrors(ctx context.Context, attemptID string, ds *deploy.Status) {
	if err := deploy.UpdateStatus(ctx, attemptID, ds); err != nil {
		logging.Errorf(ctx, "Failed to update status for deploy attempt %s to %v", attemptID, ds)
	}
}

// removeDUTWithHostnames deletes duts with the given hostnames.
//
// The function returns the deleted duts.
// If multiple DUTs have the same hostname, that is in hostnames, they are all deleted.
func removeDUTWithHostnames(s *gitstore.InventoryStore, hostnames []string) []*inventory.DeviceUnderTest {
	duts := s.Lab.Duts
	toRemove := stringset.NewFromSlice(hostnames...)
	removedDuts := make([]*inventory.DeviceUnderTest, 0, len(hostnames))
	for i := 0; i < len(duts); {
		d := duts[i]
		h := d.GetCommon().GetHostname()
		if !toRemove.Has(h) {
			i++
			continue
		}
		removedDuts = append(removedDuts, d)
		duts = deleteAtIndex(duts, i)
	}
	s.Lab.Duts = duts
	return removedDuts
}

func deleteAtIndex(duts []*inventory.DeviceUnderTest, i int) []*inventory.DeviceUnderTest {
	copy(duts[i:], duts[i+1:])
	duts[len(duts)-1] = nil
	return duts[:len(duts)-1]
}

func parseDUTSpecs(specs []byte) (*inventory.CommonDeviceSpecs, error) {
	var parsed inventory.CommonDeviceSpecs
	if err := proto.Unmarshal(specs, &parsed); err != nil {
		return nil, errors.Annotate(err, "parse DUT specs").Tag(grpcutil.InvalidArgumentTag).Err()
	}
	return &parsed, nil
}

func parseAndEnsureDUTSpecs(specs []byte) (*inventory.CommonDeviceSpecs, error) {
	parsed, err := parseDUTSpecs(specs)
	if err != nil {
		return nil, err
	}
	// If the host is a labstation, ensure the os_type has correct value.
	hostname := parsed.GetHostname()
	if looksLikeLabstation(hostname) && parsed.GetLabels().GetOsType() != inventory.SchedulableLabels_OS_TYPE_LABSTATION {
		return nil, errors.Reason("The os_type of %s must be OS_TYPE_LABSTATION (please ensure to use `skylab add-labstation` for labstations deployment)", hostname).Err()
	}
	return parsed, nil
}

func parseManyDUTSpecs(specsArr [][]byte) ([]*inventory.CommonDeviceSpecs, error) {
	var out []*inventory.CommonDeviceSpecs
	out = make([]*inventory.CommonDeviceSpecs, 0, len(specsArr))
	for _, item := range specsArr {
		if parsed, err := parseAndEnsureDUTSpecs(item); err == nil {
			out = append(out, parsed)
		} else {
			return nil, err
		}
	}
	return out, nil
}

func hasServoPortAttribute(d *inventory.CommonDeviceSpecs) bool {
	_, found := getAttributeByKey(d, servoPortAttributeKey)
	return found
}

// getAttributeByKey by returns the value for the attribute with the given key,
// and whether the key was found.
func getAttributeByKey(d *inventory.CommonDeviceSpecs, key string) (string, bool) {
	for _, a := range d.Attributes {
		if *a.Key == key {
			return *a.Value, true
		}
	}
	return "", false
}

func deployActionArgs(a *fleet.DutDeploymentActions) string {
	s := make([]string, 0, 5)
	if a.GetStageImageToUsb() {
		s = append(s, "stage-usb")
	}
	if a.GetInstallTestImage() {
		s = append(s, "install-test-image")
		s = append(s, "update-label")
	}
	if a.GetInstallFirmware() {
		s = append(s, "install-firmware")
		s = append(s, "verify-recovery-mode")
	}
	if a.GetSetupLabstation() {
		s = append(s, "setup-labstation")
		s = append(s, "update-label")
	}
	if a.GetRunPreDeployVerification() {
		s = append(s, "run-pre-deploy-verification")
	}
	return strings.Join(s, ",")
}

func getDeployTags(attempID string) []string {
	return []string{
		fmt.Sprintf("deployAttemptID:%s", attempID),
		"task:Deploy",
	}
}
