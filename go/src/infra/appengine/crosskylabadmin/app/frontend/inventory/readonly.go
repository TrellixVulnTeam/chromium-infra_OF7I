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

// Package inventory implements the fleet.Inventory service end-points of
// corsskylabadmin.
package inventory

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"
	"go.chromium.org/luci/grpc/grpcutil"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/frontend/internal/datastore/dronecfg"
	"infra/appengine/crosskylabadmin/app/frontend/internal/datastore/freeduts"
	dsinventory "infra/appengine/crosskylabadmin/app/frontend/internal/datastore/inventory"
	dssv "infra/appengine/crosskylabadmin/app/frontend/internal/datastore/stableversion"
	"infra/appengine/crosskylabadmin/app/frontend/internal/gitstore"
	"infra/appengine/crosskylabadmin/app/frontend/internal/metrics/utilization"
	"infra/libs/skylab/inventory"
)

const beagleboneServo = "beaglebone_servo"

// ListServers implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) ListServers(ctx context.Context, req *fleet.ListServersRequest) (resp *fleet.ListServersResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	return nil, status.Error(codes.Unimplemented, "ListServers not yet implemented")
}

// GetDutInfo implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) GetDutInfo(ctx context.Context, req *fleet.GetDutInfoRequest) (resp *fleet.GetDutInfoResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	if err = req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var dut *dsinventory.DeviceUnderTest
	if req.Id != "" {
		dut, err = dsinventory.GetSerializedDUTByID(ctx, req.Id)
	} else {
		dut, err = dsinventory.GetSerializedDUTByHostname(ctx, req.Hostname)
	}

	if err != nil {
		if datastore.IsErrNoSuchEntity(err) {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return &fleet.GetDutInfoResponse{
		Spec:    dut.Data,
		Updated: google.NewTimestamp(dut.Updated),
	}, nil
}

// GetDroneConfig implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) GetDroneConfig(ctx context.Context, req *fleet.GetDroneConfigRequest) (resp *fleet.GetDroneConfigResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	e, err := dronecfg.Get(ctx, req.Hostname)
	if err != nil {
		if datastore.IsErrNoSuchEntity(err) {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return nil, err
	}
	resp = &fleet.GetDroneConfigResponse{}
	for _, d := range e.DUTs {
		resp.Duts = append(resp.Duts, &fleet.GetDroneConfigResponse_Dut{
			Id:       d.ID,
			Hostname: d.Hostname,
		})
	}
	return resp, nil
}

// ListRemovedDuts implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) ListRemovedDuts(ctx context.Context, req *fleet.ListRemovedDutsRequest) (resp *fleet.ListRemovedDutsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	duts, err := freeduts.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	resp = &fleet.ListRemovedDutsResponse{}
	for _, d := range duts {
		t, err := ptypes.TimestampProto(d.ExpireTime)
		if err != nil {
			return nil, err
		}
		resp.Duts = append(resp.Duts, &fleet.ListRemovedDutsResponse_Dut{
			Id:         d.ID,
			Hostname:   d.Hostname,
			Bug:        d.Bug,
			Comment:    d.Comment,
			ExpireTime: t,
			Model:      d.Model,
		})
	}
	return resp, nil
}

// GetStableVersion implements the method from fleet.InventoryServer interface
func (is *ServerImpl) GetStableVersion(ctx context.Context, req *fleet.GetStableVersionRequest) (resp *fleet.GetStableVersionResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	return getStableVersionImpl(ctx, req.BuildTarget, req.Model, req.Hostname)
}

// ReportInventory reports metrics of duts in inventory.
func (is *ServerImpl) ReportInventory(ctx context.Context, req *fleet.ReportInventoryRequest) (resp *fleet.ReportInventoryResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	store, err := is.newStore(ctx)
	if err != nil {
		return nil, err
	}
	if err := store.Refresh(ctx); err != nil {
		return nil, err
	}

	duts, err := GetDutsByEnvironment(ctx, store)
	if err != nil {
		return nil, err
	}
	utilization.ReportInventoryMetrics(ctx, duts)
	utilization.ReportServerMetrics(ctx, store.Infrastructure.GetServers())
	return &fleet.ReportInventoryResponse{}, nil
}

