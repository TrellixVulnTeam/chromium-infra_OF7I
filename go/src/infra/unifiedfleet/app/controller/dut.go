// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"

	iv2api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

const (
	// Servo port ranges from 9980 to 9999
	// https://chromium.googlesource.com/chromiumos/third_party/hdctools/+/refs/heads/master/servo/servod.py#50
	// However, as there are devices with servo ports < 9980. Limit the validation to 9900.
	servoPortMax = 9999
	servoPortMin = 9900
)

var servoV3HostnameRegex = regexp.MustCompile(`.*-servo`)

// createDUT creates ChromeOSMachineLSE entities for a DUT.
//
// creates one MachineLSE for DUT and updates another MachineLSE for the
// Labstation(with new Servo info from DUT)
func createDUT(ctx context.Context, machinelse *ufspb.MachineLSE) (*ufspb.MachineLSE, error) {
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
				return err
			}
			// Check if the servo port is assigned, If missing assign a new one.
			if err := assignServoPortIfMissing(labstationMachinelse, newServo); err != nil {
				return err
			}
			// Check if the ServoHostName and ServoPort are already in use
			_, err = validateServoInfoForDUT(ctx, newServo, machinelse.GetName())
			if err != nil {
				return err
			}
			// Update the Labstation MachineLSE with new Servo information.
			// Append the new Servo entry to the Labstation
			appendServoEntryToLabstation(newServo, labstationMachinelse)
			machinelses = append(machinelses, labstationMachinelse)
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
		// TODO: skip logging labstation changes for now
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

// updateChromeOSMachineLSEDUT updates ChromeOSMachineLSE entities.
//
// updates one MachineLSE for DUT and updates Labstation MachineLSE
// (with new Servo info from DUT). If DUT is connected to the same
// Labstation but different port, The servo entry in Labstation is updated.
// If DUT is connected to a different labstation, then old servo info of DUT
// is removed from old Labstation and new servo info from the DUT is added
// to the new labstation.
func updateDUT(ctx context.Context, machinelse *ufspb.MachineLSE) (*ufspb.MachineLSE, error) {
	// TODO(eshwarn) : provide partial update for dut.
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(machinelse)

		// Get the existing MachineLSE(DUT)
		oldMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if err != nil {
			return errors.Annotate(err, "Failed to get existing MachineLSE").Err()
		}

		// Validate the input
		if err := validateUpdateMachineLSE(ctx, oldMachinelse, machinelse, nil); err != nil {
			return errors.Annotate(err, "Validation error - Failed to update ChromeOSMachineLSEDUT").Err()
		}

		var machine *ufspb.Machine
		if len(machinelse.GetMachines()) > 0 {
			// Get machine to get lab and rack info for machinelse table indexing
			machine, err = GetMachine(ctx, machinelse.GetMachines()[0])
			if err != nil {
				return errors.Annotate(err, "Unable to get machine %s", machinelse.GetMachines()[0]).Err()
			}
			setOutputField(ctx, machine, machinelse)
		}

		machinelses := []*ufspb.MachineLSE{machinelse}

		// Check if the DUT has Servo information.
		// Update Labstation MachineLSE with new Servo info.
		newServo := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if newServo != nil {
			// Check if the Labstation MachineLSE exists in the system.
			newLabstationMachinelse, err := getLabstationMachineLSE(ctx, newServo.GetServoHostname())
			if err != nil {
				return err
			}
			// Check if a servo port is assigned. Assign one if its not
			if err := assignServoPortIfMissing(newLabstationMachinelse, newServo); err != nil {
				return err
			}
			// Check if the ServoHostName and ServoPort are already in use
			_, err = validateServoInfoForDUT(ctx, newServo, machinelse.GetName())
			if err != nil {
				return err
			}
			// Update the Labstation MachineLSE with new Servo information.
			oldServo := oldMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
			// check if the DUT is connected to the same Labstation or different Labstation
			if newServo.GetServoHostname() == oldServo.GetServoHostname() {
				// DUT is connected to the same Labstation,
				// replace the oldServo entry from the Labstation with the newServo entry
				replaceServoEntryInLabstation(oldServo, newServo, newLabstationMachinelse)
				machinelses = append(machinelses, newLabstationMachinelse)
			} else {
				// DUT is connected to a different Labstation,
				// remove the oldServo entry of DUT form oldLabstationMachinelse
				oldLabstationMachinelse, err := inventory.GetMachineLSE(ctx, oldServo.GetServoHostname())
				if err != nil {
					return err
				}
				removeServoEntryFromLabstation(oldServo, oldLabstationMachinelse)
				machinelses = append(machinelses, oldLabstationMachinelse)
				// Append the newServo entry of DUT to the newLabstationMachinelse
				appendServoEntryToLabstation(newServo, newLabstationMachinelse)
				machinelses = append(machinelses, newLabstationMachinelse)
			}
		}

		// BatchUpdate both DUT and Labstation/s
		_, err = inventory.BatchUpdateMachineLSEs(ctx, machinelses)
		if err != nil {
			logging.Errorf(ctx, "Failed to BatchUpdate ChromeOSMachineLSEDUTs %s", err)
			return err
		}
		// TODO: skip logging labstation changes for now
		hc.LogMachineLSEChanges(oldMachinelse, machinelse)

		// Update states
		if err := hc.stUdt.addLseStateHelper(ctx, machinelse, machine); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update MachineLSE DUT in datastore: %s", err)
		return nil, err
	}
	return machinelse, nil
}

// assignServoPortIfMissing assigns a servo port to the given servo
// if it's missing. Returns error if the port is out of range.
func assignServoPortIfMissing(labstation *ufspb.MachineLSE, newServo *ufslab.Servo) error {
	// If servo port is assigned, nothing is modified.
	if newServo.GetServoPort() != 0 {
		return nil
	}
	// If the servo is assigned in an invalid range return error
	if port := newServo.GetServoPort(); port > servoPortMax || port < servoPortMin {
		return errors.Reason("Servo port %v is invalid. Valid servo port range [%v, %v]", port, servoPortMax, servoPortMin).Err()
	}
	// If servo is  a servo v3 host then assign port 9999
	// TODO(anushruth): Avoid hostname regex by querying machine.
	if servoV3HostnameRegex.MatchString(newServo.GetServoHostname()) {
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
			break
		}
	}
	return nil
}

// validateDeviceConfig checks if the corresponding device config exists in IV2
//
// Checks if the device configuration is known by querying IV2. Returns error if the device config doesn't exist.
func validateDeviceConfig(ctx context.Context, dut *ufspb.Machine) error {
	devConfigID, err := extractDeviceConfigID(dut)
	if err != nil {
		return err
	}

	es, err := external.GetServerInterface(ctx)
	if err != nil {
		return err
	}

	inv2Client, err := es.NewCrosInventoryInterfaceFactory(ctx, config.Get(ctx).GetCrosInventoryHost())
	if err != nil {
		return err
	}

	resp, err := inv2Client.DeviceConfigsExists(ctx, &iv2api.DeviceConfigsExistsRequest{
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
