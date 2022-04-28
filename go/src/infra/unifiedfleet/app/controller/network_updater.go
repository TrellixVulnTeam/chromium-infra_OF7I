// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

type networkUpdater struct {
	Hostname string
	Changes  []*ufspb.ChangeEvent
	Msgs     []*history.SnapshotMsgEntity
}

func (nu *networkUpdater) logChanges(changes []*ufspb.ChangeEvent, msg *history.SnapshotMsgEntity) {
	nu.Changes = append(nu.Changes, changes...)
	if msg != nil {
		nu.Msgs = append(nu.Msgs, msg)
	}
}

// deleteDHCPHelper deletes ip configs for a given hostname
//
// Can be used in a transaction
func (nu *networkUpdater) deleteDHCPHelper(ctx context.Context) error {
	dhcp, err := configuration.GetDHCPConfig(ctx, nu.Hostname)
	if util.IsInternalError(err) {
		return errors.Annotate(err, "Fail to query dhcp for host %s", nu.Hostname).Tag(grpcutil.FailedPreconditionTag).Err()
	}
	if err == nil && dhcp != nil {
		if err := nu.deleteHostHelper(ctx, dhcp); err != nil {
			return err
		}
	}
	return nil
}

// Delete all ip-related configs
//
// Can be used in a transaction
func (nu *networkUpdater) deleteHostHelper(ctx context.Context, dhcp *ufspb.DHCPConfig) error {
	logging.Debugf(ctx, "Found existing dhcp configs for host %s", dhcp.GetHostname())
	logging.Debugf(ctx, "Deleting dhcp %s (%s)", dhcp.GetHostname(), dhcp.GetIp())
	if err := configuration.DeleteDHCP(ctx, dhcp.GetHostname()); err != nil {
		return errors.Annotate(err, "deleteHostHelper - Fail to delete dhcp for hostname %q", dhcp.GetHostname()).Tag(grpcutil.FailedPreconditionTag).Err()
	}
	nu.logChanges(LogDHCPChanges(dhcp, nil))
	ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
	if err != nil {
		return errors.Annotate(err, "deleteHostHelper - Fail to query ip by ipv4 str: %q", dhcp.GetIp()).Tag(grpcutil.FailedPreconditionTag).Err()
	}
	if ips == nil {
		return nil
	}
	oldIP := proto.Clone(ips[0]).(*ufspb.IP)
	ips[0].Occupied = false
	logging.Debugf(ctx, "Update ip %s to non-occupied", ips[0].GetIpv4Str())
	if _, err := configuration.BatchUpdateIPs(ctx, ips); err != nil {
		return errors.Annotate(err, "deleteHostHelper - Fail to update ip: %q (ipv4: %q, vlan %q)", ips[0].GetId(), ips[0].GetIpv4Str(), ips[0].GetVlan()).Tag(grpcutil.FailedPreconditionTag).Err()
	}
	nu.Changes = append(nu.Changes, LogIPChanges(oldIP, ips[0])...)
	return nil
}

func getFreeIPHelper(ctx context.Context, vlanName string) (*ufspb.IP, error) {
	ips, err := getFreeIP(ctx, vlanName, 1)
	if err != nil {
		return nil, errors.Annotate(err, "GetFreeIP").Err()
	}
	if ips[0].GetIpv4Str() == "" {
		return nil, fmt.Errorf("found invalid ip %q (ipv4 %q) in vlan %s", ips[0].GetId(), ips[0].GetIpv4(), vlanName)
	}
	logging.Debugf(ctx, "Get free ip %s", ips[0].GetIpv4Str())
	return ips[0], nil
}

func getSpecifiedIP(ctx context.Context, ipv4Str string) (*ufspb.IP, error) {
	ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{
		"ipv4_str": ipv4Str,
	})
	if err != nil {
		return nil, errors.Annotate(err, "Fail to query ip entity by %s", ipv4Str).Err()
	}
	if len(ips) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "IP %s does not exist", ipv4Str)
	}
	if len(ips) > 1 {
		var vlans []string
		for _, ip := range ips {
			vlans = append(vlans, ip.GetVlan())
		}
		return nil, status.Errorf(codes.InvalidArgument, "IP %s exists in multiple vlans: %v", ipv4Str, vlans)
	}
	if ips[0].GetOccupied() {
		dhcps, err := configuration.QueryDHCPConfigByPropertyName(ctx, "ipv4", ipv4Str)
		if err != nil {
			return nil, errors.Annotate(err, "IP %s is occupied, but fail to query the corresponding dhcp", ipv4Str).Err()
		}
		if dhcps != nil {
			return nil, status.Errorf(codes.InvalidArgument, "IP %s is occupied by host %s", ipv4Str, dhcps[0].GetHostname())
		}
	}
	if ips[0].GetReserve() {
		logging.Debugf(ctx, "User %s trying to use reserved IP %s", auth.CurrentUser(ctx).Email, ips[0].GetIpv4Str())
	}
	return ips[0], nil
}

