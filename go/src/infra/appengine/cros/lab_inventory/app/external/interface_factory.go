// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package external

import (
	"context"
	"fmt"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/info"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	ufspb "infra/unifiedfleet/api/v1/models"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

// InterfaceFactoryKey is the key used to store instance of InterfaceFactory in context.
var InterfaceFactoryKey = Key("invV2 external-server-interface key")

// Key is a type for use in adding values to context. It is not recommended to use plain string as key.
type Key string

// UFSInterfaceFactory is a constructor for a api.UFSClient
type UFSInterfaceFactory func(ctx context.Context, host string) (UFSClient, error)

// InterfaceFactory provides a collection of interfaces to external clients.
type InterfaceFactory struct {
	ufsInterfaceFactory UFSInterfaceFactory
}

// UFSClient refers to the fake UFS client
type UFSClient interface {
	GetDutState(ctx context.Context, in *ufsapi.GetDutStateRequest, opts ...grpc.CallOption) (*lab.DutState, error)
	GetMachine(ctx context.Context, in *ufsapi.GetMachineRequest, opts ...grpc.CallOption) (*ufspb.Machine, error)
	GetMachineLSE(ctx context.Context, in *ufsapi.GetMachineLSERequest, opts ...grpc.CallOption) (*ufspb.MachineLSE, error)
	ListMachines(ctx context.Context, in *ufsapi.ListMachinesRequest, opts ...grpc.CallOption) (*ufsapi.ListMachinesResponse, error)
	ListMachineLSEs(ctx context.Context, in *ufsapi.ListMachineLSEsRequest, opts ...grpc.CallOption) (*ufsapi.ListMachineLSEsResponse, error)
	UpdateDutState(ctx context.Context, in *ufsapi.UpdateDutStateRequest, opts ...grpc.CallOption) (*lab.DutState, error)
	ListDutStates(ctx context.Context, in *ufsapi.ListDutStatesRequest, opts ...grpc.CallOption) (*ufsapi.ListDutStatesResponse, error)
	CreateMachineLSE(context.Context, *ufsapi.CreateMachineLSERequest, ...grpc.CallOption) (*ufspb.MachineLSE, error)
	CreateAsset(context.Context, *ufsapi.CreateAssetRequest, ...grpc.CallOption) (*ufspb.Asset, error)
	RackRegistration(context.Context, *ufsapi.RackRegistrationRequest, ...grpc.CallOption) (*ufspb.Rack, error)
	GetAsset(context.Context, *ufsapi.GetAssetRequest, ...grpc.CallOption) (*ufspb.Asset, error)
	GetRack(context.Context, *ufsapi.GetRackRequest, ...grpc.CallOption) (*ufspb.Rack, error)
	UpdateMachineLSE(context.Context, *ufsapi.UpdateMachineLSERequest, ...grpc.CallOption) (*ufspb.MachineLSE, error)
	DeleteMachineLSE(context.Context, *ufsapi.DeleteMachineLSERequest, ...grpc.CallOption) (*emptypb.Empty, error)
}

// GetServerInterface retrieves the ExternalServerInterface from context.
func GetServerInterface(ctx context.Context) (*InterfaceFactory, error) {
	if esif := ctx.Value(InterfaceFactoryKey); esif != nil {
		return esif.(*InterfaceFactory), nil
	}
	return nil, errors.Reason("InterfaceFactory not initialized in context").Err()
}

// WithServerInterface adds the external server interface to context.
func WithServerInterface(ctx context.Context) context.Context {
	return context.WithValue(ctx, InterfaceFactoryKey, &InterfaceFactory{
		ufsInterfaceFactory: ufsInterfaceFactoryImpl,
	})
}

// NewUFSInterfaceFactory creates a new UFSInterface.
func (es *InterfaceFactory) NewUFSInterfaceFactory(ctx context.Context, host string) (UFSClient, error) {
	if es.ufsInterfaceFactory == nil {
		es.ufsInterfaceFactory = ufsInterfaceFactoryImpl
	}
	return es.ufsInterfaceFactory(ctx, host)
}

func ufsInterfaceFactoryImpl(ctx context.Context, host string) (UFSClient, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsCredentialsForwarder)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport to UFS service").Err()
	}
	return ufsapi.NewFleetPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
		Options: &prpc.Options{
			UserAgent: fmt.Sprintf("%s/%s", info.AppID(ctx), "3.0.0"),
		},
	}), nil
}