// UpdateCachedInventory implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) UpdateCachedInventory(ctx context.Context, req *fleet.UpdateCachedInventoryRequest) (resp *fleet.UpdateCachedInventoryResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	store, err := is.newStore(ctx)
	if err != nil {
		return nil, err
	}
	if err := store.Refresh(ctx); err != nil {
		return nil, err
	}
	duts := dutsInCurrentEnvironment(ctx, store.Lab.GetDuts())
	if err := dsinventory.UpdateDUTs(ctx, duts); err != nil {
		return nil, err
	}
	es := makeDroneConfigs(ctx, store.Infrastructure, store.Lab)
	if err := dronecfg.Update(ctx, es); err != nil {
		return nil, err
	}
	if err := updateFreeDUTs(ctx, store); err != nil {
		return nil, err
	}
	return &fleet.UpdateCachedInventoryResponse{}, nil
}

func dutsInCurrentEnvironment(ctx context.Context, duts []*inventory.DeviceUnderTest) []*inventory.DeviceUnderTest {
	// TODO(crbug.com/947322): Disable this temporarily until it
	// can be implemented properly.  This updates the cache of
	// DUTs which can only be queried by hostname or ID, so it is
	// not problematic to also cache DUTs in the wrong environment
	// (prod vs dev).
	return duts
}

func makeDroneConfigs(ctx context.Context, inf *inventory.Infrastructure, lab *inventory.Lab) []dronecfg.Entity {
	dutHostnames := makeDUTHostnameMap(lab.GetDuts())
	var entities []dronecfg.Entity
	for _, s := range inf.GetServers() {
		if !isDrone(s) {
			continue
		}
		e := dronecfg.Entity{
			Hostname: s.GetHostname(),
		}
		for _, d := range s.GetDutUids() {
			h, ok := dutHostnames[d]
			if !ok {
				logging.Infof(ctx, "DUT ID %s doesn't match any hostname", d)
				continue
			}
			e.DUTs = append(e.DUTs, dronecfg.DUT{
				ID:       d,
				Hostname: h,
			})
		}
		entities = append(entities, e)
	}
	return entities
}

// makeDUTHostnameMap makes a mapping from DUT IDs to DUT hostnames.
func makeDUTHostnameMap(duts []*inventory.DeviceUnderTest) map[string]string {
	m := make(map[string]string)
	for _, d := range duts {
		c := d.GetCommon()
		m[c.GetId()] = c.GetHostname()
	}
	return m
}

func isDrone(s *inventory.Server) bool {
	for _, r := range s.GetRoles() {
		if r == inventory.Server_ROLE_SKYLAB_DRONE {
			return true
		}
	}
	return false
}

func updateFreeDUTs(ctx context.Context, s *gitstore.InventoryStore) error {
	ic := newGlobalInvCache(ctx, s)
	var free []freeduts.DUT
	for dutID, d := range ic.idToDUT {
		if _, ok := ic.droneForDUT[dutID]; ok {
			continue
		}
		free = append(free, freeDUTInfo(d))
	}
	stale, err := getStaleFreeDUTs(ctx, free)
	if err != nil {
		return errors.Annotate(err, "update free duts").Err()
	}
	if err := freeduts.Remove(ctx, stale); err != nil {
		return errors.Annotate(err, "update free duts").Err()
	}
	if err := freeduts.Add(ctx, free); err != nil {
		return errors.Annotate(err, "update free duts").Err()
	}
	return nil
}

// getStaleFreeDUTs returns the free DUTs in datastore that are no longer
// free, given the currently free DUTs passed as an argument.
func getStaleFreeDUTs(ctx context.Context, free []freeduts.DUT) ([]freeduts.DUT, error) {
	freeMap := make(map[string]bool, len(free))
	for _, d := range free {
		freeMap[d.ID] = true
	}
	all, err := freeduts.GetAll(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "get stale free duts").Err()
	}
	stale := make([]freeduts.DUT, 0, len(all))
	for _, d := range all {
		if _, ok := freeMap[d.ID]; !ok {
			stale = append(stale, d)
		}
	}
	return stale, nil
}

// freeDUTInfo returns the free DUT info to store for a DUT.
func freeDUTInfo(d *inventory.DeviceUnderTest) freeduts.DUT {
	c := d.GetCommon()
	rr := d.GetRemovalReason()
	var t time.Time
	if ts := rr.GetExpireTime(); ts != nil {
		t = time.Unix(ts.GetSeconds(), int64(ts.GetNanos())).UTC()
	}
	return freeduts.DUT{
		ID:         c.GetId(),
		Hostname:   c.GetHostname(),
		Bug:        rr.GetBug(),
		Comment:    rr.GetComment(),
		ExpireTime: t,
		Model:      c.GetLabels().GetModel(),
	}
}