// Update a dhcp change
//
// Can be used in a transaction
func (nu *networkUpdater) updateDHCPWithMac(ctx context.Context, macAddress string) (*ufspb.DHCPConfig, error) {
	oldDhcp, _ := configuration.GetDHCPConfig(ctx, nu.Hostname)
	dhcp := proto.Clone(oldDhcp).(*ufspb.DHCPConfig)
	if oldDhcp != nil && oldDhcp.GetMacAddress() != macAddress {
		dhcp.MacAddress = macAddress
		if _, err := configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{dhcp}); err != nil {
			return nil, errors.Annotate(err, "updateDHCPWithMac - Failed to update dhcp for host %s (mac %s)", nu.Hostname, macAddress).Tag(grpcutil.FailedPreconditionTag).Err()
		}
		nu.logChanges(LogDHCPChanges(oldDhcp, dhcp))
	}
	return dhcp, nil
}

// Rename a dhcp change
//
// Can be used in a transaction
func (nu *networkUpdater) renameDHCP(ctx context.Context, oldHost, newHost, newMacAddress string) (*ufspb.DHCPConfig, error) {
	var oldDhcp *ufspb.DHCPConfig
	if oldHost != "" {
		oldDhcp, _ = configuration.GetDHCPConfig(ctx, oldHost)
	}
	if oldDhcp == nil {
		return nil, nil
	}
	dhcp := proto.Clone(oldDhcp).(*ufspb.DHCPConfig)
	dhcp.Hostname = newHost
	if newMacAddress != "" {
		dhcp.MacAddress = newMacAddress
	}
	if err := configuration.DeleteDHCP(ctx, oldHost); err != nil {
		return nil, err
	}
	if _, err := configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{dhcp}); err != nil {
		return nil, errors.Annotate(err, "renameDHCP - Failed to update dhcp from host %q to host %q", oldHost, newHost).Tag(grpcutil.FailedPreconditionTag).Err()
	}
	nu.logChanges(LogDHCPChanges(oldDhcp, nil))
	nu.logChanges(LogDHCPChanges(nil, dhcp))
	return dhcp, nil
}

// Find free ip and update ip-related configs
//
// Can be used in a transaction
func (nu *networkUpdater) addHostHelper(ctx context.Context, vlanName, ipv4Str, macAddress string) (*ufspb.DHCPConfig, error) {
	var ip *ufspb.IP
	var err error
	if ipv4Str != "" {
		if ip, err = getSpecifiedIP(ctx, ipv4Str); err != nil {
			return nil, errors.Annotate(err, "addHostHelper").Tag(grpcutil.FailedPreconditionTag).Err()
		}
	} else {
		if ip, err = getFreeIPHelper(ctx, vlanName); err != nil {
			return nil, errors.Annotate(err, "addHostHelper").Tag(grpcutil.FailedPreconditionTag).Err()
		}
	}

	oldIP := proto.Clone(ip).(*ufspb.IP)
	ip.Occupied = true
	if _, err := configuration.BatchUpdateIPs(ctx, []*ufspb.IP{ip}); err != nil {
		return nil, errors.Annotate(err, "addHostHelper - Failed to update IP %s (%s)", ip.GetId(), ip.GetIpv4Str()).Tag(grpcutil.FailedPreconditionTag).Err()
	}
	nu.Changes = append(nu.Changes, LogIPChanges(oldIP, ip)...)
	oldDhcp, err := configuration.GetDHCPConfig(ctx, nu.Hostname)
	dhcp := &ufspb.DHCPConfig{
		Hostname:   nu.Hostname,
		Ip:         ip.GetIpv4Str(),
		Vlan:       ip.GetVlan(),
		MacAddress: macAddress,
	}
	if _, err := configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{dhcp}); err != nil {
		return nil, errors.Annotate(err, "addHostHelper - Failed to update dhcp for host %s (mac %s)", nu.Hostname, macAddress).Tag(grpcutil.FailedPreconditionTag).Err()
	}
	nu.logChanges(LogDHCPChanges(oldDhcp, dhcp))
	return dhcp, nil
}

func (nu *networkUpdater) addVMHostHelper(ctx context.Context, nwOpt *ufsAPI.NetworkOption, vm *ufspb.VM) error {
	if nwOpt.GetVlan() == "" && nwOpt.GetIp() == "" {
		return status.Errorf(codes.InvalidArgument, "vlan are required for adding a host for a vm")
	}
	nu.Hostname = vm.GetName()
	// 1. Verify if the hostname is already set with IP. if yes, remove the current dhcp configs, update ip.occupied to false
	if err := nu.deleteDHCPHelper(ctx); err != nil {
		return err
	}

	// 2. Get free ip, update the dhcp config and ip.occupied to true
	dhcp, err := nu.addHostHelper(ctx, nwOpt.GetVlan(), nwOpt.GetIp(), vm.GetMacAddress())
	if err != nil {
		return err
	}
	vm.Vlan = dhcp.GetVlan()
	vm.Ip = dhcp.GetIp()
	return nil
}

