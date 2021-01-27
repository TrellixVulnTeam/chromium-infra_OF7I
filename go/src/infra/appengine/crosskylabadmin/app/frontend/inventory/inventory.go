// Copyright 2018 The LUCI Authors.
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
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/kylelemons/godebug/pretty"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	authclient "go.chromium.org/luci/auth"
	gitilesApi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/clients"
	"infra/appengine/crosskylabadmin/app/config"
	dataSV "infra/appengine/crosskylabadmin/app/frontend/internal/datastore/stableversion"
	"infra/appengine/crosskylabadmin/app/gitstore"
	"infra/cros/lab_inventory/manufacturingconfig"
	sv "infra/cros/stableversion"
	"infra/libs/git"
	"infra/libs/skylab/inventory"
)

var prettyConfig = &pretty.Config{
	TrackCycles: true,
}

// GerritFactory is a contsructor for a GerritClient
type GerritFactory func(c context.Context, host string) (gitstore.GerritClient, error)

// GitilesFactory is a contsructor for a GerritClient
type GitilesFactory func(c context.Context, host string) (gitstore.GitilesClient, error)

// SwarmingFactory is a constructor for a SwarmingClient.
type SwarmingFactory func(c context.Context, host string) (clients.SwarmingClient, error)

// TrackerFactory is a constructor for a TrackerServer object.
type TrackerFactory func() fleet.TrackerServer

// StableVersionGitClientFactory is a constructor for a git client pointed at the source of truth
// for the stable version information
type StableVersionGitClientFactory func(c context.Context) (git.ClientInterface, error)

// ServerImpl implements the fleet.InventoryServer interface.
type ServerImpl struct {
	// GerritFactory is an optional factory function for creating gerrit client.
	//
	// If GerritFactory is nil, clients.NewGerritClient is used.
	GerritFactory GerritFactory

	// GitilesFactory is an optional factory function for creating gitiles client.
	//
	// If GitilesFactory is nil, clients.NewGitilesClient is used.
	GitilesFactory GitilesFactory

	// SwarmingFactory is an optional factory function for creating clients.
	//
	// If SwarmingFactory is nil, clients.NewSwarmingClient is used.
	SwarmingFactory SwarmingFactory

	// TrackerServerFactory is a required factory function for creating a tracker object.
	//
	// TODO(pprabhu) Move tracker/tasker to individual sub-packages and inject
	// dependencies directly (instead of factory functions).
	TrackerFactory TrackerFactory

	// StableVersionGitClientFactory
	StableVersionGitClientFactory StableVersionGitClientFactory

	// updateLimiter rate limits UpdateDutLabels.
	updateLimiter     rateLimiter
	updateLimiterOnce sync.Once
}

type getStableVersionRecordsResult struct {
	cros     map[string]string
	faft     map[string]string
	firmware map[string]string
}

var transientErrorRetriesTemplate = retry.ExponentialBackoff{
	Limited: retry.Limited{
		Delay: 200 * time.Millisecond,
		// Don't retry too often, leaving some headroom for clients to retry if they wish.
		Retries: 3,
	},
	// Slow down quickly so as to not flood outbound requests on retries.
	Multiplier: 4,
	MaxDelay:   5 * time.Second,
}

// transientErrorRetries returns a retry.Factory to use on transient errors on
// outbound requests.
func transientErrorRetries() retry.Factory {
	next := func() retry.Iterator {
		it := transientErrorRetriesTemplate
		return &it
	}
	return transient.Only(next)
}

func (is *ServerImpl) newGerritClient(c context.Context, host string) (gitstore.GerritClient, error) {
	if is.GerritFactory != nil {
		return is.GerritFactory(c, host)
	}
	return clients.NewGerritClient(c, host)
}

func (is *ServerImpl) newGitilesClient(c context.Context, host string) (gitstore.GitilesClient, error) {
	if is.GitilesFactory != nil {
		return is.GitilesFactory(c, host)
	}
	return clients.NewGitilesClient(c, host)
}

func (is *ServerImpl) newSwarmingClient(ctx context.Context, host string) (clients.SwarmingClient, error) {
	if is.SwarmingFactory != nil {
		return is.SwarmingFactory(ctx, host)
	}
	return clients.NewSwarmingClient(ctx, host)
}

