package controller

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

// deployLabstationMaskPaths contains paths for which deploy task if required.
var deployLabstationMaskPaths = []string{
	"machines",
	"labstation.rpm.host",
	"labstation.rpm.outlet",
}

// CreateLabstation creates a new labstation entry in UFS.
func CreateLabstation(ctx context.Context, lse *ufspb.MachineLSE) (*ufspb.MachineLSE, error) {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(lse)

		// Get machine to get zone and rack info for machinelse table indexing
		machine, err := GetMachine(ctx, lse.GetMachines()[0])
		if err != nil {
			return errors.Annotate(err, "unable to get machine %s", lse.GetMachines()[0]).Err()
		}

		// Validate input
		if err := validateCreateMachineLSE(ctx, lse, nil, machine); err != nil {
			return errors.Annotate(err, "Validation error - Failed to create labstation").Err()
		}

		//Copy for logging
		oldMachine := proto.Clone(machine).(*ufspb.Machine)

		machine.ResourceState = ufspb.State_STATE_SERVING
		// Fill the rack/zone OUTPUT only fields for indexing machinelse table/vm table
		setOutputField(ctx, machine, lse)

		// Create the machinelse
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return errors.Annotate(err, "Fail to update machine %s", machine.GetName()).Err()
		}
		hc.LogMachineChanges(oldMachine, machine)
		lse.ResourceState = ufspb.State_STATE_REGISTERED
		if _, err := inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "Failed to BatchUpdate MachineLSEs %s", lse.Name).Err()
		}

		if err := hc.stUdt.addLseStateHelper(ctx, lse, machine); err != nil {
			return errors.Annotate(err, "Fail to update host state").Err()
		}
		hc.LogMachineLSEChanges(nil, lse)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create machinelse in datastore: %s", err)
		return nil, err
	}
	return lse, nil
}

// UpdateLabstation validates and updates the given labstation machine LSE.
func UpdateLabstation(ctx context.Context, machinelse *ufspb.MachineLSE, mask *field_mask.FieldMask) (*ufspb.MachineLSE, error) {
	f := func(ctx context.Context) error {

		hc := getHostHistoryClient(machinelse)

		// Get the existing MachineLSE(Labstation).
		oldMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if err != nil {
			return errors.Annotate(err, "Failed to get existing Labstation").Err()
		}
		// Validate the input. Not passing the update mask as there is a different validation for labstation.
		if err := validateUpdateMachineLSE(ctx, oldMachinelse, machinelse, nil); err != nil {
			return errors.Annotate(err, "Validation error - Failed to update ChromeOSMachineLSEDUT").Err()
		}

		// Assign hostname to the labstation.
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = machinelse.GetHostname()
		// Copy output only fields.
		machinelse.Zone = oldMachinelse.GetZone()
		machinelse.Rack = oldMachinelse.GetRack()

		var machine *ufspb.Machine

		// Validate the update mask and process it.
		if mask != nil && len(mask.Paths) > 0 {
			// Partial update with mask
			if err := validateUpdateLabstationMask(mask, machinelse); err != nil {
				return errors.Annotate(err, "UpdateLabstation - Failed update mask validation").Err()
			}
			if machinelse, err = processUpdateLabstationMask(ctx, proto.Clone(oldMachinelse).(*ufspb.MachineLSE), machinelse, mask); err != nil {
				return errors.Annotate(err, "UpdateLabstation - Failed to process update mask").Err()
			}
		} else {
			// Full update, Machines cannot be empty.
			if len(machinelse.GetMachines()) > 0 {
				if machinelse.GetMachines()[0] != oldMachinelse.GetMachines()[0] {
					if machine, err = GetMachine(ctx, machinelse.GetMachines()[0]); err != nil {
						return err
					}
					// Check if we have permission for the new machine.
					if err := util.CheckPermission(ctx, util.InventoriesUpdate, machine.GetRealm()); err != nil {
						return err
					}
					// Machine was changed. Copy zone and rack info into machinelse.
					setOutputField(ctx, machine, machinelse)
				}
			} else {
				// Empty machines field, Invalid update.
				return status.Error(codes.InvalidArgument, "UpdateLabstation - machines field cannot be empty/nil.")
			}
			// Copy old state if state was not updated.
			if machinelse.GetResourceState() == ufspb.State_STATE_UNSPECIFIED {
				machinelse.ResourceState = oldMachinelse.GetResourceState()
			}
		}

		_, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{machinelse})
		if err != nil {
			logging.Errorf(ctx, "Failed to BatchUpdate ChromeOSMachineLSEDUTs %s", err)
			return err
		}
		hc.LogMachineLSEChanges(oldMachinelse, machinelse)

		// Update state for the labstation.
		if err := hc.stUdt.updateStateHelper(ctx, machinelse.GetResourceState()); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update MachineLSE DUT in datastore: %s", err)
		return nil, errors.Annotate(err, "UpdateLabstation - Failed to update").Err()
	}
	return machinelse, nil
}

