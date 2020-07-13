// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"infra/unifiedfleet/app/model/datastore"
	"net/http"

	"github.com/golang/protobuf/proto"
	authclient "go.chromium.org/luci/auth"
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/grpc/prpc"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"go.chromium.org/luci/server/auth"
	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/cros/sheet"
)

const (
	// https://cloud.google.com/datastore/docs/concepts/limits
	importPageSize int = 500
	// DefaultMachineDBService indicates the default machineDB host for importing
	// TODO(xixuan): rename this variable in the following CL
	DefaultMachineDBService string = "machine-db-dev"
	// ProdMachineDBService indicates the prod machine db service name
	ProdMachineDBService string = "machine-db.appspot.com"
	datacenterConfigFile string = "datacenters.cfg"

	spreadSheetScope string = "https://www.googleapis.com/auth/spreadsheets.readonly"
)

// CfgInterfaceFactory is a contsructor for a luciconfig.Interface
// For potential unittest usage
type CfgInterfaceFactory func(ctx context.Context) luciconfig.Interface

// MachineDBInterfaceFactory is a constructor for a crimson.CrimsonClient
// For potential unittest usage
type MachineDBInterfaceFactory func(ctx context.Context, host string) (crimson.CrimsonClient, error)

// CrosInventoryInterfaceFactory is a constructor for a invV2Api.InventoryClient
type CrosInventoryInterfaceFactory func(ctx context.Context, host string) (CrosInventoryClient, error)

// SheetInterfaceFactory is a constructor for a sheet.ClientInterface
type SheetInterfaceFactory func(ctx context.Context) (sheet.ClientInterface, error)

// FleetServerImpl implements the configuration server interfaces.
type FleetServerImpl struct {
	cfgInterfaceFactory           CfgInterfaceFactory
	machineDBInterfaceFactory     MachineDBInterfaceFactory
	crosInventoryInterfaceFactory CrosInventoryInterfaceFactory
	sheetInterfaceFactory         SheetInterfaceFactory
	importPageSize                int
}

// CrosInventoryClient refers to the fake inventory v2 client
type CrosInventoryClient interface {
	ListCrosDevicesLabConfig(ctx context.Context, in *invV2Api.ListCrosDevicesLabConfigRequest, opts ...grpc.CallOption) (*invV2Api.ListCrosDevicesLabConfigResponse, error)
}

func (cs *FleetServerImpl) newMachineDBInterfaceFactory(ctx context.Context, host string) (crimson.CrimsonClient, error) {
	if cs.machineDBInterfaceFactory != nil {
		return cs.machineDBInterfaceFactory(ctx, host)
	}
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	pclient := &prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
	}
	return crimson.NewCrimsonPRPCClient(pclient), nil
}

func (cs *FleetServerImpl) newCrosInventoryInterfaceFactory(ctx context.Context, host string) (CrosInventoryClient, error) {
	if cs.crosInventoryInterfaceFactory != nil {
		return cs.crosInventoryInterfaceFactory(ctx, host)
	}
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return invV2Api.NewInventoryPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
	}), nil
}

func (cs *FleetServerImpl) newSheetInterface(ctx context.Context) (sheet.ClientInterface, error) {
	if cs.sheetInterfaceFactory != nil {
		return cs.sheetInterfaceFactory(ctx)
	}
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, spreadSheetScope))
	if err != nil {
		return nil, err
	}
	return sheet.NewClient(ctx, &http.Client{Transport: t})
}

func (cs *FleetServerImpl) getImportPageSize() int {
	if cs.importPageSize == 0 {
		return importPageSize
	}
	return cs.importPageSize
}

// Error messages for data import
var (
	machineDBServiceFailure     = "Fail to call machine DB service: %s"
	crosInventoryServiceFailure = "Fail to call Inventory V2 service: %s"

	successStatus                    = status.New(codes.OK, "")
	emptyConfigSourceStatus          = status.New(codes.InvalidArgument, "Invalid argument - Config source is empty")
	invalidConfigServiceName         = status.New(codes.FailedPrecondition, "The config service name is invalid")
	invalidConfigFileContentStatus   = status.New(codes.FailedPrecondition, "The config file format is invalid")
	configServiceFailureStatus       = status.New(codes.Internal, "Fail to get configs from luci config service")
	machineDBConnectionFailureStatus = status.New(codes.Internal, "Fail to initialize connection to machine DB")
	machineDBServiceFailureStatus    = func(service string) *status.Status {
		return status.New(codes.Internal, fmt.Sprintf(machineDBServiceFailure, service))
	}
	crosInventoryConnectionFailureStatus = status.New(codes.Internal, "Fail to initialize connection to Inventory V2")
	crosInventoryServiceFailureStatus    = func(service string) *status.Status {
		return status.New(codes.Internal, fmt.Sprintf(crosInventoryServiceFailure, service))
	}
	sheetConnectionFailureStatus = status.New(codes.Internal, "Fail to initialize connection to Google sheet")
	insertDatastoreFailureStatus = status.New(codes.Internal, "Fail to insert entity into datastore while importing")
)

func processImportDatastoreRes(resp *datastore.OpResults, err error) *status.Status {
	if resp == nil && err == nil {
		return successStatus
	}
	if err != nil {
		s, ok := status.FromError(err)
		if ok {
			return s
		}
	}
	fails := resp.Failed()
	if err == nil && len(fails) == 0 {
		return successStatus
	}
	if len(fails) > 0 {
		details := errorToDetails(fails)
		s, errs := insertDatastoreFailureStatus.WithDetails(details...)
		if errs == nil {
			return s
		}
	}
	return insertDatastoreFailureStatus
}

func errorToDetails(res []*datastore.OpResult) []proto.Message {
	anys := make([]proto.Message, 0)
	for _, r := range res {
		e := &errdetails.ErrorInfo{
			Reason: r.Err.Error(),
		}
		anys = append(anys, e)
	}
	return anys
}
