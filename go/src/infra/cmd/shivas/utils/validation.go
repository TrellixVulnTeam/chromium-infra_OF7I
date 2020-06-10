package utils

import (
	"context"
	"strings"

	UfleetAPI "infra/unifiedfleet/api/v1/rpc"
	UfleetUtil "infra/unifiedfleet/app/util"
)

const (
	notFound string = "Entity not found."
)

// EntityExists checks if the given resource exists in the system
func EntityExists(ctx context.Context, ic UfleetAPI.FleetClient, resource, name string) bool {
	switch resource {
	case "MachineLSE":
		return MachineLSEExists(ctx, ic, name)
	case "Machine":
		return MachineExists(ctx, ic, name)
	case "Rack":
		return RackExists(ctx, ic, name)
	case "ChromePlatform":
		return ChromePlatformExists(ctx, ic, name)
	case "Nic":
		return NicExists(ctx, ic, name)
	case "KVM":
		return KVMExists(ctx, ic, name)
	case "RPM":
		return RPMExists(ctx, ic, name)
	case "Switch":
		return SwitchExists(ctx, ic, name)
	case "Drac":
		return DracExists(ctx, ic, name)
	default:
		return false
	}
}

// MachineLSEExists checks if the given MachineLSE exists in the system
func MachineLSEExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetMachineLSE(ctx, &UfleetAPI.GetMachineLSERequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.MachineLSECollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// MachineExists checks if the given Machine exists in the system
func MachineExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetMachine(ctx, &UfleetAPI.GetMachineRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.MachineCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// RackExists checks if the given Rack exists in the system
func RackExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetRack(ctx, &UfleetAPI.GetRackRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.RackCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// ChromePlatformExists checks if the given ChromePlatform exists in the system
func ChromePlatformExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetChromePlatform(ctx, &UfleetAPI.GetChromePlatformRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.ChromePlatformCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// NicExists checks if the given Nic exists in the system
func NicExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetNic(ctx, &UfleetAPI.GetNicRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.NicCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// KVMExists checks if the given KVM exists in the system
func KVMExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetKVM(ctx, &UfleetAPI.GetKVMRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.KVMCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// RPMExists checks if the given RPM exists in the system
func RPMExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetRPM(ctx, &UfleetAPI.GetRPMRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.RPMCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// SwitchExists checks if the given Switch exists in the system
func SwitchExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetSwitch(ctx, &UfleetAPI.GetSwitchRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.SwitchCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}

// DracExists checks if the given Drac exists in the system
func DracExists(ctx context.Context, ic UfleetAPI.FleetClient, name string) bool {
	if len(name) == 0 {
		return false
	}
	_, err := ic.GetDrac(ctx, &UfleetAPI.GetDracRequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.DracCollection, name),
	})
	if err != nil && (strings.Contains(err.Error(), notFound) || strings.Contains(err.Error(), UfleetAPI.InvalidCharacters)) {
		return false
	}
	return true
}