func (is *ServerImpl) newStore(ctx context.Context) (*gitstore.InventoryStore, error) {
	inventoryConfig := config.Get(ctx).Inventory
	gerritC, err := is.newGerritClient(ctx, inventoryConfig.GerritHost)
	if err != nil {
		return nil, errors.Annotate(err, "create inventory store").Err()
	}
	gitilesC, err := is.newGitilesClient(ctx, inventoryConfig.GitilesHost)
	if err != nil {
		return nil, errors.Annotate(err, "create inventory store").Err()
	}
	return gitstore.NewInventoryStore(gerritC, gitilesC), nil
}

func (is *ServerImpl) newInventoryClient(ctx context.Context) (inventoryClient, error) {
	cfg := config.Get(ctx).InventoryProvider
	gitstore, err := is.newStore(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "create duo client").Err()
	}

	client, err := newDuoClient(ctx, gitstore, cfg.GetHost(), int(cfg.GetReadTrafficRatio()), int(cfg.GetWriteTrafficRatio()), cfg.GetTestingDeviceUuids(), cfg.GetTestingDeviceNames(), cfg.GetInventoryV2Only())
	if err != nil {
		return nil, errors.Annotate(err, "create duo client").Err()
	}
	return client, nil
}

func (is *ServerImpl) newStableVersionGitClient(ctx context.Context) (git.ClientInterface, error) {
	if is.StableVersionGitClientFactory != nil {
		return is.StableVersionGitClientFactory(ctx)
	}
	hc, err := getAuthenticatedHTTPClient(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "newStableVersionGitClient").Err()
	}
	return getStableVersionGitClient(ctx, hc)
}

// UpdateDeviceConfig implements updating device config to inventory.
func (is *ServerImpl) UpdateDeviceConfig(ctx context.Context, req *fleet.UpdateDeviceConfigRequest) (resp *fleet.UpdateDeviceConfigResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	cfg := config.Get(ctx).Inventory
	gitilesC, err := is.newGitilesClient(ctx, cfg.GitilesHost)
	if err != nil {
		return nil, errors.Annotate(err, "fail to update device config").Err()
	}
	deviceConfigs, err := GetDeviceConfig(ctx, gitilesC)
	if err != nil {
		return nil, errors.Annotate(err, "fail to fetch device configs").Err()
	}
	err = SaveDeviceConfig(ctx, deviceConfigs)
	if err != nil {
		return nil, errors.Annotate(err, "fail to save device config to datastore").Err()
	}
	store, err := is.newStore(ctx)
	if err != nil {
		return nil, err
	}
	if err := store.Refresh(ctx); err != nil {
		return nil, errors.Annotate(err, "fail to refresh inventory store").Err()
	}
	url, err := updateDeviceConfig(ctx, deviceConfigs, store)
	if err != nil {
		return nil, err
	}
	logging.Infof(ctx, "successfully update device config: %s", url)

	return &fleet.UpdateDeviceConfigResponse{}, nil
}

func updateDeviceConfig(ctx context.Context, deviceConfigs map[string]*device.Config, s *gitstore.InventoryStore) (string, error) {
	for _, d := range s.Lab.GetDuts() {
		c := d.GetCommon()
		dcID := getIDForInventoryLabels(ctx, c.GetLabels())
		standardDC, ok := deviceConfigs[dcID]
		if !ok {
			continue
		}
		inventory.ConvertDeviceConfig(standardDC, c)
	}
	url, err := s.Commit(ctx, fmt.Sprintf("Update device config"))
	if gitstore.IsEmptyErr(err) {
		return "no commit for empty diff", nil
	}
	if err != nil {
		return "", errors.Annotate(err, "fail to update device config").Err()
	}
	return url, nil
}

// UpdateManufacturingConfig backfill parts of manufacturing config to inventory V1.
func (is *ServerImpl) UpdateManufacturingConfig(ctx context.Context, req *fleet.UpdateManufacturingConfigRequest) (resp *fleet.UpdateManufacturingConfigResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	cfg := config.Get(ctx).Inventory
	gitilesC, err := is.newGitilesClient(ctx, cfg.GitilesHost)
	if err != nil {
		return nil, errors.Annotate(err, "fail to update manufacturing config").Err()
	}
	configs, err := GetManufacturingConfig(ctx, gitilesC)
	if err != nil {
		return nil, errors.Annotate(err, "fail to fetch manufacturing configs").Err()
	}
	store, err := is.newStore(ctx)
	if err != nil {
		return nil, err
	}
	if err := store.Refresh(ctx); err != nil {
		return nil, errors.Annotate(err, "fail to refresh inventory store").Err()
	}
	url, err := updateManufacturingConfig(ctx, configs, store)
	if err != nil {
		return nil, err
	}
	logging.Infof(ctx, "successfully update manufacturing config: %s", url)

	return &fleet.UpdateManufacturingConfigResponse{
		ChangeUrl: url,
	}, nil
}

