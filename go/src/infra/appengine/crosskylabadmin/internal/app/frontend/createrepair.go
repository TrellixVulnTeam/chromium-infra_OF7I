// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/internal/app/clients"
	"infra/appengine/crosskylabadmin/internal/app/config"
	"infra/appengine/crosskylabadmin/internal/app/frontend/internal/worker"
	"infra/appengine/crosskylabadmin/site"
	"infra/cros/recovery/tasknames"
	"infra/libs/skylab/buildbucket"
	"infra/libs/skylab/buildbucket/labpack"
	"infra/libs/skylab/common/heuristics"
)

// Paris represents a decision to use the paris stack for this request.
const paris = "paris"

// Legacy represents a decision to use the legacy stack for this request.
const legacy = "legacy"

// Reason is a rationale for why we made the decision that we made.
type reason int

const (
	parisNotEnabled reason = iota
	allDevicesAreOptedIn
	noPools
	wrongPool
	scoreBelowThreshold
	scoreTooHigh
	thresholdZero
	malformedPolicy
	nilArgument
	notALabstation
)

// ReasonMessageMap maps each reason to a readable description.
var reasonMessageMap = map[reason]string{
	parisNotEnabled:      "PARIS is not enabled",
	allDevicesAreOptedIn: "All devices are opted in",
	noPools:              "Device has no pools, possibly due to error calling UFS",
	wrongPool:            "Device has a pool not matching opted-in pools",
	scoreBelowThreshold:  "Random score associated with is below threshold, authorizing new flow",
	scoreTooHigh:         "Random score associated with task is too high",
	thresholdZero:        "Route labstation repair task: a threshold of zero implies that optinAllLabstations should be set, but optinAllLabstations is not set",
	malformedPolicy:      "Unrecognized policy",
	nilArgument:          "A required argument was unexpectedly nil",
	notALabstation:       "Paris not enabled yet for non-labstations",
}

// UFSErrorPolicy controls how UFS errors are handled.
type ufsErrorPolicy string

// UFS error policy constants.
// Error policy constants are defined in go/src/infra/appengine/crosskylabadmin/app/config/config.proto.
//
// Strict   -- fail on UFS error even if we don't need the result
// Fallback -- if we encounter a UFS error, fall back to the legacy path.
// Lax      -- if we do not need the UFS response to make a decision, do not fail the request.
const (
	// The strict policy causes all UFS error requests to be treated as fatal and causes the request to fail.
	ufsErrorPolicyStrict   ufsErrorPolicy = "strict"
	ufsErrorPolicyFallback                = "fallback"
	ufsErrorPolicyLax                     = "lax"
)

// NormalizeError policy normalizes a string into the canonical name for a policy.
func normalizeErrorPolicy(policy string) (ufsErrorPolicy, error) {
	policy = strings.ToLower(policy)
	switch policy {
	case "", "default", "fallback":
		return ufsErrorPolicyFallback, nil
	case "strict":
		return ufsErrorPolicyStrict, nil
	case "lax":
		return ufsErrorPolicyLax, nil
	}
	return "", fmt.Errorf("unrecognized policy: %q", policy)
}

