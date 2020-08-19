// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package dutstate provides representation of states of DUT in Swarming
// and reading and updating a state in UFS service.
package dutstate

import (
	"context"
	"log"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufsProto "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// State is an enum for host state.
type State string

// All DUT states.
const (
	Ready        State = "ready"
	NeedsRepair  State = "needs_repair"
	NeedsReset   State = "needs_reset"
	RepairFailed State = "repair_failed"
	// TODO(xixuan): https://bugs.chromium.org/p/chromium/issues/detail?id=1025040#c19
	// This needs_deploy state may be lost and get changed to needs_repair when the
	// local state file of each bot on drone gets wiped, which usually happens when bots
	// get restarted. Drone container image upgrade, drone server memory overflow, or
	// drone server restart can cause the swarming bots to restart.
	NeedsDeploy State = "needs_deploy"
	// Device reserved for analysis or hold by lab
	Reserved State = "reserved"
	// Device under manual repair interaction by lab
	ManualRepair State = "manual_repair"
	// Device required manual attention to be fixed
	NeedsManualRepair State = "needs_manual_repair"
	// Device is not fixable due issues with hardware and has to be replaced
	NeedsReplacement State = "needs_replacement"
)

const defaultState = NeedsRepair

// Info represent information of the state and last updated time.
type Info struct {
	// State represents the state of the DUT from Swarming.
	State State
	// Time represents in Unix time of the last updated DUT state recorded.
	Time int64
}

// UFSClient represents short set of method of ufsAPI.FleetClient.
type UFSClient interface {
	GetState(ctx context.Context, req *ufsAPI.GetStateRequest, opts ...grpc.CallOption) (*ufsProto.StateRecord, error)
	UpdateState(ctx context.Context, req *ufsAPI.UpdateStateRequest, opts ...grpc.CallOption) (*ufsProto.StateRecord, error)
}

// Read read state from UFS.
//
// If state not exist in the UFS the state will be default and time is 0.
func Read(ctx context.Context, c UFSClient, host string) Info {
	resourceName := makeUFSResourceName(host)

	log.Printf("dutstate: Try to read state for %s", host)
	res, err := c.GetState(ctx, &ufsAPI.GetStateRequest{
		ResourceName: resourceName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Printf("dutstate: State not initialized for %s; %s", host, err)
		} else {
			log.Printf("dutstate: Fail to read state for %s; %s", host, err)
		}
		// For default state time will not set and equal 0.
		return Info{
			State: defaultState,
		}
	}
	return Info{
		State: convertFromUFSState(res.GetState()),
		Time:  res.GetUpdateTime().Seconds,
	}
}

// Update push new DUT state to UFS.
func Update(ctx context.Context, c UFSClient, host string, state State) error {
	ufsState := convertToUFSState(state)
	resourceName := makeUFSResourceName(host)

	log.Printf("dutstate: Try to update state %s: %q (%q)", host, state, ufsState)
	_, err := c.UpdateState(ctx, &ufsAPI.UpdateStateRequest{
		State: &ufsProto.StateRecord{
			ResourceName: resourceName,
			State:        ufsState,
		},
	})
	if err != nil {
		return errors.Annotate(err, "set state %q for %q in UFS", state, host).Err()
	}
	return nil
}

func convertToUFSState(state State) ufsProto.State {
	if ufsState, ok := stateToUFS[state]; ok {
		return ufsState
	}
	return ufsProto.State_STATE_UNSPECIFIED
}

func convertFromUFSState(state ufsProto.State) State {
	if s, ok := stateFromUFS[state]; ok {
		return s
	}
	return defaultState
}

func makeUFSResourceName(host string) string {
	return ufsUtil.AddPrefix(ufsUtil.HostCollection, host)
}

var stateToUFS = map[State]ufsProto.State{
	Ready:             ufsProto.State_STATE_SERVING,
	NeedsReset:        ufsProto.State_STATE_NEEDS_RESET,
	NeedsRepair:       ufsProto.State_STATE_NEEDS_REPAIR,
	RepairFailed:      ufsProto.State_STATE_REPAIR_FAILED,
	NeedsDeploy:       ufsProto.State_STATE_DEPLOYED_PRE_SERVING,
	Reserved:          ufsProto.State_STATE_RESERVED,
	ManualRepair:      ufsProto.State_STATE_DEPLOYED_TESTING,
	NeedsManualRepair: ufsProto.State_STATE_DISABLED,
	NeedsReplacement:  ufsProto.State_STATE_DECOMMISSIONED,
}

var stateFromUFS = map[ufsProto.State]State{
	ufsProto.State_STATE_SERVING:              Ready,
	ufsProto.State_STATE_NEEDS_RESET:          NeedsReset,
	ufsProto.State_STATE_NEEDS_REPAIR:         NeedsRepair,
	ufsProto.State_STATE_REPAIR_FAILED:        RepairFailed,
	ufsProto.State_STATE_DEPLOYED_PRE_SERVING: NeedsDeploy,
	ufsProto.State_STATE_RESERVED:             Reserved,
	ufsProto.State_STATE_DEPLOYED_TESTING:     ManualRepair,
	ufsProto.State_STATE_DISABLED:             NeedsManualRepair,
	ufsProto.State_STATE_DECOMMISSIONED:       NeedsReplacement,
}
