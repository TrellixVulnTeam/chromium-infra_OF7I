// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventoryclient

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/status"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	invV1Api "infra/appengine/crosskylabadmin/api/fleet/v1"
	protos "infra/libs/fleet/protos"
	ufs "infra/libs/fleet/protos/go"
	"infra/libs/skylab/inventory"
	rem "infra/libs/skylab/inventory/removalreason"
)

// Client defines the common interface for the inventory client used by
// various command line tools.
type Client interface {
	GetDutInfo(context.Context, string, bool) (*inventory.DeviceUnderTest, error)
	DeleteDUTs(context.Context, []string, *authcli.Flags, rem.RemovalReason, io.Writer) (bool, error)
	BatchUpdateDUTs(context.Context, *invV1Api.BatchUpdateDutsRequest, io.Writer) error
	FilterDUTHostnames(context.Context, []string) ([]string, error)
	UpdateLabstations(context.Context, string, string, string) (*invV2Api.UpdateLabstationsResponse, error)
	UpdateDUT(context.Context, *inventory.CommonDeviceSpecs) error
}

// V2Client is an API client for the inventory V2 service.
type V2Client struct {
	ic invV2Api.InventoryClient
}

// NewInventoryClient creates a new instance of inventory client.
func NewInventoryClient(hc *http.Client,
	inventoryService string,
	options *prpc.Options,
) *V2Client {
	return &V2Client{
		ic: invV2Api.NewInventoryPRPCClient(&prpc.Client{
			C:       hc,
			Host:    inventoryService,
			Options: options,
		}),
	}
}

// UpdateDUT takes the device specifications for a DUT and updates its entry in the inventory.
func (client *V2Client) UpdateDUT(ctx context.Context, newSpecs *inventory.CommonDeviceSpecs) error {
	// Copy from https://chromium.git.corp.google.com/infra/infra/+/d0b7fa7d180b2fa273ddd93cf6e6183b65c3b32a/go/src/infra/appengine/crosskylabadmin/app/frontend/inventory/clientv2.go#145
	devicesToUpdate, labstations, _, err := invV2Api.ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{newSpecs})
	if err != nil {
		return errors.Annotate(err, "convert DUT spec").Err()
	}
	if len(devicesToUpdate) == 0 {
		devicesToUpdate = labstations
	}

	f := func() error {
		if rsp, err := client.ic.UpdateCrosDevicesSetup(ctx, &invV2Api.UpdateCrosDevicesSetupRequest{
			Devices:       devicesToUpdate,
			PickServoPort: true,
		}); err != nil {
			return err
		} else if len(rsp.FailedDevices) > 0 {
			// There's only one device under updating.
			return errors.Reason(rsp.FailedDevices[0].ErrorMsg).Err()
		}
		return nil
	}
	err = retry.Retry(ctx, transientErrorRetries(), f, retry.LogCallback(ctx, "UpdateDUT (v2)"))
	if err != nil {
		if er, ok := status.FromError(err); ok {
			return errors.Reason("update setup configs: " + er.Message()).Err()
		}
		return errors.Annotate(err, "update setup configs").Err()
	}

	return nil
}

// UpdateLabstations is similar to UpdateDUT but updates a labstation instead.
// Since labstations manage devices like servos and DUTs, updating a labstation potentially
// involves modifying multiple tracked by the inventory in a way that can't be done as a sequence
// of individual steps without breaking invariants.
func (client *V2Client) UpdateLabstations(ctx context.Context, hostname, servosToDelete, dutToAdd string) (*invV2Api.UpdateLabstationsResponse, error) {
	req := &invV2Api.UpdateLabstationsRequest{
		Hostname: hostname,
	}
	if servosToDelete != "" {
		req.DeletedServos = []string{servosToDelete}
	}
	if dutToAdd != "" {
		req.AddedDUTs = []string{dutToAdd}
	}
	return client.ic.UpdateLabstations(ctx, req)
}

// BatchUpdateDUTs updates many DUTs.
func (client *V2Client) BatchUpdateDUTs(ctx context.Context, req *fleet.BatchUpdateDutsRequest, writer io.Writer) error {
	properties := make([]*invV2Api.DeviceProperty, len(req.GetDutProperties()))
	for i, r := range req.GetDutProperties() {
		properties[i] = &invV2Api.DeviceProperty{
			Hostname: r.GetHostname(),
			Pool:     r.GetPool(),
			Rpm: &invV2Api.DeviceProperty_Rpm{
				PowerunitName:   r.GetRpm().GetPowerunitHostname(),
				PowerunitOutlet: r.GetRpm().GetPowerunitOutlet(),
			},
		}
	}
	_, err := client.ic.BatchUpdateDevices(ctx, &invV2Api.BatchUpdateDevicesRequest{
		DeviceProperties: properties,
	})
	if err != nil {
		return errors.Annotate(err, "fail to update Duts").Err()
	}
	fmt.Fprintln(writer, "Successfully batch updated.")
	return nil
}