// validateUpdateLabstationMask validates the labstation update mask.
func validateUpdateLabstationMask(mask *field_mask.FieldMask, machinelse *ufspb.MachineLSE) error {
	// GetLabstation should return an object. Otherwise UpdateLabstation isn't called
	labstation := machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation()
	rpm := labstation.GetRpm()
	if rpm == nil {
		// Assign an empty rpm to avoid segfaults.
		rpm = &chromeosLab.RPM{}
	}

	maskSet := make(map[string]struct{}) // Set of all the masks
	for _, path := range mask.Paths {
		maskSet[path] = struct{}{}
	}
	// validate the give field mask
	for _, path := range mask.Paths {
		switch path {
		case "name":
			return status.Error(codes.InvalidArgument, "validateUpdateMachineLSELabstationUpdateMask - name cannot be updated, delete and create a new machinelse instead.")
		case "update_time":
			return status.Error(codes.InvalidArgument, "validateUpdateMachineLSELabstationUpdateMask - update_time cannot be updated, it is a output only field.")
		case "machines":
			if machinelse.GetMachines() == nil || len(machinelse.GetMachines()) == 0 || machinelse.GetMachines()[0] == "" {
				return status.Error(codes.InvalidArgument, "machines field cannot be empty/nil.")
			}
		case "labstation.hostname":
			return status.Error(codes.InvalidArgument, "validateUpdateMachineLSELabstationUpdateMask - hostname cannot be updated, delete and create a new dut.")
		case "labstation.rpm.host":
			// Check for deletion of the host. Outlet cannot be updated if host is deleted.
			if _, ok := maskSet["labstation.rpm.outlet"]; ok && rpm.GetPowerunitName() == "" && rpm.GetPowerunitOutlet() != "" {
				return status.Error(codes.InvalidArgument, "validateUpdateMachineLSELabstationUpdateMask - Deleting rpm host deletes everything. Cannot update outlet.")
			}
		case "labstation.rpm.outlet":
			// Check for deletion of rpm outlet. This should not be possible without deleting the host.
			if _, ok := maskSet["labstation.rpm.host"]; rpm.GetPowerunitOutlet() == "" && (!ok || (ok && rpm.GetPowerunitName() != "")) {
				return status.Error(codes.InvalidArgument, "validateUpdateMachineLSELabstationUpdateMask - Cannot remove rpm outlet. Please delete rpm.")
			}
		case "deploymentTicket":
		case "tags":
		case "description":
		case "resourceState":
		case "labstation.pools":
			// valid fields, nothing to validate.
		default:
			return status.Errorf(codes.InvalidArgument, "validateUpdateMachineLSELabstationUpdateMask - unsupported update mask path %q", path)
		}
	}
	return nil
}

