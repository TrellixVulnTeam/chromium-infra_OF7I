// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	iv2api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsdevice "infra/unifiedfleet/api/v1/models/chromeos/device"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsmanufacturing "infra/unifiedfleet/api/v1/models/chromeos/manufacturing"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

const (
	// Servo port ranges from 9980 to 9999
	// https://chromium.googlesource.com/chromiumos/third_party/hdctools/+/refs/heads/master/servo/servod.py#50
	// However, as there are devices with servo ports < 9980. Limit the validation to 9900.
	servoPortMax = 9999
	servoPortMin = 9000
)

var defaultPools = []string{"DUT_POOL_QUOTA"}

// CreateDUT creates ChromeOSMachineLSE entities for a DUT.
//
// Creates one MachineLSE for DUT and updates another MachineLSE for the
// Labstation(with new Servo info from DUT)
func CreateDUT(ctx context.Context, machinelse *ufspb.MachineLSE) (*ufspb.MachineLSE, error) {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(machinelse)
		machinelses := []*ufspb.MachineLSE{machinelse}

		// Get machine to get zone and rack info for machinelse table indexing
		machine, err := GetMachine(ctx, machinelse.GetMachines()[0])
		if err != nil {
			return errors.Annotate(err, "Unable to get machine %s", machinelse.GetMachines()[0]).Err()
		}

		// Validate input
		if err := validateCreateMachineLSE(ctx, machinelse, nil, machine); err != nil {
			return errors.Annotate(err, "Validation error - Failed to Create ChromeOSMachineLSEDUT").Err()
		}

		// Validate device config
		if err := validateDeviceConfig(ctx, machine); err != nil {
			return errors.Annotate(err, "Validation error - Failed to create DUT").Err()
		}

		oldMachine := proto.Clone(machine).(*ufspb.Machine)
		machine.ResourceState = ufspb.State_STATE_SERVING
		setOutputField(ctx, machine, machinelse)

		// Check if the DUT has Servo information.
		// Update Labstation MachineLSE with new Servo info.
		newServo := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if newServo != nil && newServo.GetServoHostname() != "" {
			// Check if the Labstation MachineLSE exists in the system.
			labstationMachinelse, err := getLabstationMachineLSE(ctx, newServo.GetServoHostname())
			if err != nil {
				return errors.Annotate(err, "Validation error - Cannot get labstation").Err()
			}
			// Clone a copy for logging.
			oldLabstationMachineLseCopy := proto.Clone(labstationMachinelse).(*ufspb.MachineLSE)
			// Client to log labstation changes.
			hcLabstation := getHostHistoryClient(labstationMachinelse)
			// Check if the servo port is assigned, If missing assign a new one.
			if err := assignServoPortIfMissing(labstationMachinelse, newServo); err != nil {
				return err
			}
			// Check if the ServoHostName and ServoPort are already in use
			_, err = validateServoInfoForDUT(ctx, newServo, machinelse.GetName())
			if err != nil {
				return err
			}
			// Clean servo type and servo topology as that will be updated from SSW.
			cleanPreDeployFields(newServo)
			// Update the Labstation MachineLSE with new Servo information.
			// Append the new Servo entry to the Labstation
			if err := appendServoEntryToLabstation(ctx, newServo, labstationMachinelse); err != nil {
				return err
			}
			machinelses = append(machinelses, labstationMachinelse)
			// Log labstation changes to history client.
			hcLabstation.LogMachineLSEChanges(oldLabstationMachineLseCopy, labstationMachinelse)
			if err := hc.SaveChangeEvents(ctx); err != nil {
				return err
			}
		}

		// BatchUpdate both DUT (and its machine), and Labstation
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return errors.Annotate(err, "Fail to update machine %s", machine.GetName()).Err()
		}
		hc.LogMachineChanges(oldMachine, machine)
		machinelse.ResourceState = ufspb.State_STATE_REGISTERED
		_, err = inventory.BatchUpdateMachineLSEs(ctx, machinelses)
		if err != nil {
			return errors.Annotate(err, "Failed to BatchUpdate MachineLSEs").Err()
		}
		hc.LogMachineLSEChanges(nil, machinelse)

		// Update states
		if err := hc.stUdt.addLseStateHelper(ctx, machinelse, machine); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create MachineLSE in datastore: %s", err)
		return nil, err
	}
	return machinelse, nil
}

