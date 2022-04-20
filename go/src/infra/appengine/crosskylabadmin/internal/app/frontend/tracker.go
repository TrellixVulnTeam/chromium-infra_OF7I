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

package frontend

import (
	"context"
	"fmt"
	"sync"
	"time"

	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/data/strpair"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/grpc/grpcutil"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/internal/app/clients"
	"infra/appengine/crosskylabadmin/internal/app/config"
	swarming_utils "infra/appengine/crosskylabadmin/internal/app/frontend/internal/swarming"
	"infra/cros/lab_inventory/utilization"
)

// SwarmingFactory is a constructor for a SwarmingClient.
type SwarmingFactory func(c context.Context, host string) (clients.SwarmingClient, error)

// TrackerServerImpl implements the fleet.TrackerServer interface.
type TrackerServerImpl struct {
	// SwarmingFactory is an optional factory function for creating clients.
	//
	// If SwarmingFactory is nil, clients.NewSwarmingClient is used.
	SwarmingFactory SwarmingFactory
}

func (tsi *TrackerServerImpl) newSwarmingClient(c context.Context, host string) (clients.SwarmingClient, error) {
	if tsi.SwarmingFactory != nil {
		return tsi.SwarmingFactory(c, host)
	}
	return clients.NewSwarmingClient(c, host)
}

// PushBotsForAdminTasks implements the fleet.Tracker.pushBotsForAdminTasks() method.
func (tsi *TrackerServerImpl) PushBotsForAdminTasks(ctx context.Context, req *fleet.PushBotsForAdminTasksRequest) (res *fleet.PushBotsForAdminTasksResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	cfg := config.Get(ctx)
	sc, err := tsi.newSwarmingClient(ctx, cfg.Swarming.Host)
	if err != nil {
		return nil, errors.Annotate(err, "failed to obtain Swarming client").Err()
	}

	dutState, ok := clients.DutStateRevMap[req.GetTargetDutState()]
	if !ok {
		return nil, fmt.Errorf("DutState=%#v does not map to swarming value", req.GetTargetDutState())
	}

	// Schedule admin tasks to idle DUTs.
	dims := make(strpair.Map)
	dims[clients.DutStateDimensionKey] = []string{dutState}
	bots, err := sc.ListAliveIdleBotsInPool(ctx, cfg.Swarming.BotPool, dims)
	if err != nil {
		reason := fmt.Sprintf("failed to list alive idle cros bots with dut_state %q", dutState)
		return nil, errors.Annotate(err, reason).Err()
	}
	logging.Infof(ctx, "successfully get %d alive idle cros bots with dut_state %q.", len(bots), dutState)

	// Parse BOT id to schedule tasks for readability.
	repairBOTs := identifyBotsForRepair(ctx, bots)
	err = clients.PushRepairDUTs(ctx, repairBOTs, dutState)
	if err != nil {
		logging.Infof(ctx, "push repair bots: %v", err)
		return nil, errors.New("failed to push repair duts")
	}
	return &fleet.PushBotsForAdminTasksResponse{}, nil
}

