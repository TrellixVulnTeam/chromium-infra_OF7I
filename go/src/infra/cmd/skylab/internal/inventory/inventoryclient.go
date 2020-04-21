// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	invV1Api "infra/appengine/crosskylabadmin/api/fleet/v1"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	protos "infra/libs/fleet/protos"
	ufs "infra/libs/fleet/protos/go"
	"infra/libs/skylab/inventory"
)

// Client defines the common interface for the inventory client used by
// various command line tools.
type Client interface {
	GetDutInfo(context.Context, string, bool) (*inventory.DeviceUnderTest, error)
	DeleteDUTs(context.Context, []string, *authcli.Flags, skycmdlib.RemovalReason, io.Writer) (bool, error)
	BatchUpdateDUTs(context.Context, *invV1Api.BatchUpdateDutsRequest, io.Writer) error
}

type inventoryClientV2 struct {
	ic invV2Api.InventoryClient
}

// NewInventoryClient creates a new instance of inventory client.
func NewInventoryClient(hc *http.Client, env site.Environment) Client {
	return &inventoryClientV2{
		ic: invV2Api.NewInventoryPRPCClient(&prpc.Client{
			C:       hc,
			Host:    env.InventoryService,
			Options: site.DefaultPRPCOptions,
		}),
	}
}

func (client *inventoryClientV2) BatchUpdateDUTs(ctx context.Context, req *fleet.BatchUpdateDutsRequest, writer io.Writer) error {
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
func (client *inventoryClientV2) GetDutInfo(ctx context.Context, id string, byHostname bool) (*inventory.DeviceUnderTest, error) {
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

func (client *inventoryClientV2) DeleteDUTs(ctx context.Context, hostnames []string, authFlags *authcli.Flags, rr skycmdlib.RemovalReason, stdout io.Writer) (modified bool, err error) {
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

func (client *inventoryClientV2) updateAssets(ctx context.Context, deletedDevices []*invV2Api.DeviceOpResult, b io.Writer) {
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
		for _, assetResult := range assetResponse.Failed {
			fmt.Fprintf(b, "failed to get asset from registration for %s : %s\n", assetResult.Asset.GetId(), assetResult.GetErrorMsg())
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