// RouteRepairTask routes a repair task for a given bot.
//
// The possible return values are:
// - "legacy"  (for legacy, which is the default)
// -       ""  (indicates an error, should be treated as equivalent to "legacy" by callers)
// -  "paris"  (for PARIS, which is new)
//
// RouteRepairTask takes as an argument randFloat (which is a float64 in the closed interval [0, 1]).
// This argument is, by design, all the entropy that randFloat will need. Taking this as an argument allows
// RouteRepairTask itself to be deterministic because the caller is responsible for generating the random
// value.
func RouteRepairTask(ctx context.Context, botID string, expectedState string, pools []string, randFloat float64) (string, error) {
	if !(0.0 <= randFloat && randFloat <= 1.0) {
		return "", fmt.Errorf("Route repair task: randfloat %f is not in [0, 1]", randFloat)
	}
	var out string
	var r reason
	switch {
	case heuristics.LooksLikeLabstation(botID):
		out, r = routeRepairTaskImpl(
			ctx,
			config.Get(ctx).GetParis().GetLabstationRepair(),
			&dutRoutingInfo{
				labstation: heuristics.LooksLikeLabstation(botID),
				pools:      pools,
			},
			randFloat,
		)
	default:
		out, r = routeRepairTaskImpl(
			ctx,
			config.Get(ctx).GetParis().GetDutRepair(),
			&dutRoutingInfo{
				labstation: heuristics.LooksLikeLabstation(botID),
				pools:      pools,
			},
			randFloat,
		)
	}
	reason, ok := reasonMessageMap[r]
	if !ok {
		logging.Infof(ctx, "Unrecognized reason %d", int64(r))
	}
	logging.Infof(ctx, "Sending device repair to %q because %q", out, reason)
	return out, nil
}

// CreateRepairTask kicks off a repair job.
//
// This function will either schedule a legacy repair task or a PARIS repair task.
// Note that the ufs client can be nil.
func CreateRepairTask(ctx context.Context, botID string, expectedState string, pools []string, randFloat float64) (string, error) {
	logging.Infof(ctx, "Creating repair task for %q expected state %q with random input %f", botID, expectedState, randFloat)
	// If we encounter an error picking paris or legacy, do the safe thing and use legacy.
	taskType, err := RouteRepairTask(ctx, botID, expectedState, pools, randFloat)
	if err != nil {
		logging.Infof(ctx, "Create repair task: falling back to legacy repair by default: %s", err)
	}
	switch taskType {
	case "paris":
		url, err := createBuildbucketRepairTask(ctx, botID, expectedState)
		if err != nil {
			logging.Errorf(ctx, "Attempted and failed to create buildbucket task: %s", err)
			logging.Errorf(ctx, "Falling back to legacy flow")
			url, err = createLegacyRepairTask(ctx, botID, expectedState)
			return url, errors.Annotate(err, "fallback legacy repair task somehow failed").Err()
		}
		return url, err
	default:
		return createLegacyRepairTask(ctx, botID, expectedState)
	}
}

// DUTRoutingInfo is all the deterministic information about a DUT that is necessary to decide
// whether to use a legacy task or a paris task.
//
// For example, we DO need to know whether a DUT is a labstation or not, but we DO NOT need to know
// what the exact hostname is.
type dutRoutingInfo struct {
	labstation bool
	pools      []string
}

// RouteLabstationRepairTask takes a repair task for a labstation and routes it.
func routeRepairTaskImpl(ctx context.Context, r *config.RolloutConfig, info *dutRoutingInfo, randFloat float64) (string, reason) {
	if info == nil {
		logging.Errorf(ctx, "info cannot be nil, falling back to legacy")
		return legacy, nilArgument
	}
	// TODO(gregorynisbet): Log instead of silently falling back to the default error handling policy.
	ufsErrorPolicy, err := normalizeErrorPolicy(r.GetUfsErrorPolicy())
	if err != nil {
		logging.Infof(ctx, "error while routing labstation repair task: %s", err)
	}
	// Check that the feature is enabled at all.
	if !r.GetEnable() {
		return legacy, parisNotEnabled
	}
	// Check for malformed input data that would cause us to be unable to make a decision.
	if len(info.pools) == 0 {
		switch ufsErrorPolicy {
		case ufsErrorPolicyLax:
			// Do nothing.
		case ufsErrorPolicyStrict:
			// TODO(gregorynisbet): Make strict error handling propagate the failure back up.
			return legacy, noPools
		case ufsErrorPolicyFallback:
			return legacy, noPools
		default:
			return legacy, malformedPolicy
		}
	}
	threshold := r.GetRolloutPermille()
	myValue := math.Round(1000.0 * randFloat)
	// If the threshold is zero, let's reject all possible values of myValue.
	// This way a threshold of zero actually means 0.0% instead of 0.1%.
	valueBelowThreshold := threshold != 0 && myValue <= float64(threshold)
	if r.GetOptinAllDuts() {
		if valueBelowThreshold {
			return paris, scoreBelowThreshold
		}
		return legacy, scoreTooHigh
	}
	if threshold == 0 {
		return legacy, thresholdZero
	}
	if !r.GetOptinAllDuts() && len(r.GetOptinDutPool()) > 0 && isDisjoint(info.pools, r.GetOptinDutPool()) {
		return legacy, wrongPool
	}
	if valueBelowThreshold {
		return paris, scoreBelowThreshold
	}
	return legacy, scoreTooHigh
}