// PushBotsForAdminAuditTasks implements the fleet.Tracker.pushBotsForAdminTasks() method.
func (tsi *TrackerServerImpl) PushBotsForAdminAuditTasks(ctx context.Context, req *fleet.PushBotsForAdminAuditTasksRequest) (res *fleet.PushBotsForAdminAuditTasksResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	dutStates := map[fleet.DutState]bool{
		fleet.DutState_Ready:             true,
		fleet.DutState_NeedsRepair:       true,
		fleet.DutState_NeedsReset:        true,
		fleet.DutState_RepairFailed:      true,
		fleet.DutState_NeedsManualRepair: true,
		fleet.DutState_NeedsReplacement:  false,
		fleet.DutState_NeedsDeploy:       false,
	}

	var actions []string
	var taskname string
	switch req.Task {
	case fleet.AuditTask_ServoUSBKey:
		actions = []string{"verify-servo-usb-drive"}
		taskname = "USB-drive"
	case fleet.AuditTask_DUTStorage:
		actions = []string{"verify-dut-storage"}
		taskname = "Storage"
		dutStates[fleet.DutState_RepairFailed] = false
		dutStates[fleet.DutState_NeedsManualRepair] = false
	case fleet.AuditTask_RPMConfig:
		actions = []string{"verify-rpm-config"}
		taskname = "RPM Config"
		dutStates[fleet.DutState_RepairFailed] = false
		dutStates[fleet.DutState_NeedsManualRepair] = false
	}

	if len(actions) == 0 {
		logging.Infof(ctx, "No action specified", err)
		return nil, errors.New("failed to push audit bots")
	}

	cfg := config.Get(ctx)
	sc, err := tsi.newSwarmingClient(ctx, cfg.Swarming.Host)
	if err != nil {
		return nil, errors.Annotate(err, "failed to obtain Swarming client").Err()
	}

	// Schedule audit tasks to ready|needs_repair|needs_reset|repair_failed DUTs.
	var bots []*swarming.SwarmingRpcsBotInfo
	f := func() (err error) {
		dims := make(strpair.Map)
		bots, err = sc.ListAliveBotsInPool(ctx, cfg.Swarming.BotPool, dims)
		return err
	}
	err = retry.Retry(ctx, simple3TimesRetry(), f, retry.LogCallback(ctx, "Try get list of the BOTs"))
	if err != nil {
		return nil, errors.Annotate(err, "failed to list alive cros bots").Err()
	}
	logging.Infof(ctx, "successfully get %d alive cros bots", len(bots))
	botIDs := identifyBotsForAudit(ctx, bots, dutStates, req.Task)

	err = clients.PushAuditDUTs(ctx, botIDs, actions, taskname)
	if err != nil {
		logging.Infof(ctx, "failed push audit bots: %v", err)
		return nil, errors.New("failed to push audit bots")
	}
	return &fleet.PushBotsForAdminAuditTasksResponse{}, nil
}

// PushRepairJobsForLabstations implements the fleet.Tracker.pushLabstationsForRepair() method.
func (tsi *TrackerServerImpl) PushRepairJobsForLabstations(ctx context.Context, req *fleet.PushRepairJobsForLabstationsRequest) (res *fleet.PushRepairJobsForLabstationsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()

	cfg := config.Get(ctx)
	sc, err := tsi.newSwarmingClient(ctx, cfg.Swarming.Host)
	if err != nil {
		return nil, errors.Annotate(err, "failed to obtain Swarming client").Err()
	}

	// Schedule repair jobs to idle labstations. It's for periodically checking
	// and rebooting labstations to ensure they're in good state.
	dims := make(strpair.Map)
	dims[clients.DutOSDimensionKey] = []string{"OS_TYPE_LABSTATION"}
	bots, err := sc.ListAliveIdleBotsInPool(ctx, cfg.Swarming.BotPool, dims)
	if err != nil {
		return nil, errors.Annotate(err, "failed to list alive idle labstation bots").Err()
	}
	logging.Infof(ctx, "successfully get %d alive idle labstation bots.", len(bots))

	// Parse BOT id to schedule tasks for readability.
	botIDs := identifyLabstationsForRepair(ctx, bots)

	err = clients.PushRepairLabstations(ctx, botIDs)
	if err != nil {
		logging.Infof(ctx, "push repair labstations: %v", err)
		return nil, errors.New("failed to push repair labstations")
	}
	return &fleet.PushRepairJobsForLabstationsResponse{}, nil
}

// ReportBots reports metrics of swarming bots.
func (tsi *TrackerServerImpl) ReportBots(ctx context.Context, req *fleet.ReportBotsRequest) (res *fleet.ReportBotsResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(ctx, err)
	}()
	cfg := config.Get(ctx)
	sc, err := tsi.newSwarmingClient(ctx, cfg.Swarming.Host)
	if err != nil {
		return nil, errors.Annotate(err, "failed to obtain Swarming client").Err()
	}

	bots, err := sc.ListAliveBotsInPool(ctx, cfg.Swarming.BotPool, strpair.Map{})
	utilization.ReportMetrics(ctx, flattenAndDedpulicateBots([][]*swarming.SwarmingRpcsBotInfo{bots}))
	return &fleet.ReportBotsResponse{}, nil
}