func updateManufacturingConfig(ctx context.Context, configs map[string]*manufacturing.Config, s *gitstore.InventoryStore) (string, error) {
	for _, d := range s.Lab.GetDuts() {
		if looksLikeLabstation(d.GetCommon().GetHostname()) {
			continue
		}
		hwid, err := getHWID(d)
		if err != nil || hwid == "" {
			logging.Errorf(ctx, "missing HWID: %s", err)
			continue
		}
		c, ok := configs[hwid]
		if !ok {
			logging.Errorf(ctx, "non-existing HWID: %s (%s)", hwid, d.GetCommon().GetHostname())
			continue
		}
		l := d.GetCommon().GetLabels()
		manufacturingconfig.ConvertMCToV1Labels(c, l)
	}
	url, err := s.Commit(ctx, fmt.Sprintf("Update manufacturing config"))
	if gitstore.IsEmptyErr(err) {
		return "no commit for empty diff", nil
	}
	if err != nil {
		return "", errors.Annotate(err, "fail to update manufacturing config").Err()
	}
	return url, nil
}

func getHWID(dut *inventory.DeviceUnderTest) (string, error) {
	attrs := dut.GetCommon().GetAttributes()
	if len(attrs) == 0 {
		return "", fmt.Errorf("attributes for dut with hostname (%s) is unexpectedly empty", dut.GetCommon().GetHostname())
	}
	for _, item := range attrs {
		key := item.GetKey()
		value := item.GetValue()
		if key == "HWID" {
			return value, nil
		}
	}
	return "", fmt.Errorf("no \"HWID\" attribute for hostname (%s)", dut.GetCommon().GetHostname())
}

// UpdateDutLabels implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) UpdateDutLabels(ctx context.Context, req *fleet.UpdateDutLabelsRequest) (resp *fleet.UpdateDutLabelsResponse, err error) {
	// TODO fully remove it when remove all calls of thi method
	// https://crbug.com/1092948
	return nil, nil
}

func validateBatchUpdateDutsRequest(req *fleet.BatchUpdateDutsRequest) error {
	if len(req.GetDutProperties()) == 0 {
		return errors.New("must specify at least dut property for one host")
	}

	if len(req.GetHostnames()) > 0 {
		return errors.New("'hostnames' field in BatchUpdateRequest is deprecated, please update skylab tool")
	}

	if req.GetPool() != "" {
		return errors.New("'pool' field in BatchUpdateRequest is deprecated, please update skylab tool")
	}
	return nil
}

// BatchUpdateDuts implements the method from fleet.InventoryServer interface.
func (is *ServerImpl) BatchUpdateDuts(ctx context.Context, req *fleet.BatchUpdateDutsRequest) (resp *fleet.BatchUpdateDutsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	if err := validateBatchUpdateDutsRequest(req); err != nil {
		return nil, errors.Annotate(err, "invalid BatchUpdateDutsRequest").Err()
	}

	duts := req.GetDutProperties()
	hostnameToProperty := make(map[string]*fleet.DutProperty, len(duts))
	for _, d := range duts {
		hostnameToProperty[d.GetHostname()] = d
	}

	store, err := is.newStore(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "BatchUpdateDuts").Err()
	}
	if err := store.Refresh(ctx); err != nil {
		return nil, errors.Annotate(err, "BatchUpdateDuts").Err()
	}

	for _, d := range store.Lab.GetDuts() {
		hostname := d.GetCommon().GetHostname()
		if hostname == "" {
			logging.Infof(ctx, "empty hostname for dut %#v", d)
			continue
		}
		if dp, ok := hostnameToProperty[hostname]; ok {
			pool := dp.GetPool()
			if pool == "" {
				logging.Infof(ctx, "skip pool update for host %s as no pool is passed in", hostname)
			} else {
				assignPool(d, pool)
			}
			rpm := dp.GetRpm()
			if rpm == nil {
				logging.Infof(ctx, "skip rpm update for host %s as no rpm is passed in", hostname)
			} else {
				assignRpm(d, rpm)
			}
		}
	}

	url, err := store.Commit(ctx, "Batch update DUT labels")
	if gitstore.IsEmptyErr(err) {
		logging.Infof(ctx, "no updates, so nothing to commit")
		return &fleet.BatchUpdateDutsResponse{}, nil
	}
	if err != nil {
		logging.Infof(ctx, "commit failure: %v", err)
		return nil, errors.Annotate(err, "BatchUpdateDuts").Err()
	}

	return &fleet.BatchUpdateDutsResponse{
		Url: url,
	}, nil
}

