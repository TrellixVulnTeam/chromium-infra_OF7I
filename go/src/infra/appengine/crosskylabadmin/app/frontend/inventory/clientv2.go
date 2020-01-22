// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/skylab/inventory"
)

type invServiceClient struct {
	client api.InventoryClient
}

func newInvServiceClient(ctx context.Context, host string) (inventoryClient, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsCredentialsForwarder)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport to inventory service").Err()
	}
	ic := api.NewInventoryPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
	})

	return &invServiceClient{client: ic}, nil
}

func (c *invServiceClient) logInfo(ctx context.Context, t string, s ...interface{}) {
	logging.Infof(ctx, fmt.Sprintf("[InventoryV2Client]: %s", t), s...)
}

func (c *invServiceClient) addManyDUTsToFleet(ctx context.Context, nds []*inventory.CommonDeviceSpecs, pickServoPort bool) (string, []*inventory.CommonDeviceSpecs, error) {
	// In case there's any panic happens in the new code.
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf(ctx, "Recovered in addManyDUTsToFleet(%s)", r)
			debug.PrintStack()
		}
	}()

	c.logInfo(ctx, "Access inventory service as user: %s", auth.CurrentUser(ctx))

	devicesToAdd, labstations, _, err := api.ImportFromV1DutSpecs(nds)
	if err != nil {
		logging.Errorf(ctx, "failed to import DUT specs: %s", err)
		return "", nil, err
	}
	if len(devicesToAdd) == 0 {
		devicesToAdd = labstations
	}

	var rsp *api.AddCrosDevicesResponse
	f := func() error {
		rsp, err = c.client.AddCrosDevices(ctx, &api.AddCrosDevicesRequest{
			Devices:       devicesToAdd,
			PickServoPort: pickServoPort,
		})
		return err
	}
	err = retry.Retry(ctx, transientErrorRetries(), f, retry.LogCallback(ctx, "addManyDUTsToFleet v2"))
	if err != nil {
		return "", nil, err
	}

	passedDevSpecs := getPassedDevSpecs(nds, rsp.GetPassedDevices())
	failedDevices := rsp.GetFailedDevices()
	msgs := make([]string, 0, len(failedDevices))
	for _, d := range failedDevices {
		msgs = append(msgs, d.GetErrorMsg())
	}
	if len(msgs) > 0 {
		err = errors.Reason(strings.Join(msgs, ",")).Err()
	}
	return "URL N/A", passedDevSpecs, err
}

func getPassedDevSpecs(allDevSpecs []*inventory.CommonDeviceSpecs, passedDevices []*api.DeviceOpResult) []*inventory.CommonDeviceSpecs {
	// The response only has hostname/id, so we need to get other data from the
	// input.
	nameToSpec := map[string]*inventory.CommonDeviceSpecs{}
	passedDevSpecs := make([]*inventory.CommonDeviceSpecs, 0, len(allDevSpecs))
	for i := range allDevSpecs {
		nameToSpec[allDevSpecs[i].GetHostname()] = allDevSpecs[i]
	}

	for _, d := range passedDevices {
		pd := nameToSpec[d.GetHostname()]
		id := d.GetId()
		pd.Id = &id // The ID may be newly assigned by inventory service.
		passedDevSpecs = append(passedDevSpecs, pd)
	}
	return passedDevSpecs
}

func (c *invServiceClient) updateDUTSpecs(ctx context.Context, od, nd *inventory.CommonDeviceSpecs, pickServoPort bool) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf(ctx, "Recovered in updateDUTSpecs(%s)", r)
		}
	}()

	devicesToUpdate, labstations, _, err := api.ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{nd})
	if err != nil {
		logging.Errorf(ctx, "failed to import DUT specs: %s", err)
		return "", err
	}
	if len(devicesToUpdate) == 0 {
		devicesToUpdate = labstations
	}

	f := func() error {
		if rsp, err := c.client.UpdateCrosDevicesSetup(ctx, &api.UpdateCrosDevicesSetupRequest{
			Devices: devicesToUpdate,
			// TODO (guocb) Add reason why update.
		}); err != nil {
			return err
		} else if len(rsp.FailedDevices) > 0 {
			// There's only one device under updating.
			return errors.Reason(rsp.FailedDevices[0].ErrorMsg).Err()
		}
		return nil
	}
	err = retry.Retry(ctx, transientErrorRetries(), f, retry.LogCallback(ctx, "updateDUTSpecs v2"))
	if err != nil {
		return "", errors.Annotate(err, "update DUT spects").Err()
	}

	return "URL N/A", nil
}

func (c *invServiceClient) deleteDUTsFromFleet(ctx context.Context, ids []string) (string, []string, error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf(ctx, "Recovered in deleteDUTsFromFleet(%s)", r)
		}
	}()

	return "fake delete URL", []string{}, nil
}
