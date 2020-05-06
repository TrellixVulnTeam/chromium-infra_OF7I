// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"
	code "google.golang.org/genproto/googleapis/rpc/code"
	status "google.golang.org/genproto/googleapis/rpc/status"

	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/grpc/prpc"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"go.chromium.org/luci/server/auth"
)

const (
	machineCollection string = "machines"
	rackCollection    string = "racks"
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

// Error messages for data import
var (
	successStatus = &status.Status{
		Code: int32(code.Code_OK),
	}
	emptyConfigSource       = "Invalid argument - Config source is empty"
	emptyConfigSourceStatus = &status.Status{
		Code:    int32(code.Code_INVALID_ARGUMENT),
		Message: emptyConfigSource,
	}
	emptyMachineDBSource       = "Invalid argument - MachineDB source is empty"
	emptyMachineDBSourceStatus = &status.Status{
		Code:    int32(code.Code_INVALID_ARGUMENT),
		Message: emptyMachineDBSource,
	}
	invalidHostInMachineDBSource       = "Invalid argument - Host in MachineDB source is empty/invalid"
	invalidHostInMachineDBSourceStatus = &status.Status{
		Code:    int32(code.Code_INVALID_ARGUMENT),
		Message: invalidHostInMachineDBSource,
	}
	invalidConfigFileContent       = "The config file format is invalid"
	invalidConfigFileContentStatus = &status.Status{
		Code:    int32(code.Code_FAILED_PRECONDITION),
		Message: invalidConfigFileContent,
	}
	configServiceFailure       = "Fail to get configs from luci config service"
	configServiceFailureStatus = &status.Status{
		Code:    int32(code.Code_INTERNAL),
		Message: configServiceFailure,
	}
	machineDBConnectionFailure       = "Fail to initialize connection to machine DB"
	machineDBConnectionFailureStatus = &status.Status{
		Code:    int32(code.Code_INTERNAL),
		Message: machineDBConnectionFailure,
	}
	machineDBServiceFailure       = "Fail to call machine DB service: %s"
	machineDBServiceFailureStatus = func(service string) *status.Status {
		return &status.Status{
			Code:    int32(code.Code_INTERNAL),
			Message: fmt.Sprintf(machineDBServiceFailure, service),
		}
	}
	insertDatastoreFailure       = "Fail to insert chrome platforms into datastore in importing"
	insertDatastoreFailureStatus = &status.Status{
		Code:    int32(code.Code_INTERNAL),
		Message: insertDatastoreFailure,
	}
)