// UpdateDUT updates a chrome OS DUT.
//
// updates one MachineLSE for DUT and updates Labstation MachineLSE
// (with new Servo info from DUT). If DUT is connected to the same
// Labstation but different port, The servo entry in Labstation is updated.
// If DUT is connected to a different labstation, then old servo info of DUT
// is removed from old Labstation and new servo info from the DUT is added
// to the new labstation.
func UpdateDUT(ctx context.Context, machinelse *ufspb.MachineLSE, mask *field_mask.FieldMask) (*ufspb.MachineLSE, error) {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(machinelse)

		// Get the existing MachineLSE(DUT).
		oldMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if err != nil {
			return errors.Annotate(err, "Failed to get existing MachineLSE").Err()
		}
		// Validate that we are updating a DUT. Will lead to segfault later otherwise.
		if oldMachinelse.GetChromeosMachineLse() == nil || oldMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() == nil {
			return errors.Reason("%s is not a DUT. Cannot update", machinelse.GetName()).Err()
		}

		// Validate the input. Not passing the update mask as there is a different validation for dut.
		if err := validateUpdateMachineLSE(ctx, oldMachinelse, machinelse, nil); err != nil {
			return errors.Annotate(err, "Validation error - Failed to update ChromeOSMachineLSEDUT").Err()
		}

		var machine *ufspb.Machine

		// Validate the update mask and process it.
		if mask != nil && len(mask.Paths) > 0 {
			if err := validateUpdateMachineLSEDUTMask(mask, machinelse); err != nil {
				return err
			}
			machinelse, err = processUpdateMachineLSEUpdateMask(ctx, proto.Clone(oldMachinelse).(*ufspb.MachineLSE), machinelse, mask)
			if err != nil {
				return err
			}
		} else {
			// Full update, Machines cannot be empty.
			if len(machinelse.GetMachines()) > 0 {
				if len(oldMachinelse.GetMachines()) == 0 {
					return errors.Reason("DUT in invalid state. Delete DUT and recreate").Err()
				}
				// Check if the machines have been changed.
				if machinelse.GetMachines()[0] != oldMachinelse.GetMachines()[0] {
					// Ignore error as validateUpdateMachineLSE verifies that the given machine exists.
					machine, _ = GetMachine(ctx, machinelse.GetMachines()[0])
					setOutputField(ctx, machine, machinelse)
				}
			} else {
				// Empty machines field, Invalid update.
				return status.Error(codes.InvalidArgument, "machines field cannot be empty/nil.")
			}
			// Copy state if its not updated.
			if machinelse.GetResourceState() == ufspb.State_STATE_UNSPECIFIED {
				machinelse.ResourceState = oldMachinelse.GetResourceState()
			}
		}

		machinelses := []*ufspb.MachineLSE{machinelse}

		// Extract old and new servo.
		newServo := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		oldServo := oldMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()

		// Common refs to avoid multiple queries
		var newLabstationMachinelse *ufspb.MachineLSE
		var hcNewLabstation *HistoryClient

		// Process newServo and getNewLabstationMachinelse if available. This is done initially to avoid having to repeat it twice
		// in same labstation logic and different labstation logic.
		if newServo != nil && newServo.GetServoHostname() != "" {
			// Check if the Labstation MachineLSE exists in the system first. No use doing anything if it doesn't exist.
			newLabstationMachinelse, err = getLabstationMachineLSE(ctx, newServo.GetServoHostname())
			if err != nil {
				return err
			}

			// Check if a servo port is assigned. Assign one if its not.
			if err := assignServoPortIfMissing(newLabstationMachinelse, newServo); err != nil {
				return err
			}

			// Check if the ServoHostName and ServoPort are already in use.
			_, err = validateServoInfoForDUT(ctx, newServo, machinelse.GetName())
			if err != nil {
				return err
			}
			// For logging new Labstation changes.
			hcNewLabstation = getHostHistoryClient(newLabstationMachinelse)
		}

		// Need to replace oldServo with newServo. If the servos are connected to same labstation then we already have the labstation lse.
		// Replace the oldServo entry with newServo entry.
		if oldServo != nil && oldServo.GetServoHostname() != "" && newServo != nil && newServo.GetServoHostname() != "" && oldServo.GetServoHostname() == newServo.GetServoHostname() {
			newLabstationMachinelseCopy := proto.Clone(newLabstationMachinelse).(*ufspb.MachineLSE)
			if err := replaceServoEntryInLabstation(ctx, oldServo, newServo, newLabstationMachinelse); err != nil {
				return err
			}
			machinelses = append(machinelses, newLabstationMachinelse)
			hcNewLabstation.LogMachineLSEChanges(newLabstationMachinelseCopy, newLabstationMachinelse)
			if err := hcNewLabstation.SaveChangeEvents(ctx); err != nil {
				return err
			}
		} else {
			// Servos might be connected to different labstations or either/both of them are nil. Remove oldServo entry if available
			if oldServo != nil && oldServo.GetServoHostname() != "" {
				// Remove the servo from the labstation
				oldLabstationMachinelse, err := inventory.GetMachineLSE(ctx, oldServo.GetServoHostname())
				if err != nil {
					return err
				}

				// Copy for logging history
				oldLabstationMachineLseCopy := proto.Clone(oldLabstationMachinelse).(*ufspb.MachineLSE)
				hcOldLabstation := getHostHistoryClient(oldLabstationMachinelse)

				// Remove servo from labstation
				if err := removeServoEntryFromLabstation(ctx, oldServo, oldLabstationMachineLseCopy); err != nil {
					return err
				}

				// Record labstation change
				hcOldLabstation.LogMachineLSEChanges(oldLabstationMachinelse, oldLabstationMachineLseCopy)
				if err := hcOldLabstation.SaveChangeEvents(ctx); err != nil {
					return err
				}
				machinelses = append(machinelses, oldLabstationMachineLseCopy)
			}

			// Add new servo information if available
			if newServo != nil && newServo.GetServoHostname() != "" {

				// Make a copy to log changes for the labstation.
				newLabstationMachinelseCopy := proto.Clone(newLabstationMachinelse).(*ufspb.MachineLSE)

				// Append the newServo entry of DUT to the newLabstationMachinelse.
				if err := appendServoEntryToLabstation(ctx, newServo, newLabstationMachinelse); err != nil {
					return err
				}

				hcNewLabstation.LogMachineLSEChanges(newLabstationMachinelseCopy, newLabstationMachinelse)
				if err := hcNewLabstation.SaveChangeEvents(ctx); err != nil {
					return err
				}
				machinelses = append(machinelses, newLabstationMachinelse)
			}
		}

		// BatchUpdate both DUT and Labstation(s)
		_, err = inventory.BatchUpdateMachineLSEs(ctx, machinelses)
		if err != nil {
			logging.Errorf(ctx, "Failed to BatchUpdate ChromeOSMachineLSEDUTs %s", err)
			return err
		}
		hc.LogMachineLSEChanges(oldMachinelse, machinelse)

		// Update state changes.
		dutState := machinelse.GetResourceState()
		if err := hc.stUdt.updateStateHelper(ctx, dutState); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update MachineLSE DUT in datastore: %s", err)
		return nil, errors.Annotate(err, "UpdateDUT - failed transaction").Err()
	}
	return machinelse, nil
}