// CreateBuildbucketRepairTask creates a new repair task for a labstation.
// Err should be non-nil if and only if a task was created.
// We rely on this signal to decide whether to fall back to the legacy flow.
func createBuildbucketRepairTask(ctx context.Context, botID string, expectedState string) (string, error) {
	logging.Infof(ctx, "Using new repair flow for bot %q with expected state %q", botID, expectedState)
	transport, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return "", errors.Annotate(err, "failed to get RPC transport").Err()
	}
	hc := &http.Client{
		Transport: transport,
	}
	bc, err := buildbucket.NewClient2(ctx, hc, site.DefaultPRPCOptions, "chromeos", "labpack", "labpack")
	if err != nil {
		logging.Errorf(ctx, "error creating buildbucket client: %q", err)
		return "", errors.Annotate(err, "create buildbucket repair task").Err()
	}
	p := &labpack.Params{
		UnitName:       heuristics.NormalizeBotNameToDeviceName(botID),
		TaskName:       string(tasknames.Recovery),
		EnableRecovery: true,
		// TODO(gregorynisbet): This is our own name, move it to the config.
		AdminService: "chromeos-skylab-bot-fleet.appspot.com",
		// NOTE: We use the UFS service, not the Inventory service here.
		InventoryService: config.Get(ctx).GetUFS().GetHost(),
		NoStepper:        false,
		NoMetrics:        false,
		UpdateInventory:  true,
		// TODO(gregorynisbet): Pass config file to labpack task.
		Configuration: "",
	}
	taskID, err := labpack.ScheduleTask(ctx, bc, labpack.CIPDProd, p)
	if err != nil {
		logging.Errorf(ctx, "error scheduling task: %q", err)
		return "", errors.Annotate(err, "create buildbucket repair task").Err()
	}
	return bc.BuildURL(taskID), nil
}

// CreateLegacyRepairTask creates a legacy repair task for a labstation.
func createLegacyRepairTask(ctx context.Context, botID string, expectedState string) (string, error) {
	logging.Infof(ctx, "Using legacy repair flow for bot %q", botID)
	at := worker.AdminTaskForType(ctx, fleet.TaskType_Repair)
	sc, err := clients.NewSwarmingClient(ctx, config.Get(ctx).Swarming.Host)
	if err != nil {
		return "", errors.Annotate(err, "failed to obtain swarming client").Err()
	}
	cfg := config.Get(ctx)
	taskURL, err := runTaskByBotID(ctx, at, sc, botID, expectedState, cfg.Tasker.BackgroundTaskExpirationSecs, cfg.Tasker.BackgroundTaskExecutionTimeoutSecs)
	if err != nil {
		return "", errors.Annotate(err, "fail to create repair task for %s", botID).Err()
	}
	return taskURL, nil
}

// IsDisjoint returns true if and only if two sequences have no elements in common.
func isDisjoint(a []string, b []string) bool {
	bMap := make(map[string]bool, len(b))
	for _, item := range b {
		bMap[item] = true
	}
	for _, item := range a {
		if bMap[item] {
			return false
		}
	}
	return true
}