func (nu *networkUpdater) addLseHostHelper(ctx context.Context, nwOpt *ufsAPI.NetworkOption, lse *ufspb.MachineLSE) error {
	if nwOpt.GetNic() == "" {
		return status.Errorf(codes.InvalidArgument, "nic is required for adding a host for a machine")
	}
	var onlyUpdateNic bool
	if nwOpt.GetVlan() == "" && nwOpt.GetIp() == "" {
		dhcp, err := configuration.GetDHCPConfig(ctx, lse.GetHostname())
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "fail to get existing dhcp record for host %s when only updating nic: %s", lse.GetHostname(), err)
		}
		if dhcp == nil {
			return status.Errorf(codes.InvalidArgument, "one of vlan and ip is required for adding a host for a machine")
		}
		nwOpt.Vlan = dhcp.GetVlan()
		nwOpt.Ip = dhcp.GetIp()
		onlyUpdateNic = true
	}
	nicName := nwOpt.GetNic()
	// Assigning IP to this host.
	nic, err := registration.GetNic(ctx, nicName)
	if err != nil {
		return errors.Annotate(err, fmt.Sprintf("Fail to get nic by name %s", nicName)).Err()
	}
	found := false
	for _, m := range lse.GetMachines() {
		if m == nic.GetMachine() {
			found = true
		}
	}
	if !found {
		return status.Errorf(codes.InvalidArgument, "Nic %s doesn't belong to any of the machines assocated with this host: %#v", nicName, lse.GetMachines())
	}

	nu.Hostname = lse.GetHostname()
	var dhcp *ufspb.DHCPConfig
	if onlyUpdateNic {
		dhcp, err = nu.updateDHCPWithMac(ctx, nic.GetMacAddress())
		if err != nil {
			return err
		}
	} else {
		// 3. Verify if the hostname is already set with IP. if yes, remove the current dhcp configs, update ip.occupied to false
		if err := nu.deleteDHCPHelper(ctx); err != nil {
			return err
		}

		// 4. Get free ip, update the dhcp config and ip.occupied to true
		dhcp, err = nu.addHostHelper(ctx, nwOpt.GetVlan(), nwOpt.GetIp(), nic.GetMacAddress())
		if err != nil {
			return err
		}
	}

	// 5. Update lse to contain the nic which is used to map to the ip and vlan.
	lse.Nic = nic.Name
	lse.Vlan = dhcp.GetVlan()
	lse.Ip = dhcp.GetIp()
	return nil
}

func (nu *networkUpdater) deleteLseHostHelper(ctx context.Context, lse *ufspb.MachineLSE) error {
	for _, m := range lse.GetChromeBrowserMachineLse().GetVms() {
		nu.Hostname = m.GetName()
		if err := nu.deleteDHCPHelper(ctx); err != nil {
			return err
		}
	}

	nu.Hostname = lse.GetHostname()
	if err := nu.deleteDHCPHelper(ctx); err != nil {
		return err
	}
	return nil
}

func (nu *networkUpdater) updateVlanAndIPTable(ctx context.Context, newVlan *ufspb.Vlan) error {
	ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"vlan": newVlan.GetName()})
	if err != nil {
		return err
	}
	toUpdateIPs := make([]*ufspb.IP, 0)

	_, _, freeStartIP, freeEndIP, err := util.ParseVlan(newVlan.GetName(), newVlan.GetVlanAddress(), newVlan.GetFreeStartIpv4Str(), newVlan.GetFreeEndIpv4Str())
	newVlan.FreeStartIpv4Str = freeStartIP
	newVlan.FreeEndIpv4Str = freeEndIP

	freeStartIPInt, err := util.IPv4StrToInt(freeStartIP)
	if err != nil {
		return err
	}
	freeEndIPInt, err := util.IPv4StrToInt(freeEndIP)
	if err != nil {
		return err
	}
	newIPs := newVlan.GetReservedIps()
	newIPMap := make(map[string]bool, len(newIPs))
	for _, ip := range newIPs {
		newIPMap[ip] = true
	}

	for _, ip := range ips {
		_, okNew := newIPMap[ip.GetIpv4Str()]
		if ip.GetIpv4() < freeStartIPInt || ip.GetIpv4() > freeEndIPInt {
			if !ip.GetReserve() {
				oldIP := proto.Clone(ip).(*ufspb.IP)
				ip.Reserve = true
				toUpdateIPs = append(toUpdateIPs, ip)
				nu.Changes = append(nu.Changes, LogIPChanges(oldIP, ip)...)
			}
		} else {
			if okNew && !ip.GetReserve() {
				oldIP := proto.Clone(ip).(*ufspb.IP)
				ip.Reserve = true
				toUpdateIPs = append(toUpdateIPs, ip)
				nu.Changes = append(nu.Changes, LogIPChanges(oldIP, ip)...)
			}
			if !okNew && ip.GetReserve() {
				oldIP := proto.Clone(ip).(*ufspb.IP)
				ip.Reserve = false
				toUpdateIPs = append(toUpdateIPs, ip)
				nu.Changes = append(nu.Changes, LogIPChanges(oldIP, ip)...)
			}
		}
	}
	if _, err := configuration.BatchUpdateIPs(ctx, toUpdateIPs); err != nil {
		return err
	}
	return nil
}