// assignServoPortIfMissing assigns a servo port to the given servo
// if it's missing. Returns error if the port is out of range.
func assignServoPortIfMissing(labstation *ufspb.MachineLSE, newServo *chromeosLab.Servo) error {
	// If servo port is assigned, nothing is modified.
	if newServo.GetServoPort() != 0 {
		// If the servo is assigned in an invalid range return error
		if port := newServo.GetServoPort(); int(port) > servoPortMax || int(port) < servoPortMin {
			return errors.Reason("Servo port %v is invalid. Valid servo port range [%v, %v]", port, servoPortMax, servoPortMin).Err()
		}
		return nil
	}
	// If servo is  a servo v3 host then assign port 9999
	// TODO(anushruth): Avoid hostname regex by querying machine.
	if util.ServoV3HostnameRegex.MatchString(newServo.GetServoHostname()) {
		newServo.ServoPort = int32(servoPortMax)
		return nil
	}
	ports := make(map[int32]struct{}) // set of ports
	servos := labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	for _, servo := range servos {
		ports[servo.GetServoPort()] = struct{}{} // Assign an empty struct. Note: Empty structs don't take memory
	}
	for idx := int32(servoPortMax); idx >= int32(servoPortMin); idx-- {
		// Assign the highest port available to the servo
		if _, ok := ports[idx]; !ok {
			newServo.ServoPort = int32(idx)
			return nil
		}
	}
	return errors.Reason("Unable to assign a servo port. Check if %s has ports available", labstation.GetHostname()).Err()
}