// GetDutInfo gets the dut information from inventory v2 service.
func (client *V2Client) GetDutInfo(ctx context.Context, id string, byHostname bool) (*inventory.DeviceUnderTest, error) {
	devID := &invV2Api.DeviceID{Id: &invV2Api.DeviceID_ChromeosDeviceId{ChromeosDeviceId: id}}
	if byHostname {
		devID = &invV2Api.DeviceID{Id: &invV2Api.DeviceID_Hostname{Hostname: id}}
	}
	rsp, err := client.ic.GetCrosDevices(ctx, &invV2Api.GetCrosDevicesRequest{
		Ids: []*invV2Api.DeviceID{devID},
	})
	if err != nil {
		return nil, errors.Annotate(err, "get dutinfo for %s", id).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		result := rsp.FailedDevices[0]
		return nil, errors.Reason("failed to get device %s: %s", result.Hostname, result.ErrorMsg).Err()
	}
	if len(rsp.Data) != 1 {
		return nil, errors.Reason("no info returned for %s", id).Err()
	}
	return invV2Api.AdaptToV1DutSpec(rsp.Data[0])
}

// DeleteDUTs deletes DUTs from the inventory and tracks the reason for the removal.
func (client *V2Client) DeleteDUTs(ctx context.Context, hostnames []string, authFlags *authcli.Flags, rr rem.RemovalReason, stdout io.Writer) (modified bool, err error) {
	var devIds []*invV2Api.DeviceID
	for _, h := range hostnames {
		devIds = append(devIds, &invV2Api.DeviceID{Id: &invV2Api.DeviceID_Hostname{Hostname: h}})
	}
	// RemovalReason is to be added into DeleteCrosDevicesRequest.
	rsp, err := client.ic.DeleteCrosDevices(ctx, &invV2Api.DeleteCrosDevicesRequest{
		Ids: devIds,
		Reason: &invV2Api.DeleteCrosDevicesRequest_Reason{
			Bug:     rr.Bug,
			Comment: rr.Comment,
		},
	})
	if err != nil {
		return false, errors.Annotate(err, "remove devices for %s ...", hostnames[0]).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		var reasons []string
		for _, d := range rsp.FailedDevices {
			reasons = append(reasons, fmt.Sprintf("%s:%s", d.Hostname, d.ErrorMsg))
		}
		return false, errors.Reason("failed to remove device: %s", strings.Join(reasons, ", ")).Err()
	}
	b := bufio.NewWriter(stdout)
	fmt.Fprintln(b, "Deleted DUT hostnames")
	for _, d := range rsp.RemovedDevices {
		fmt.Fprintln(b, d.Hostname)
	}
	// TODO(eshwarn) : move this into DeleteCrosDevices in inventoryV2 layer
	client.updateAssets(ctx, rsp.RemovedDevices, b)
	b.Flush()
	return len(rsp.RemovedDevices) > 0, nil
}

func (client *V2Client) updateAssets(ctx context.Context, deletedDevices []*invV2Api.DeviceOpResult, b io.Writer) {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
		}
	}()
	if len(deletedDevices) < 0 {
		return
	}
	var existingAssetsIDs = make([]string, 0, len(deletedDevices))
	var existingAssets = make([]*protos.ChopsAsset, 0, len(deletedDevices))
	for _, deletedDevice := range deletedDevices {
		existingAssetsIDs = append(existingAssetsIDs, deletedDevice.GetId())
		existingAssets = append(existingAssets,
			&protos.ChopsAsset{
				Id:       deletedDevice.GetId(),
				Location: &ufs.Location{},
			})
	}
	assetResponse, _ := client.ic.GetAssets(ctx, &invV2Api.AssetIDList{Id: existingAssetsIDs})
	if assetResponse != nil {
		for _, assetResult := range assetResponse.Passed {
			fmt.Fprintf(b, "AssetId: %s , Old Location: %s\n", assetResult.GetAsset().GetId(), assetResult.GetAsset().GetLocation().String())
		}
	}
	// Update existing assets in registration system
	assetResponse, _ = client.ic.UpdateAssets(ctx, &invV2Api.AssetList{Asset: existingAssets})
	if assetResponse != nil {
		for _, assetResult := range assetResponse.Passed {
			fmt.Fprintf(b, "AssetId: %s , New Location: %s\n", assetResult.GetAsset().GetId(), assetResult.GetAsset().GetLocation().String())
		}
	}
}

// FilterDUTHostnames produces a list of only the DUT hostnames that exist.
func (client *V2Client) FilterDUTHostnames(ctx context.Context, hostnames []string) ([]string, error) {
	var out []string
	// The RPC will fail if no hostnames are provided, so return early instead.
	if len(hostnames) == 0 {
		return out, nil
	}
	req := &invV2Api.GetCrosDevicesRequest{}
	for _, hostname := range hostnames {
		req.Ids = append(req.Ids, &invV2Api.DeviceID{Id: &invV2Api.DeviceID_Hostname{Hostname: hostname}})
	}
	rsp, err := client.ic.GetCrosDevices(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get DUT information").Err()
	}
	for _, item := range rsp.Data {
		hostname := item.GetLabConfig().GetDut().GetHostname()
		out = append(out, hostname)

	}
	return out, nil
}

// Set up the client-side retry strategy for inventory APIs.
// Slow down the retry to not flood the external APIs.
var transientErrorRetriesTemplate = retry.ExponentialBackoff{
	Limited: retry.Limited{
		Delay:   200 * time.Millisecond,
		Retries: 3,
	},
	Multiplier: 4,
	MaxDelay:   5 * time.Second,
}

// transientErrorRetries returns a retry.Factory to use on transient errors on
// outbound requests.
func transientErrorRetries() retry.Factory {
	next := func() retry.Iterator {
		it := transientErrorRetriesTemplate
		return &it
	}
	return transient.Only(next)
}