// processUpdateLabstationMask processes the update mask provided for the labstation and returns oldMachineLSE updated.
func processUpdateLabstationMask(ctx context.Context, oldMachineLSE, newMachineLSE *ufspb.MachineLSE, mask *field_mask.FieldMask) (*ufspb.MachineLSE, error) {
	oldLabstation := oldMachineLSE.GetChromeosMachineLse().GetDeviceLse().GetLabstation()
	newLabstation := newMachineLSE.GetChromeosMachineLse().GetDeviceLse().GetLabstation()
	for _, path := range mask.Paths {
		switch path {
		case "machines":
			if len(newMachineLSE.GetMachines()) == 0 || newMachineLSE.GetMachines()[0] == "" {
				return nil, status.Errorf(codes.InvalidArgument, "Cannot delete asset connected to %s", oldMachineLSE.GetName())
			}
			// Get machine to get zone and rack info for machinelse table indexing
			machine, err := GetMachine(ctx, newMachineLSE.GetMachines()[0])
			if err != nil {
				return oldMachineLSE, errors.Annotate(err, "Unable to get machine %s", newMachineLSE.GetMachines()[0]).Err()
			}
			// Check if we have permission for the new machine.
			if err := util.CheckPermission(ctx, util.InventoriesUpdate, machine.GetRealm()); err != nil {
				return nil, err
			}
			oldMachineLSE.Machines = newMachineLSE.GetMachines()
			setOutputField(ctx, machine, oldMachineLSE)
		case "resourceState":
			// Avoid setting state to unspecified.
			if newMachineLSE.GetResourceState() != ufspb.State_STATE_UNSPECIFIED {
				oldMachineLSE.ResourceState = newMachineLSE.GetResourceState()
			}
		case "tags":
			if tags := newMachineLSE.GetTags(); tags != nil && len(tags) > 0 {
				// Regular tag updates append to the existing tags.
				oldMachineLSE.Tags = mergeTags(oldMachineLSE.GetTags(), newMachineLSE.GetTags())
			} else {
				// Updating tags without any input clears the tags.
				oldMachineLSE.Tags = nil
			}
		case "description":
			oldMachineLSE.Description = newMachineLSE.Description
		case "deploymentTicket":
			oldMachineLSE.DeploymentTicket = newMachineLSE.GetDeploymentTicket()
		case "labstation.pools":
			oldLabstation.Pools = newLabstation.GetPools()
		case "labstation.rpm.host":
			if newLabstation.GetRpm() == nil || newLabstation.GetRpm().GetPowerunitName() == "" {
				// Ensure that outlet is not being updated when deleting RPM.
				if util.ContainsAnyStrings(mask.Paths, "labstation.rpm.outlet") && newLabstation.GetRpm() != nil && newLabstation.GetRpm().GetPowerunitOutlet() != "" {
					return nil, status.Errorf(codes.InvalidArgument, "Cannot delete RPM and update outlet to %s", newLabstation.GetRpm().GetPowerunitOutlet())
				}
				oldLabstation.Rpm = nil
			} else {
				if oldLabstation.Rpm == nil {
					oldLabstation.Rpm = &chromeosLab.RPM{}
				}
				oldLabstation.GetRpm().PowerunitName = newLabstation.GetRpm().GetPowerunitName()
			}
		case "labstation.rpm.outlet":
			if newLabstation.GetRpm() == nil || newLabstation.GetRpm().GetPowerunitOutlet() == "" {
				// Ensure host is being cleared if the outlet is cleared
				if util.ContainsAnyStrings(mask.Paths, "labstation.rpm.host") && newLabstation.GetRpm() != nil && newLabstation.GetRpm().GetPowerunitName() != "" {
					return nil, status.Errorf(codes.InvalidArgument, "Cannot update RPM to %s and delete outlet", newLabstation.GetRpm().GetPowerunitName())
				}
				// Delete rpm
				oldLabstation.Rpm = nil
			} else {
				if oldLabstation.Rpm == nil {
					oldLabstation.Rpm = &chromeosLab.RPM{}
				}
				// Copy the outlet for update
				oldLabstation.GetRpm().PowerunitOutlet = newLabstation.GetRpm().GetPowerunitOutlet()
			}
		default:
			// Ideally, this piece of code should never execute unless validation is wrong.
			return nil, status.Errorf(codes.Internal, "Unable to process update mask for %s", path)
		}
	}
	return oldMachineLSE, nil
}