// validateDeviceConfig checks if the corresponding device config exists in IV2
//
// Checks if the device configuration is known by querying IV2. Returns error if the device config doesn't exist.
func validateDeviceConfig(ctx context.Context, dut *ufspb.Machine) error {
	devConfigID, err := extractDeviceConfigID(dut)
	if err != nil {
		return err
	}

	invV2Client, err := getInventoryV2Client(ctx)
	if err != nil {
		return err
	}

	resp, err := invV2Client.DeviceConfigsExists(ctx, &iv2api.DeviceConfigsExistsRequest{
		ConfigIds: []*device.ConfigId{devConfigID},
	})

	if err != nil {
		return errors.Annotate(err, "Device config validation failed").Err()
	}
	if !resp.GetExists()[0] {
		return errors.Reason("Device config doesn't exist").Err()
	}
	return nil
}

// extractDeviceConfigID returns a corresponding ConfigID object from machine.
func extractDeviceConfigID(dut *ufspb.Machine) (*device.ConfigId, error) {
	crosMachine := dut.GetChromeosMachine()
	if crosMachine == nil {
		return nil, errors.Reason("Invalid machine type. Not a chrome OS machine").Err()
	}

	// Convert the build target and model to lower case to avoid mismatch due to case.
	buildTarget := strings.ToLower(crosMachine.GetBuildTarget())
	model := strings.ToLower(crosMachine.GetModel())
	return &device.ConfigId{
		PlatformId: &device.PlatformId{
			Value: buildTarget,
		},
		ModelId: &device.ModelId{
			Value: model,
		},
	}, nil

}

// cleanPreDeployFields clears servo type and topology.
func cleanPreDeployFields(servo *chromeosLab.Servo) {
	servo.ServoType = ""
	servo.ServoTopology = nil
}

