// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"infra/unifiedfleet/app/model/datastore"
	"net/http"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/grpc/prpc"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"go.chromium.org/luci/server/auth"
)

const (
	machineCollection        string = "machines"
	rackCollection           string = "racks"
	chromePlatformCollection string = "chromeplatforms"
	machineLSECollection     string = "machineLSEs"
	rackLSECollection        string = "rackLSEs"
	// https://cloud.google.com/datastore/docs/concepts/limits
	importPageSize int = 500
)

// CfgInterfaceFactory is a contsructor for a luciconfig.Interface
// For potential unittest usage
type CfgInterfaceFactory func(ctx context.Context) luciconfig.Interface

// MachineDBInterfaceFactory is a constructor for a crimson.CrimsonClient
// For potential unittest usage
type MachineDBInterfaceFactory func(ctx context.Context, host string) (crimson.CrimsonClient, error)

// FleetServerImpl implements the configuration server interfaces.
type FleetServerImpl struct {
	cfgInterfaceFactory       CfgInterfaceFactory
	machineDBInterfaceFactory MachineDBInterfaceFactory
	importPageSize            int
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

func (cs *FleetServerImpl) getImportPageSize() int {
	if cs.importPageSize == 0 {
		return importPageSize
	}
	return cs.importPageSize
}

// Error messages for data import
var (
	machineDBServiceFailure = "Fail to call machine DB service: %s"

	successStatus                    = status.New(codes.OK, "")
	emptyConfigSourceStatus          = status.New(codes.InvalidArgument, "Invalid argument - Config source is empty")
	invalidConfigFileContentStatus   = status.New(codes.FailedPrecondition, "The config file format is invalid")
	configServiceFailureStatus       = status.New(codes.Internal, "Fail to get configs from luci config service")
	machineDBConnectionFailureStatus = status.New(codes.Internal, "Fail to initialize connection to machine DB")
	machineDBServiceFailureStatus    = func(service string) *status.Status {
		return status.New(codes.Internal, fmt.Sprintf(machineDBServiceFailure, service))
	}
	insertDatastoreFailureStatus = status.New(codes.Internal, "Fail to insert chrome platforms into datastore in importing")
)

func processImportDatastoreRes(resp *datastore.OpResults, err error) *status.Status {
	fails := resp.Failed()
	if err == nil && len(fails) == 0 {
		return successStatus
	}
	if err != nil {
		s, ok := status.FromError(err)
		if ok {
			return s
		}
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

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