// getBotsFromSwarming lists bots by calling the Swarming service.
func getBotsFromSwarming(ctx context.Context, sc clients.SwarmingClient, pool string, sels []*fleet.BotSelector) ([]*swarming.SwarmingRpcsBotInfo, error) {
	// No filters implies get all bots.
	if len(sels) == 0 {
		bots, err := sc.ListAliveBotsInPool(ctx, pool, strpair.Map{})
		if err != nil {
			return nil, errors.Annotate(err, "failed to get bots in pool %s", pool).Err()
		}
		return bots, nil
	}

	bots := make([][]*swarming.SwarmingRpcsBotInfo, 0, len(sels))
	// Protects access to bots
	m := &sync.Mutex{}
	err := parallel.WorkPool(clients.MaxConcurrentSwarmingCalls, func(workC chan<- func() error) {
		for i := range sels {
			// In-scope variable for goroutine closure.
			sel := sels[i]
			workC <- func() error {
				bs, ierr := getFilteredBotsFromSwarming(ctx, sc, pool, sel)
				if ierr != nil {
					return ierr
				}
				m.Lock()
				defer m.Unlock()
				bots = append(bots, bs)
				return nil
			}
		}
	})
	return flattenAndDedpulicateBots(bots), err
}

// getFilteredBotsFromSwarming lists bots for a single selector by calling the
// Swarming service.
//
// This function is intended to be used in a parallel.WorkPool().
func getFilteredBotsFromSwarming(ctx context.Context, sc clients.SwarmingClient, pool string, sel *fleet.BotSelector) ([]*swarming.SwarmingRpcsBotInfo, error) {
	dims := make(strpair.Map)
	if id := sel.GetDutId(); id != "" {
		dims[clients.DutIDDimensionKey] = []string{id}
	}
	if m := sel.GetDimensions().GetModel(); m != "" {
		dims[clients.DutModelDimensionKey] = []string{m}
	}
	if p := sel.GetDimensions().GetPools(); len(p) > 0 {
		dims[clients.DutPoolDimensionKey] = p
	}
	if n := sel.GetDimensions().GetDutName(); n != "" {
		dims[clients.DutNameDimensionKey] = []string{n}
	}

	if len(dims) == 0 {
		return nil, fmt.Errorf("empty selector %v", sel)
	}
	bs, err := sc.ListAliveBotsInPool(ctx, pool, dims)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get bots in pool %s with dimensions %s", pool, dims).Err()
	}
	return bs, nil
}

func flattenAndDedpulicateBots(nb [][]*swarming.SwarmingRpcsBotInfo) []*swarming.SwarmingRpcsBotInfo {
	bm := make(map[string]*swarming.SwarmingRpcsBotInfo)
	for _, bs := range nb {
		for _, b := range bs {
			bm[b.BotId] = b
		}
	}
	bots := make([]*swarming.SwarmingRpcsBotInfo, 0, len(bm))
	for _, v := range bm {
		bots = append(bots, v)
	}
	return bots
}

var dutStatesForRepairTask = map[fleet.DutState]bool{
	fleet.DutState_NeedsRepair:       true,
	fleet.DutState_RepairFailed:      true,
	fleet.DutState_NeedsManualRepair: true,
}

// identifyBotsForRepair identifies duts that need run admin repair.
func identifyBotsForRepair(ctx context.Context, bots []*swarming.SwarmingRpcsBotInfo) (repairBOTs []string) {
	repairBOTs = make([]string, 0, len(bots))
	for _, b := range bots {
		dims := swarming_utils.DimensionsMap(b.Dimensions)
		os, err := swarming_utils.ExtractSingleValuedDimension(dims, clients.DutOSDimensionKey)
		// Some bot may not have os dimension(e.g. scheduling unit), so we ignore the error here.
		if err == nil && os == "OS_TYPE_LABSTATION" {
			continue
		}
		id, err := swarming_utils.ExtractSingleValuedDimension(dims, clients.BotIDDimensionKey)
		if err != nil {
			logging.Warningf(ctx, "failed to obtain BOT id for bot %q", b.BotId)
			continue
		}

		s := clients.GetStateDimension(b.Dimensions)
		if dutStatesForRepairTask[s] {
			logging.Infof(ctx, "BOT: %s - Needs repair", id)
			repairBOTs = append(repairBOTs, id)
		}
	}
	return repairBOTs
}

