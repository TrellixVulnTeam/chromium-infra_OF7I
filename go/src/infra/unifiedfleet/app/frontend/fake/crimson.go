// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
)

// CrimsonClient mocks the crimsonClient
type CrimsonClient struct {
	MachineNames []string
	Nics         []*crimson.NIC
}

// ListDatacenters mocks the ListDatacenters of crimsonClient
func (c *CrimsonClient) ListDatacenters(ctx context.Context, in *crimson.ListDatacentersRequest, opts ...grpc.CallOption) (*crimson.ListDatacentersResponse, error) {
	return new(crimson.ListDatacentersResponse), nil
}

// ListFreeIPs mocks the ListFreeIPs of crimsonClient
func (c *CrimsonClient) ListFreeIPs(ctx context.Context, in *crimson.ListFreeIPsRequest, opts ...grpc.CallOption) (*crimson.ListIPsResponse, error) {
	return new(crimson.ListIPsResponse), nil
}

// ListKVMs mocks the ListKVMs of crimsonClient
func (c *CrimsonClient) ListKVMs(ctx context.Context, in *crimson.ListKVMsRequest, opts ...grpc.CallOption) (*crimson.ListKVMsResponse, error) {
	out := new(crimson.ListKVMsResponse)
	return out, nil
}

// ListOSes mocks the ListOSes of crimsonClient
func (c *CrimsonClient) ListOSes(ctx context.Context, in *crimson.ListOSesRequest, opts ...grpc.CallOption) (*crimson.ListOSesResponse, error) {
	out := new(crimson.ListOSesResponse)
	return out, nil
}

// ListPlatforms mocks the ListPlatforms of crimsonClient
func (c *CrimsonClient) ListPlatforms(ctx context.Context, in *crimson.ListPlatformsRequest, opts ...grpc.CallOption) (*crimson.ListPlatformsResponse, error) {
	out := new(crimson.ListPlatformsResponse)
	return out, nil
}

// ListRacks mocks the ListRacks of crimsonClient
func (c *CrimsonClient) ListRacks(ctx context.Context, in *crimson.ListRacksRequest, opts ...grpc.CallOption) (*crimson.ListRacksResponse, error) {
	out := new(crimson.ListRacksResponse)
	return out, nil
}

// ListSwitches mocks the ListSwitches of crimsonClient
func (c *CrimsonClient) ListSwitches(ctx context.Context, in *crimson.ListSwitchesRequest, opts ...grpc.CallOption) (*crimson.ListSwitchesResponse, error) {
	out := new(crimson.ListSwitchesResponse)
	return out, nil
}

// ListVLANs mocks the ListVLANs of crimsonClient
func (c *CrimsonClient) ListVLANs(ctx context.Context, in *crimson.ListVLANsRequest, opts ...grpc.CallOption) (*crimson.ListVLANsResponse, error) {
	out := new(crimson.ListVLANsResponse)
	return out, nil
}

// CreateMachine mocks the CreateMachine of crimsonClient
func (c *CrimsonClient) CreateMachine(ctx context.Context, in *crimson.CreateMachineRequest, opts ...grpc.CallOption) (*crimson.Machine, error) {
	out := new(crimson.Machine)
	return out, nil
}

// DeleteMachine mocks the DeleteMachine of crimsonClient
func (c *CrimsonClient) DeleteMachine(ctx context.Context, in *crimson.DeleteMachineRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	return out, nil
}

// ListMachines mocks the ListMachines of crimsonClient
func (c *CrimsonClient) ListMachines(ctx context.Context, in *crimson.ListMachinesRequest, opts ...grpc.CallOption) (*crimson.ListMachinesResponse, error) {
	out := make([]*crimson.Machine, 0)
	for _, n := range c.MachineNames {
		out = append(out, &crimson.Machine{
			Name: n,
		})
	}
	return &crimson.ListMachinesResponse{
		Machines: out,
	}, nil
}

// RenameMachine mocks the RenameMachine of crimsonClient
func (c *CrimsonClient) RenameMachine(ctx context.Context, in *crimson.RenameMachineRequest, opts ...grpc.CallOption) (*crimson.Machine, error) {
	out := new(crimson.Machine)
	return out, nil
}

// UpdateMachine mocks the UpdateMachine of crimsonClient
func (c *CrimsonClient) UpdateMachine(ctx context.Context, in *crimson.UpdateMachineRequest, opts ...grpc.CallOption) (*crimson.Machine, error) {
	out := new(crimson.Machine)
	return out, nil
}