// getStableVersionImpl returns all the stable versions associated with a given buildTarget and model
// NOTE: hostname is explicitly allowed to be "". If hostname is "", then no hostname was provided in the GetStableVersion RPC call
func getStableVersionImpl(ctx context.Context, buildTarget string, model string, hostname string) (*fleet.GetStableVersionResponse, error) {
	logging.Infof(ctx, "getting stable version for buildTarget: %s and model: %s", buildTarget, model)

	if hostname == "" {
		logging.Infof(ctx, "hostname not provided, using buildTarget (%s) and model (%s)", buildTarget, model)
		return getStableVersionImplNoHostname(ctx, buildTarget, model)
	}

	logging.Infof(ctx, "hostname (%s) provided, ignoring user-provided buildTarget (%s) and model (%s)", hostname, buildTarget, model)
	return getStableVersionImplWithHostname(ctx, hostname)
}

// getStableVersionImplNoHostname returns stableversion information given a buildTarget and model
// TODO(gregorynisbet): Consider under what circumstances an error leaving this function
// should be considered transient or non-transient.
// If the dut in question is a beaglebone servo, then failing to get the firmware version
// is non-fatal.
func getStableVersionImplNoHostname(ctx context.Context, buildTarget string, model string) (*fleet.GetStableVersionResponse, error) {
	logging.Infof(ctx, "getting stable version for buildTarget: (%s) and model: (%s)", buildTarget, model)
	var err error
	merr := errors.NewMultiError()
	out := &fleet.GetStableVersionResponse{}

	out.CrosVersion, err = dssv.GetCrosStableVersion(ctx, buildTarget)
	if err != nil {
		merr = append(merr, err)
	}
	out.FaftVersion, err = dssv.GetFaftStableVersion(ctx, buildTarget, model)
	if err != nil {
		logging.Infof(ctx, "faft stable version does not exist: (%#v)", err)
	}
	// successful early exit if we have a beaglebone servo
	if buildTarget == beagleboneServo || model == beagleboneServo {
		return out, nil
	}
	out.FirmwareVersion, err = dssv.GetFirmwareStableVersion(ctx, buildTarget, model)
	if err != nil {
		logging.Errorf(ctx, "getStableVersionImplNoHostname: failed to get firmware version (%s) (%s)", buildTarget, model)
		merr = append(merr, err)
	}
	if len(merr) != 0 {
		// TODO(gregorynisbet): Consider a different error handling strategy.
		// Wrap the error so it's non-transient.
		logging.Infof(ctx, "getStableVersionImplNoHostname: errors (%#v)", merr)
		return nil, fmt.Errorf("getStableVersionImplNoHostname: errors (%s)", merr)
	}
	return out, nil
}

// getStableVersionImplWithHostname return stable version information given just a hostname
// TODO(gregorynisbet): Consider under what circumstances an error leaving this function
// should be considered transient or non-transient.
func getStableVersionImplWithHostname(ctx context.Context, hostname string) (*fleet.GetStableVersionResponse, error) {
	logging.Infof(ctx, "getting stable version for given hostname (%s)", hostname)
	var err error

	// If the DUT in question is a labstation or a servo (i.e. is a servo host), then it does not have
	// its own servo host.
	if looksLikeServo(hostname) {
		logging.Infof(ctx, "beaglebone servo provided")
		return getStableVersionImplNoHostname(ctx, beagleboneServo, "")
	}
	if looksLikeLabstation(hostname) {
		logging.Infof(ctx, "concluded that hostname (%s) is a servo host", hostname)
	}

	dut, err := getDUT(ctx, hostname)
	if err != nil {
		logging.Infof(ctx, "failed to get DUT: (%#v)", err)
		// TODO(gregorynisbet): Consider a different error handling strategy.
		// Wrap the error so it's non-transient.
		return nil, fmt.Errorf("failed to get DUT (%s)", err)
	}

	buildTarget := dut.GetCommon().GetLabels().GetBoard()
	model := dut.GetCommon().GetLabels().GetModel()

	out, err := getStableVersionImplNoHostname(ctx, buildTarget, model)
	if err != nil {
		// TODO(gregorynisbet): Consider a different error handling strategy.
		// Wrap the error so it's non-transient.
		logging.Infof(ctx, "failed to get stable version info: (%#v)", err)
		return nil, fmt.Errorf("failed to get stable version info (%s)", err)
	}

	if looksLikeLabstation(hostname) {
		return out, nil
	}
	servoHostHostname, err := getServoHostHostname(dut)
	if err != nil {
		// TODO(gregorynisbet): Consider a different error handling strategy.
		// Wrap the error so it's non-transient.
		logging.Infof(ctx, "getting hostname of servohost (%#v)", err)
		return nil, fmt.Errorf("getting hostname of servohost (%s)", err)
	}
	servoStableVersion, err := getCrosVersionFromServoHost(ctx, servoHostHostname)
	if err != nil {
		// TODO(gregorynisbet): Consider a different error handling strategy.
		// Wrap the error so it's non-transient.
		logging.Infof(ctx, "getting cros version from servo host (%#v)", err)
		return nil, fmt.Errorf("getting cros version from servo host (%s)", err)
	}
	out.ServoCrosVersion = servoStableVersion

	return out, nil
}