// identifyBotsForAudit identifies duts to run admin audit.
func identifyBotsForAudit(ctx context.Context, bots []*swarming.SwarmingRpcsBotInfo, dutStateMap map[fleet.DutState]bool, auditTask fleet.AuditTask) []string {
	logging.Infof(ctx, "Filtering bots for task: %s", auditTask)
	botIDs := make([]string, 0, len(bots))
	for _, b := range bots {
		dims := swarming_utils.DimensionsMap(b.Dimensions)
		os, err := swarming_utils.ExtractSingleValuedDimension(dims, clients.DutOSDimensionKey)
		// Some bot may not have os dimension(e.g. scheduling unit), so we ignore the error here.
		if err == nil && os == "OS_TYPE_LABSTATION" {
			continue
		}

		id, err := swarming_utils.ExtractSingleValuedDimension(dims, clients.BotIDDimensionKey)
		if err != nil {
			logging.Warningf(ctx, "failed to obtain BOT id for bot %q", b.BotId)
			continue
		}
		switch auditTask {
		case fleet.AuditTask_ServoUSBKey:
			// Disable skip to verify flakiness. (b/229656121)
			// state := swarming_utils.ExtractBotState(b).ServoUSBState
			// if len(state) > 0 && state[0] == "NEED_REPLACEMENT" {
			// 	logging.Infof(ctx, "Skipping BOT with id: %q as USB-key marked for replacement", b.BotId)
			// 	continue
			// }
		case fleet.AuditTask_DUTStorage:
			state := swarming_utils.ExtractBotState(b).StorageState
			if len(state) > 0 && state[0] == "NEED_REPLACEMENT" {
				logging.Infof(ctx, "Skipping BOT with id: %q as storage marked for replacement", b.BotId)
				continue
			}
		case fleet.AuditTask_RPMConfig:
			state := swarming_utils.ExtractBotState(b).RpmState
			if len(state) > 0 && state[0] != "UNKNOWN" {
				// expecting that RPM is going through check everytime when we do any update on setup.
				logging.Infof(ctx, "Skipping BOT with id: %q as RPM was already audited", b.BotId)
				continue
			}
		}

		s := clients.GetStateDimension(b.Dimensions)
		if v, ok := dutStateMap[s]; ok && v {
			botIDs = append(botIDs, id)
		} else {
			logging.Infof(ctx, "Skipping BOT with id: %q", b.BotId)
		}
	}
	return botIDs
}

// identifyLabstationsForRepair identifies labstations that need repair.
func identifyLabstationsForRepair(ctx context.Context, bots []*swarming.SwarmingRpcsBotInfo) []string {
	botIDs := make([]string, 0, len(bots))
	for _, b := range bots {
		dims := swarming_utils.DimensionsMap(b.Dimensions)
		os, err := swarming_utils.ExtractSingleValuedDimension(dims, clients.DutOSDimensionKey)
		if err != nil {
			logging.Warningf(ctx, "failed to obtain os type for bot %q", b.BotId)
			continue
		} else if os != "OS_TYPE_LABSTATION" {
			continue
		}

		id, err := swarming_utils.ExtractSingleValuedDimension(dims, clients.BotIDDimensionKey)
		if err != nil {
			logging.Warningf(ctx, "failed to obtain BOT id for bot %q", b.BotId)
			continue
		}

		botIDs = append(botIDs, id)
	}
	return botIDs
}

// simple3TimesRetryIterator simple retry iterator to try 3 times.
var simple3TimesRetryIterator = retry.ExponentialBackoff{
	Limited: retry.Limited{
		Delay:   200 * time.Millisecond,
		Retries: 3,
	},
}

// simple3TimesRetry returns a retry.Factory based on simple3TimesRetryIterator.
func simple3TimesRetry() retry.Factory {
	return func() retry.Iterator {
		return &simple3TimesRetryIterator
	}
}