// validateUpdateMachineLSEDUTMask validates the input mask for the given machineLSE.
//
// Assumes that dut and mask aren't empty. This is because this function is not called otherwise.
func validateUpdateMachineLSEDUTMask(mask *field_mask.FieldMask, machinelse *ufspb.MachineLSE) error {
	var servo *chromeosLab.Servo
	var rpm *chromeosLab.RPM

	// GetDut should return an object. Otherwise UpdateDUT isn't called
	dut := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut()
	if peripherals := dut.GetPeripherals(); peripherals != nil {
		servo = peripherals.GetServo()
		rpm = peripherals.GetRpm()
	}

	maskSet := make(map[string]struct{}) // Set of all the masks
	for _, path := range mask.Paths {
		maskSet[path] = struct{}{}
	}
	// validate the give field mask
	for _, path := range mask.Paths {
		switch path {
		case "name":
			return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - name cannot be updated, delete and create a new machinelse instead.")
		case "update_time":
			return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - update_time cannot be updated, it is a output only field.")
		case "machines":
			if machinelse.GetMachines() == nil || len(machinelse.GetMachines()) == 0 || machinelse.GetMachines()[0] == "" {
				return status.Error(codes.InvalidArgument, "machines field cannot be empty/nil.")
			}
		case "dut.hostname":
			return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - hostname cannot be updated, delete and create a new dut.")
		case "dut.servo.hostname":
			if _, ok := maskSet["dut.servo.port"]; servo.GetServoHostname() == "" && ok && servo.GetServoPort() != int32(0) {
				return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - Cannot update servo port. Servo host is being reset.")
			}
			if _, ok := maskSet["dut.servo.serial"]; servo.GetServoHostname() == "" && ok && servo.GetServoSerial() != "" {
				return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - Cannot update servo serial. Servo host is being reset.")
			}
			if _, ok := maskSet["dut.servo.setup"]; servo.GetServoHostname() == "" && ok {
				return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - Cannot update servo setup. Servo host is being reset.")
			}
		case "dut.rpm.host":
			// Check for deletion of the host. Outlet cannot be updated if host is deleted.
			if _, ok := maskSet["dut.rpm.outlet"]; ok && rpm.GetPowerunitName() == "" && rpm.GetPowerunitOutlet() != "" {
				return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - Deleting rpm host deletes everything. Cannot update outlet.")
			}
		case "dut.rpm.outlet":
			// Check for deletion of rpm outlet. This should not be possible without deleting the host.
			if _, ok := maskSet["dut.rpm.host"]; rpm.GetPowerunitOutlet() == "" && (!ok || (ok && rpm.GetPowerunitName() != "")) {
				return status.Error(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - Cannot remove rpm outlet. Please delete rpm.")
			}
		case "deploymentTicket":
		case "tags":
		case "description":
		case "resourceState":
		case "dut.pools":
		case "dut.servo.port":
		case "dut.servo.serial":
		case "dut.servo.setup":
		case "dut.servo.type":
		case "dut.servo.topology":
			// valid fields, nothing to validate.
		default:
			return status.Errorf(codes.InvalidArgument, "validateUpdateMachineLSEDUTUpdateMask - unsupported update mask path %q", path)
		}
	}
	return nil
}

// processUpdateMachineLSEUpdateMask process the update mask and returns the machine lse with updated parameters.
func processUpdateMachineLSEUpdateMask(ctx context.Context, oldMachineLse, newMachineLse *ufspb.MachineLSE, mask *field_mask.FieldMask) (*ufspb.MachineLSE, error) {
	// Extract all the peripherals to avoid doing it for every update in loop.
	var oldServo, newServo *chromeosLab.Servo
	var oldRPM, newRPM *chromeosLab.RPM
	oldDut := oldMachineLse.GetChromeosMachineLse().GetDeviceLse().GetDut()
	newDut := newMachineLse.GetChromeosMachineLse().GetDeviceLse().GetDut()
	if oldDut != nil {
		if oldPeripherals := oldDut.GetPeripherals(); oldPeripherals != nil {
			// Assign empty structs to avoid panics
			oldServo = oldPeripherals.GetServo()
			if oldServo == nil {
				oldServo = &chromeosLab.Servo{}
			}
			oldRPM = oldPeripherals.GetRpm()
			if oldRPM == nil {
				oldRPM = &chromeosLab.RPM{}
			}
		}
	}
	if newDut != nil {
		if newPeripherals := newDut.GetPeripherals(); newPeripherals != nil {
			// Assign empty structs to avoid panics
			newServo = newPeripherals.GetServo()
			if newServo == nil {
				newServo = &chromeosLab.Servo{}
			}
			newRPM = newPeripherals.GetRpm()
			if newRPM == nil {
				newRPM = &chromeosLab.RPM{}
			}
		}
	}
	for _, path := range mask.Paths {
		switch path {
		case "machines":
			// Get machine to get zone and rack info for machinelse table indexing
			machine, err := GetMachine(ctx, newMachineLse.GetMachines()[0])
			if err != nil {
				return oldMachineLse, errors.Annotate(err, "Unable to get machine %s", newMachineLse.GetMachines()[0]).Err()
			}
			oldMachineLse.Machines = newMachineLse.GetMachines()
			// Check permission
			if err := util.CheckPermission(ctx, util.InventoriesUpdate, machine.GetRealm()); err != nil {
				return oldMachineLse, err
			}
			setOutputField(ctx, machine, oldMachineLse)
		case "mlseprototype":
			oldMachineLse.MachineLsePrototype = newMachineLse.GetMachineLsePrototype()
		case "resourceState":
			// Avoid setting unspecified state.
			if newMachineLse.GetResourceState() != ufspb.State_STATE_UNSPECIFIED {
				oldMachineLse.ResourceState = newMachineLse.GetResourceState()
			}
		case "tags":
			if tags := newMachineLse.GetTags(); tags != nil && len(tags) > 0 {
				// Regular tag updates append to the existing tags.
				oldMachineLse.Tags = mergeTags(oldMachineLse.GetTags(), newMachineLse.GetTags())
			} else {
				// Updating tags without any input clears the tags.
				oldMachineLse.Tags = nil
			}
		case "description":
			oldMachineLse.Description = newMachineLse.Description
		case "deploymentTicket":
			oldMachineLse.DeploymentTicket = newMachineLse.GetDeploymentTicket()
		default:
			if strings.HasPrefix(path, "dut") {
				if strings.HasPrefix(path, "dut.servo") {
					processUpdateMachineLSEServoMask(oldServo, newServo, path)
				}
				if strings.HasPrefix(path, "dut.rpm") {
					processUpdateMachineLSERPMMask(oldRPM, newRPM, path)
				}
				processUpdateMachineLSEDUTMask(oldDut, newDut, path)
			}
		}
	}
	if oldServo.GetServoHostname() != "" {
		oldDut.GetPeripherals().Servo = oldServo
	} else { // Reset servo if the servo host is reset.
		oldDut.GetPeripherals().Servo = nil
	}
	if oldRPM.GetPowerunitName() != "" {
		oldDut.GetPeripherals().Rpm = oldRPM
	} else { // Reset RPM if the rpm host is reset.
		oldDut.GetPeripherals().Rpm = nil
	}
	// return existing/old machinelse with new updated values.
	return oldMachineLse, nil
}

// processUpdateMachineLSEUDTMask returns updated dut with the new parameters from the mask.
func processUpdateMachineLSEDUTMask(oldDut, newDut *chromeosLab.DeviceUnderTest, path string) {
	switch path {
	case "dut.pools":
		if pools := newDut.GetPools(); pools != nil && len(pools) > 0 {
			oldDut.Pools = newDut.GetPools()
		} else {
			// Assign default pools if nothing is given.
			oldDut.Pools = defaultPools
		}
	}
}

// processUpdateMachineLSEServoMask returns servo with new updated params from the mask.
func processUpdateMachineLSEServoMask(oldServo, newServo *chromeosLab.Servo, path string) {
	switch path {
	case "dut.servo.hostname":
		oldServo.ServoHostname = newServo.GetServoHostname()
	case "dut.servo.port":
		oldServo.ServoPort = newServo.GetServoPort()
	case "dut.servo.serial":
		oldServo.ServoSerial = newServo.GetServoSerial()
	case "dut.servo.type":
		oldServo.ServoType = newServo.GetServoType()
	case "dut.servo.topology":
		oldServo.ServoTopology = newServo.GetServoTopology()
	}
}

// processUpdateMacineLSERPMMask returns rpm with new updated params from the mask
func processUpdateMachineLSERPMMask(oldRPM, newRPM *chromeosLab.RPM, path string) {
	switch path {
	case "dut.rpm.host":
		oldRPM.PowerunitName = newRPM.GetPowerunitName()
	case "dut.rpm.outlet":
		oldRPM.PowerunitOutlet = newRPM.GetPowerunitOutlet()
	}
}

// GetChromeOSDeviceData returns ChromeOSDeviceData for the given id/hostname from InvV2 and UFS.
func GetChromeOSDeviceData(ctx context.Context, id, hostname string) (*ufspb.ChromeOSDeviceData, error) {
	var lse *ufspb.MachineLSE
	var err error
	if hostname != "" {
		lse, err = GetMachineLSE(ctx, hostname)
		if err != nil {
			return nil, err
		}
		if len(lse.GetMachines()) != 0 {
			id = lse.GetMachines()[0]
		}
	} else {
		machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", id, false)
		if err != nil {
			return nil, err
		}
		if len(machinelses) == 0 {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("DUT not found for asset id %s", id))
		}
		lse = machinelses[0]
	}
	dutState, err := state.GetDutState(ctx, id)
	if err != nil {
		logging.Warningf(ctx, "DutState for %s not found. Error: %s", id, err)
	}
	machine, err := GetMachine(ctx, id)
	if err != nil {
		logging.Errorf(ctx, "Machine for %s not found. Error: %s", id, err)
		return &ufspb.ChromeOSDeviceData{
			LabConfig: lse,
		}, nil
	}
	invV2Client, err := getInventoryV2Client(ctx)
	if err != nil {
		logging.Errorf(ctx, "Failed to InvV2Client. Error: %s", err)
		return &ufspb.ChromeOSDeviceData{
			LabConfig: lse,
		}, nil
	}
	devConfig, err := getDeviceConfig(ctx, invV2Client, machine)
	if err != nil {
		logging.Warningf(ctx, "DeviceConfig for %s not found. Error: %s", id, err)
	}
	hwid := machine.GetChromeosMachine().GetHwid()
	mfgConfig, err := getManufacturingConfig(ctx, invV2Client, hwid)
	if err != nil {
		logging.Warningf(ctx, "ManufacturingConfig for %s not found. Error: %s", hwid, err)
	}
	hwidData, err := getHwidData(ctx, invV2Client, hwid)
	if err != nil {
		logging.Warningf(ctx, "Hwid data for %s not found. Error: %s", hwid, err)
	}
	return &ufspb.ChromeOSDeviceData{
		LabConfig:           lse,
		Machine:             machine,
		DeviceConfig:        devConfig,
		ManufacturingConfig: mfgConfig,
		HwidData:            hwidData,
		DutState:            dutState,
	}, nil
}