// getServoHostHostname gets the servo host hostname associated with a dut
// for instance, a labstation is a servo host.
func getServoHostHostname(dut *inventory.DeviceUnderTest) (string, error) {
	attrs := dut.GetCommon().GetAttributes()
	if len(attrs) == 0 {
		return "", fmt.Errorf("attributes for dut with hostname (%s) is unexpectedly empty", dut.GetCommon().GetHostname())
	}
	for _, item := range attrs {
		key := item.GetKey()
		value := item.GetValue()
		if key == "servo_host" {
			if value == "" {
				return "", fmt.Errorf("\"servo_host\" attribute unexpectedly has value \"\" for hostname (%s)", dut.GetCommon().GetHostname())
			}
			return value, nil
		}
	}
	return "", fmt.Errorf("no \"servo_host\" attribute for hostname (%s)", dut.GetCommon().GetHostname())
}

// getDUT returns the DUT associated with a particular hostname from datastore
func getDUT(ctx context.Context, hostname string) (*inventory.DeviceUnderTest, error) {
	resp, err := dsinventory.GetSerializedDUTByHostname(ctx, hostname)
	if err != nil {
		msg := fmt.Sprintf("getting serialized DUT by hostname for (%s)", hostname)
		return nil, errors.Annotate(err, msg).Err()
	}
	dut := &inventory.DeviceUnderTest{}
	if err := proto.Unmarshal(resp.Data, dut); err != nil {
		msg := fmt.Sprintf("unserializing DUT for hostname (%s)", hostname)
		return nil, errors.Annotate(err, msg).Err()
	}
	return dut, nil
}

// This is a heuristic to check if something is a labstation and might be wrong.
func looksLikeLabstation(hostname string) bool {
	return strings.Contains(hostname, "labstation")
}

// This is a heuristic to check if something is a servo and might be wrong.
func looksLikeServo(hostname string) bool {
	return strings.Contains(hostname, "servo")
}

// getCrosVersionFromServoHost returns the cros version associated with a particular servo host
// hostname : hostname of the servo host (e.g. labstation)
// NOTE: If hostname is "", this indicates the absence of a relevant servo host.
// This can happen if the DUT in question is already a labstation, for instance.
func getCrosVersionFromServoHost(ctx context.Context, hostname string) (string, error) {
	if hostname == "" {
		logging.Infof(ctx, "getCrosVersionFromServoHost: skipping empty hostname \"\"")
		return "", nil
	}
	if looksLikeLabstation(hostname) {
		logging.Infof(ctx, "getCrosVersionFromServoHost: identified labstation servohost hostname (%s)", hostname)
		dut, err := getDUT(ctx, hostname)
		if err != nil {
			return "", errors.Annotate(err, "get labstation dut info").Err()
		}
		buildTarget := dut.GetCommon().GetLabels().GetBoard()
		if buildTarget == "" {
			return "", fmt.Errorf("no buildTarget for hostname (%s)", hostname)
		}
		out, err := dssv.GetCrosStableVersion(ctx, buildTarget)
		if err != nil {
			return "", errors.Annotate(err, "getting labstation stable version").Err()
		}
		return out, nil
	}
	if looksLikeServo(hostname) {
		logging.Infof(ctx, "getCrosVersionFromServoHost: identified beaglebone servohost hostname (%s)", hostname)
		out, err := dssv.GetCrosStableVersion(ctx, beagleboneServo)
		if err != nil {
			return "", errors.Annotate(err, "getting beaglebone servo stable version").Err()
		}
		return out, nil
	}
	logging.Infof(ctx, "getCrosVersionFromServoHost: unrecognized hostname (%s)", hostname)
	return "", fmt.Errorf("unrecognized hostname (%s) is not a labstation or beaglebone servo", hostname)
}