// Assign pool to a single device.
func assignPool(d *inventory.DeviceUnderTest, pool string) {
	cp, ok := inventory.SchedulableLabels_DUTPool_value[pool]
	if ok {
		d.GetCommon().GetLabels().CriticalPools = []inventory.SchedulableLabels_DUTPool{inventory.SchedulableLabels_DUTPool(cp)}
		d.GetCommon().GetLabels().SelfServePools = nil
	} else {
		d.GetCommon().GetLabels().CriticalPools = nil
		d.GetCommon().GetLabels().SelfServePools = []string{pool}
	}
}

// Assign rpm info to a single device.
func assignRpm(d *inventory.DeviceUnderTest, rpm *fleet.DutProperty_Rpm) {
	attrs := d.GetCommon().GetAttributes()
	attrs = setOrAppend(attrs, "powerunit_hostname", rpm.GetPowerunitHostname())
	attrs = setOrAppend(attrs, "powerunit_outlet", rpm.GetPowerunitOutlet())
	d.GetCommon().Attributes = attrs
}

func setOrAppend(attrs []*inventory.KeyValue, key string, value string) []*inventory.KeyValue {
	for _, att := range attrs {
		if att.GetKey() == key {
			att.Value = &value
			return attrs
		}
	}
	attrs = append(attrs, &inventory.KeyValue{
		Key:   &key,
		Value: &value,
	})
	return attrs
}

// PushInventoryToQueen implements the method from fleet.InventoryServer interface.
//
// This RPC is deprecated. The cron job is moved to InventoryV2[pushToDroneQueenCronHandler].
// https://source.corp.google.com/chromium_infra/go/src/infra/appengine/cros/lab_inventory/app/cron/cron.go;l=312
func (is *ServerImpl) PushInventoryToQueen(ctx context.Context, req *fleet.PushInventoryToQueenRequest) (resp *fleet.PushInventoryToQueenResponse, err error) {
	return &fleet.PushInventoryToQueenResponse{}, nil
}

// queenDroneName returns the name of the fake drone whose DUTs should
// be pushed to the drone queen service.
func queenDroneName(env string) string {
	return queenDronePrefix + env
}

const queenDronePrefix = "drone-queen-"

// DumpStableVersionToDatastore takes stable version info from the git repo where it lives
// and dumps it to datastore
func (is *ServerImpl) DumpStableVersionToDatastore(ctx context.Context, in *fleet.DumpStableVersionToDatastoreRequest) (*fleet.DumpStableVersionToDatastoreResponse, error) {
	client, err := is.newStableVersionGitClient(ctx)
	if err != nil {
		logging.Errorf(ctx, "get git client: %s", err)
		return nil, errors.Annotate(err, "get git client").Err()
	}
	return dumpStableVersionToDatastoreImpl(ctx, client.GetFile)
}

// dumpStableVersionToDatastoreImpl takes some way of getting a file and a context and writes to datastore
func dumpStableVersionToDatastoreImpl(ctx context.Context, getFile func(context.Context, string) (string, error)) (*fleet.DumpStableVersionToDatastoreResponse, error) {
	contents, err := getFile(ctx, config.Get(ctx).StableVersionConfig.StableVersionDataPath)
	if err != nil {
		logging.Errorf(ctx, "fetch file: %s", err)
		return nil, errors.Annotate(err, "fetch file").Err()
	}
	stableVersions, err := parseStableVersions(contents)
	if err != nil {
		logging.Errorf(ctx, "parse json: %s", err)
		return nil, errors.Annotate(err, "parse json").Err()
	}
	m := getStableVersionRecords(ctx, stableVersions)
	merr := errors.NewMultiError()
	if err := dataSV.PutManyCrosStableVersion(ctx, m.cros); err != nil {
		merr = append(merr, errors.Annotate(err, "put cros stable version").Err())
	}
	if err := dataSV.PutManyFirmwareStableVersion(ctx, m.firmware); err != nil {
		merr = append(merr, errors.Annotate(err, "put firmware stable version").Err())
	}
	if err := dataSV.PutManyFaftStableVersion(ctx, m.faft); err != nil {
		merr = append(merr, errors.Annotate(err, "put firmware stable version").Err())
	}
	if len(merr) != 0 {
		logging.Errorf(ctx, "error writing stable versions: %s", merr)
		return nil, merr
	}
	logging.Infof(ctx, "successfully wrote stable versions")
	return &fleet.DumpStableVersionToDatastoreResponse{}, nil
}