// getDeviceConfig get device config form InvV2
func getDeviceConfig(ctx context.Context, inv2Client external.CrosInventoryClient, machine *ufspb.Machine) (*ufsdevice.Config, error) {
	devConfigID, err := extractDeviceConfigID(machine)
	if err != nil {
		return nil, err
	}
	resp, err := inv2Client.GetDeviceConfig(ctx, &iv2api.GetDeviceConfigRequest{
		ConfigId: devConfigID,
	})
	if err != nil {
		return nil, err
	}
	s := proto.MarshalTextString(resp)
	var devConfig ufsdevice.Config
	proto.UnmarshalText(s, &devConfig)
	logging.Debugf(ctx, "InvV2 device config:\n %+v\nUFS device config:\n %+v ", resp, &devConfig)
	return &devConfig, err
}

// getManufacturingConfig get manufacturing config form InvV2
func getManufacturingConfig(ctx context.Context, inv2Client external.CrosInventoryClient, id string) (*ufsmanufacturing.Config, error) {
	resp, err := inv2Client.GetManufacturingConfig(ctx, &iv2api.GetManufacturingConfigRequest{
		Name: id,
	})
	if err != nil {
		return nil, err
	}
	s := proto.MarshalTextString(resp)
	var mfgConfig ufsmanufacturing.Config
	proto.UnmarshalText(s, &mfgConfig)
	logging.Debugf(ctx, "InvV2 manufacturing config:\n %+v\nUFS manufacturing config:\n %+v ", resp, &mfgConfig)
	return &mfgConfig, err
}

// getHwidData get hwid data form InvV2
func getHwidData(ctx context.Context, inv2Client external.CrosInventoryClient, id string) (*ufspb.HwidData, error) {
	resp, err := inv2Client.GetHwidData(ctx, &iv2api.GetHwidDataRequest{
		Name: id,
	})
	if err != nil {
		return nil, err
	}
	s := proto.MarshalTextString(resp)
	var hwidData ufspb.HwidData
	proto.UnmarshalText(s, &hwidData)
	logging.Debugf(ctx, "InvV2 hwid data:\n %+v\nUFS hwid data:\n %+v ", resp, &hwidData)
	return &hwidData, err
}

func getInventoryV2Client(ctx context.Context) (external.CrosInventoryClient, error) {
	es, err := external.GetServerInterface(ctx)
	if err != nil {
		return nil, err
	}
	return es.NewCrosInventoryInterfaceFactory(ctx, config.Get(ctx).GetCrosInventoryHost())
}