// CreateNIC mocks the CreateNIC of crimsonClient
func (c *CrimsonClient) CreateNIC(ctx context.Context, in *crimson.CreateNICRequest, opts ...grpc.CallOption) (*crimson.NIC, error) {
	out := new(crimson.NIC)
	return out, nil
}

// DeleteNIC mocks the DeleteNIC of crimsonClient
func (c *CrimsonClient) DeleteNIC(ctx context.Context, in *crimson.DeleteNICRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	return out, nil
}

// ListNICs mocks the ListNICs of crimsonClient
func (c *CrimsonClient) ListNICs(ctx context.Context, in *crimson.ListNICsRequest, opts ...grpc.CallOption) (*crimson.ListNICsResponse, error) {
	return &crimson.ListNICsResponse{
		Nics: c.Nics,
	}, nil
}

// UpdateNIC mocks the UpdateNIC of crimsonClient
func (c *CrimsonClient) UpdateNIC(ctx context.Context, in *crimson.UpdateNICRequest, opts ...grpc.CallOption) (*crimson.NIC, error) {
	out := new(crimson.NIC)
	return out, nil
}

// CreateDRAC mocks the CreateDRAC of crimsonClient
func (c *CrimsonClient) CreateDRAC(ctx context.Context, in *crimson.CreateDRACRequest, opts ...grpc.CallOption) (*crimson.DRAC, error) {
	out := new(crimson.DRAC)
	return out, nil
}

// ListDRACs mocks the ListDRACs of crimsonClient
func (c *CrimsonClient) ListDRACs(ctx context.Context, in *crimson.ListDRACsRequest, opts ...grpc.CallOption) (*crimson.ListDRACsResponse, error) {
	out := new(crimson.ListDRACsResponse)
	return out, nil
}

// UpdateDRAC mocks the UpdateDRAC of crimsonClient
func (c *CrimsonClient) UpdateDRAC(ctx context.Context, in *crimson.UpdateDRACRequest, opts ...grpc.CallOption) (*crimson.DRAC, error) {
	out := new(crimson.DRAC)
	return out, nil
}

// CreatePhysicalHost mocks the CreatePhysicalHost of crimsonClient
func (c *CrimsonClient) CreatePhysicalHost(ctx context.Context, in *crimson.CreatePhysicalHostRequest, opts ...grpc.CallOption) (*crimson.PhysicalHost, error) {
	out := new(crimson.PhysicalHost)
	return out, nil
}

// ListPhysicalHosts mocks the ListPhysicalHosts of crimsonClient
func (c *CrimsonClient) ListPhysicalHosts(ctx context.Context, in *crimson.ListPhysicalHostsRequest, opts ...grpc.CallOption) (*crimson.ListPhysicalHostsResponse, error) {
	out := new(crimson.ListPhysicalHostsResponse)
	return out, nil
}

// UpdatePhysicalHost mocks the UpdatePhysicalHost of crimsonClient
func (c *CrimsonClient) UpdatePhysicalHost(ctx context.Context, in *crimson.UpdatePhysicalHostRequest, opts ...grpc.CallOption) (*crimson.PhysicalHost, error) {
	out := new(crimson.PhysicalHost)
	return out, nil
}

// FindVMSlots mocks the FindVMSlots of crimsonClient
func (c *CrimsonClient) FindVMSlots(ctx context.Context, in *crimson.FindVMSlotsRequest, opts ...grpc.CallOption) (*crimson.FindVMSlotsResponse, error) {
	out := new(crimson.FindVMSlotsResponse)
	return out, nil
}

// CreateVM mocks the CreateVM of crimsonClient
func (c *CrimsonClient) CreateVM(ctx context.Context, in *crimson.CreateVMRequest, opts ...grpc.CallOption) (*crimson.VM, error) {
	out := new(crimson.VM)
	return out, nil
}

// ListVMs mocks the ListVMs of crimsonClient
func (c *CrimsonClient) ListVMs(ctx context.Context, in *crimson.ListVMsRequest, opts ...grpc.CallOption) (*crimson.ListVMsResponse, error) {
	out := new(crimson.ListVMsResponse)
	return out, nil
}

// UpdateVM mocks the UpdateVM of crimsonClient
func (c *CrimsonClient) UpdateVM(ctx context.Context, in *crimson.UpdateVMRequest, opts ...grpc.CallOption) (*crimson.VM, error) {
	out := new(crimson.VM)
	return out, nil
}

// DeleteHost mocks the DeleteHost of crimsonClient
func (c *CrimsonClient) DeleteHost(ctx context.Context, in *crimson.DeleteHostRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	return out, nil
}