type updateDutLabelsRequest struct {
	dutID     string
	labels    *inventory.SchedulableLabels
	reason    string
	oldLabels *inventory.SchedulableLabels
}

func getDUTByID(lab *inventory.Lab, id string) (*inventory.DeviceUnderTest, bool) {
	for _, d := range lab.GetDuts() {
		if d.GetCommon().GetId() == id {
			return d, true
		}
	}
	return nil, false
}

func mapHostnameToDUTs(duts []*inventory.DeviceUnderTest) map[string]*inventory.DeviceUnderTest {
	m := make(map[string]*inventory.DeviceUnderTest)
	for _, d := range duts {
		m[d.GetCommon().GetHostname()] = d
	}
	return m
}

func parseStableVersions(contents string) (*lab_platform.StableVersions, error) {
	var stableVersions lab_platform.StableVersions
	if err := jsonpb.Unmarshal(strings.NewReader(contents), &stableVersions); err != nil {
		return nil, errors.Annotate(err, "unmarshal stableversions json").Err()
	}
	return &stableVersions, nil
}

// getStableVersionRecords takes a StableVersions proto and produces a structure containing maps from
// key names (buildTarget or buildTarget+model) to stable version strings
func getStableVersionRecords(ctx context.Context, stableVersions *lab_platform.StableVersions) getStableVersionRecordsResult {
	cros := make(map[string]string)
	faft := make(map[string]string)
	firmware := make(map[string]string)
	for _, item := range stableVersions.GetCros() {
		buildTarget := item.GetKey().GetBuildTarget().GetName()
		model := item.GetKey().GetModelId().GetValue()
		version := item.GetVersion()
		key, err := sv.JoinBuildTargetModel(buildTarget, model)
		if err != nil {
			logging.Infof(ctx, "buildTarget and/or model contains invalid sequence: %s", err)
			continue
		}
		cros[key] = version
	}
	for _, item := range stableVersions.GetFirmware() {
		buildTarget := item.GetKey().GetBuildTarget().GetName()
		model := item.GetKey().GetModelId().GetValue()
		version := item.GetVersion()
		key, err := sv.JoinBuildTargetModel(buildTarget, model)
		if err != nil {
			logging.Infof(ctx, "buildTarget and/or model contains invalid sequence: %s", err)
			continue
		}
		firmware[key] = version
	}
	for _, item := range stableVersions.GetFaft() {
		buildTarget := item.GetKey().GetBuildTarget().GetName()
		model := item.GetKey().GetModelId().GetValue()
		version := item.GetVersion()
		key, err := sv.JoinBuildTargetModel(buildTarget, model)
		if err != nil {
			logging.Infof(ctx, "buildTarget and/or model contains invalid sequence: %s", err)
			continue
		}
		faft[key] = version
	}
	return getStableVersionRecordsResult{
		cros:     cros,
		faft:     faft,
		firmware: firmware,
	}
}

func getAuthenticatedHTTPClient(ctx context.Context) (*http.Client, error) {
	transport, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, gitilesApi.OAuthScope))
	if err != nil {
		return nil, errors.Annotate(err, "new authenticated http client").Err()
	}
	return &http.Client{Transport: transport}, nil
}

func getStableVersionGitClient(ctx context.Context, hc *http.Client) (git.ClientInterface, error) {
	cfg := config.Get(ctx)
	s := cfg.StableVersionConfig
	if s == nil {
		return nil, fmt.Errorf("DumpStableVersionToDatastore: app config does not have StableVersionConfig")
	}
	client, err := git.NewClient(ctx, hc, s.GerritHost, s.GitilesHost, s.Project, s.Branch)
	if err != nil {
		return nil, errors.Annotate(err, "get git client").Err()
	}
	return client, nil
}
