// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/inventory"
)

// createDUT creates ChromeOSMachineLSE entities for a DUT.
//
// creates one MachineLSE for DUT and updates another MachineLSE for the
// Labstation(with new Servo info from DUT)
func createDUT(ctx context.Context, machinelse *ufspb.MachineLSE, machineNames []string) (*ufspb.MachineLSE, error) {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(machinelse)
		machinelses := []*ufspb.MachineLSE{machinelse}
		// Validate input
		err := validateCreateMachineLSE(ctx, machinelse, machineNames, nil)
		if err != nil {
			return errors.Annotate(err, "Validation error - Failed to Create ChromeOSMachineLSEDUT").Err()
		}

		// Get machine to get lab and rack info for machinelse table indexing
		machine, err := GetMachine(ctx, machineNames[0])
		if err != nil {
			return errors.Annotate(err, "Unable to get machine %s", machineNames[0]).Err()
		}

		setOutputField(ctx, machine, machinelse)

		// Check if the DUT has Servo information.
		// Update Labstation MachineLSE with new Servo info.
		newServo := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if newServo != nil {
			// Check if the Labstation MachineLSE exists in the system.
			labstationMachinelse, err := getLabstationMachineLSE(ctx, newServo.GetServoHostname())
			if err != nil {
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

		// BatchUpdate both DUT and Labstation
		_, err = inventory.BatchUpdateMachineLSEs(ctx, machinelses)
		if err != nil {
			return errors.Annotate(err, "Failed to BatchUpdate MachineLSEs").Err()
		}
		// TODO: skip logging labstation changes for now
		hc.LogMachineLSEChanges(nil, machinelse)

		// Update states
		if err := hc.stUdt.addLseStateHelper(ctx, machinelse); err != nil {
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
func updateDUT(ctx context.Context, machinelse *ufspb.MachineLSE, machineNames []string) (*ufspb.MachineLSE, error) {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(machinelse)
		// Validate the input
		err := validateUpdateMachineLSE(ctx, machinelse, machineNames, nil)
		if err != nil {
			return errors.Annotate(err, "Validation error - Failed to update ChromeOSMachineLSEDUT").Err()
		}

		// Get the existing MachineLSE(DUT)
		oldMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if err != nil {
			return errors.Annotate(err, "Failed to get existing MachineLSE").Err()
		}

		if machineNames == nil || len(machineNames) == 0 {
			// Overwrite the OUTPUT_ONLY fields
			// This is output only field. Assign already existing values.
			machinelse.Machines = oldMachinelse.GetMachines()
		}
		if len(machinelse.GetMachines()) > 0 {
			// Get machine to get lab and rack info for machinelse table indexing
			machine, err := GetMachine(ctx, machinelse.GetMachines()[0])
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
		if err := hc.stUdt.addLseStateHelper(ctx, machinelse); err != nil {
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
