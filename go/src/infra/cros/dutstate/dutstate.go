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
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	ufsProto "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// State is an enum for host state.
type State string

// All DUT states.
const (
	// Device ready to run tests.
	Ready State = "ready"
	// Provisioning failed on the device and required verified and repair.
	NeedsRepair State = "needs_repair"
	// Test failed on the device and required reset the state.
	NeedsReset State = "needs_reset"
	// Device did not recovered after running repair task on it.
	RepairFailed State = "repair_failed"
	// Device prepared to be deployed to the lab.
	NeedsDeploy State = "needs_deploy"
	// Device reserved for analysis or hold by lab
	Reserved State = "reserved"
	// Device under manual repair interaction by lab
	ManualRepair State = "manual_repair"
	// Device required manual attention to be fixed
	NeedsManualRepair State = "needs_manual_repair"
	// Device is not fixable due issues with hardware and has to be replaced
	NeedsReplacement State = "needs_replacement"
	// Device state when state is not present or cannot be read from UFS.
	Unknown State = "unknown"
)

// Info represent information of the state and last updated time.
type Info struct {
	// State represents the state of the DUT from Swarming.
	State State
	// Time represents in Unix time of the last updated DUT state recorded.
	Time int64
}

// UFSClient represents short set of method of ufsAPI.FleetClient.
type UFSClient interface {
	GetMachineLSE(ctx context.Context, req *ufsAPI.GetMachineLSERequest, opts ...grpc.CallOption) (*ufsProto.MachineLSE, error)
	UpdateMachineLSE(ctx context.Context, req *ufsAPI.UpdateMachineLSERequest, opts ...grpc.CallOption) (*ufsProto.MachineLSE, error)
}

// String provides string representation of the DUT state.
func (s State) String() string {
	return string(s)
}

// Read read state from UFS.
//
// If state not exist in the UFS the state will be default and time is 0.
func Read(ctx context.Context, c UFSClient, host string) Info {
	ctx = setupContext(ctx, ufsUtil.OSNamespace)
	log.Printf("dutstate: Try to read DUT/Labstation state for %s", host)
	res, err := c.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, host),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Printf("dutstate: DUT/Labstation not found for %s; %s", host, err)
		} else {
			log.Printf("dutstate: Fail to get DUT/Labstation for %s; %s", host, err)
		}
		// For default state time will not set and equal 0.
		return Info{
			State: Unknown,
		}
	}
	return Info{
		State: convertFromUFSState(res.GetResourceState()),
		Time:  res.GetUpdateTime().Seconds,
	}
}

// Update push new DUT/Labstation state to UFS.
func Update(ctx context.Context, c UFSClient, host string, state State) error {
	ctx = setupContext(ctx, ufsUtil.OSNamespace)
	ufsState := convertToUFSState(state)

	// Get the MachineLSE to determine if its a DUT or a Labstation.
	log.Printf("dutstate: Try to get MachineLSE for %s", host)
	res, err := c.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, host),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get DUT/Labstation for %s", host).Err()
	}

	log.Printf("dutstate: Try to update DUT/Labstation state %s: %q (%q)", host, state, ufsState)
	res.ResourceState = ufsState
	_, err = c.UpdateMachineLSE(ctx, &ufsAPI.UpdateMachineLSERequest{
		MachineLSE: res,
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"resourceState"},
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
	return Unknown
}

// setupContext sets up context with namespace
func setupContext(ctx context.Context, namespace string) context.Context {
	md := metadata.Pairs(ufsUtil.Namespace, namespace)
	return metadata.NewOutgoingContext(ctx, md)
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
