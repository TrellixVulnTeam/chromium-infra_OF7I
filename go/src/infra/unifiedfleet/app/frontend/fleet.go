// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/unifiedfleet/app/model/datastore"
)

const (
	// https://cloud.google.com/datastore/docs/concepts/limits
	importPageSize int = 500
	// DefaultMachineDBService indicates the default machineDB host for importing
	// TODO(xixuan): rename this variable in the following CL
	DefaultMachineDBService string = "machine-db"
	// ProdMachineDBService indicates the prod machine db service name
	ProdMachineDBService string = "machine-db.appspot.com"
	datacenterConfigFile string = "datacenters.cfg"
)

// FleetServerImpl implements the configuration server interfaces.
type FleetServerImpl struct {
	importPageSize int
}

func (fs *FleetServerImpl) getImportPageSize() int {
	if fs.importPageSize == 0 {
		return importPageSize
	}
	return fs.importPageSize
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
	gitConnectionFailureStatus   = status.New(codes.Internal, "Fail to initialize connection to Gitiles")
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
