// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/unifiedfleet/api/v1/proto"
	chromeosLab "infra/unifiedfleet/api/v1/proto/chromeos/lab"
	"infra/unifiedfleet/app/model/configuration"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// CreateMachineLSE creates a new machinelse in datastore.
//
// Checks if the resources referenced by the MachineLSE input already exists
// in the system before creating a new MachineLSE
func CreateMachineLSE(ctx context.Context, machinelse *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	machinelse.Name = machinelse.GetHostname()
	err := validateCreateMachineLSE(ctx, machinelse)
	if err != nil {
		return nil, errors.Annotate(err, "create machine LSE").Err()
	}
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		// ChromeOSMachineLSE for a DUT
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = machinelse.GetHostname()
		return createChromeOSMachineLSEDUT(ctx, machinelse)
	}
	// Make the Hostname of the Labstation same as the machinelse name
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = machinelse.GetHostname()
	}
	// ChromeBrowserMachineLSE, ChromeOSMachineLSE for a Server and Labstation
	return inventory.CreateMachineLSE(ctx, machinelse)
}

// UpdateMachineLSE updates machinelse in datastore.
//
// Checks if the resources referenced by the MachineLSE input already exists
// in the system before updating a MachineLSE
func UpdateMachineLSE(ctx context.Context, machinelse *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	machinelse.Name = machinelse.GetHostname()
	err := validateUpdateMachineLSE(ctx, machinelse)
	if err != nil {
		return nil, err
	}
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		// ChromeOSMachineLSE for a DUT
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = machinelse.GetHostname()
		return updateChromeOSMachineLSEDUT(ctx, machinelse)
	}
	// Make the Hostname of the Labstation same as the machinelse name
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = machinelse.GetHostname()
	}
	// ChromeBrowserMachineLSE, ChromeOSMachineLSE for a Server and Labstation
	return inventory.UpdateMachineLSE(ctx, machinelse)
}

// GetMachineLSE returns machinelse for the given id from datastore.
func GetMachineLSE(ctx context.Context, id string) (*fleet.MachineLSE, error) {
	return inventory.GetMachineLSE(ctx, id)
}

// ListMachineLSEs lists the machinelses
func ListMachineLSEs(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.MachineLSE, string, error) {
	return inventory.ListMachineLSEs(ctx, pageSize, pageToken)
}

// DeleteMachineLSE deletes the machinelse in datastore
//
// For referential data intergrity,
// Delete if this MachineLSE is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteMachineLSE(ctx context.Context, id string) error {
	err := validateDeleteMachineLSE(ctx, id)
	if err != nil {
		return err
	}
	f := func(ctx context.Context) error {
		existingMachinelse, err := inventory.GetMachineLSE(ctx, id)
		if err != nil {
			return err
		}
		// Check if it is a DUT MachineLSE and has servo info.
		// Update corresponding Labstation MachineLSE.
		if existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
			existingServo := existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
			if existingServo != nil {
				// 1. remove the existingServo entry of DUT form existingLabstationMachinelse
				existingLabstationMachinelse, err := inventory.GetMachineLSE(ctx, existingServo.GetServoHostname())
				if err != nil {
					return err
				}
				removeServoEntryFromLabstation(existingServo, existingLabstationMachinelse)
				// BatchUpdate Labstation - Using Batch update and not UpdateMachineLSE,
				// because we cant have nested transaction in datastore
				_, err = inventory.BatchUpdateMachineLSEs(ctx, []*fleet.MachineLSE{existingLabstationMachinelse})
				if err != nil {
					logging.Errorf(ctx, "Failed to BatchUpdate Labstation MachineLSE %s", err)
					return err
				}
			}
		}
		return inventory.DeleteMachineLSE(ctx, id)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete MachineLSE in datastore: %s", err)
		return err
	}
	return nil
}

// ImportOSMachineLSEs implements the logic of importing ChromeOS machine lses and related info to backend storage.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportOSMachineLSEs(ctx context.Context, labConfigs []*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig, pageSize int) (*fleetds.OpResults, error) {
	allRes := make(fleetds.OpResults, 0)
	logging.Debugf(ctx, "Importing the machine lse prototypes for OS lab")
	res, err := configuration.ImportMachineLSEPrototypes(ctx, util.GetOSMachineLSEPrototypes())
	if err != nil {
		return res, err
	}
	allRes = append(allRes, *res...)

	lses := util.ToOSMachineLSEs(labConfigs)
	deleteNonExistingMachineLSEs(ctx, lses, pageSize, "os-lab")
	logging.Debugf(ctx, "Importing %d lses", len(lses))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(lses))
		res, err := inventory.ImportMachineLSEs(ctx, lses[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(lses) {
			break
		}
	}
	return &allRes, nil
}

// ImportMachineLSEs implements the logic of importing machine lses and related info to backend storage.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportMachineLSEs(ctx context.Context, hosts []*crimson.PhysicalHost, vms []*crimson.VM, pageSize int) (*fleetds.OpResults, error) {
	allRes := make(fleetds.OpResults, 0)
	logging.Debugf(ctx, "Importing the basic lse prototypes for browser lab")
	lps := []*fleet.MachineLSEPrototype{
		{
			Name: "browser-lab:no-vm",
			VirtualRequirements: []*fleet.VirtualRequirement{
				{
					VirtualType: fleet.VirtualType_VIRTUAL_TYPE_VM,
					Min:         0,
					Max:         0,
				},
			},
		},
		{
			Name: "browser-lab:vm",
			VirtualRequirements: []*fleet.VirtualRequirement{
				{
					VirtualType: fleet.VirtualType_VIRTUAL_TYPE_VM,
					Min:         1,
					// A random number, not true.
					Max: 100,
				},
			},
		},
	}
	res, err := configuration.ImportMachineLSEPrototypes(ctx, lps)
	if err != nil {
		return res, err
	}
	allRes = append(allRes, *res...)

	lses, ips, dhcps := util.ToMachineLSEs(hosts, vms)
	deleteNonExistingMachineLSEs(ctx, lses, pageSize, "browser-lab")
	logging.Debugf(ctx, "Importing %d lses", len(lses))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(lses))
		res, err := inventory.ImportMachineLSEs(ctx, lses[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(lses) {
			break
		}
	}

	logging.Debugf(ctx, "Importing %d ips", len(ips))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(ips))
		res, err := configuration.ImportIPs(ctx, ips[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(ips) {
			break
		}
	}

	logging.Debugf(ctx, "Importing %d dhcps", len(dhcps))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(dhcps))
		res, err := configuration.ImportDHCPConfigs(ctx, dhcps[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(dhcps) {
			break
		}
	}
	return &allRes, nil
}

func deleteNonExistingMachineLSEs(ctx context.Context, machineLSEs []*fleet.MachineLSE, pageSize int, lseType string) (*fleetds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range machineLSEs {
		resMap[r.GetName()] = true
	}
	resp, err := inventory.GetAllMachineLSEs(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	var toDeleteDHCPHost []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*fleet.MachineLSE)
		if lseType == "browser-lab" && s.GetChromeosMachineLse() != nil {
			continue
		}
		if lseType == "os-lab" && s.GetChromeBrowserMachineLse() != nil {
			continue
		}
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
			toDeleteDHCPHost = append(toDeleteDHCPHost, s.GetName())
		}
		if s.GetChromeBrowserMachineLse() != nil {
			for _, vm := range s.GetChromeBrowserMachineLse().GetVms() {
				toDeleteDHCPHost = append(toDeleteDHCPHost, vm.GetHostname())
			}
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing machine lses", len(toDelete))
	allRes := *deleteByPage(ctx, toDelete, pageSize, inventory.DeleteMachineLSEs)
	logging.Debugf(ctx, "Deleting %d non-existing host and vm-related dhcps", len(toDelete))
	allRes = append(allRes, *deleteByPage(ctx, toDelete, pageSize, configuration.DeleteDHCPs)...)
	return &allRes, nil
}

// createChromeOSMachineLSEDUT creates ChromeOSMachineLSE entities.
//
// creates one MachineLSE for DUT and updates another MachineLSE for the
// Labstation(with new Servo info from DUT)
func createChromeOSMachineLSEDUT(ctx context.Context, machinelse *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	logging.Debugf(ctx, "Creating ChromeOS DUT's LSE")
	f := func(ctx context.Context) error {
		machinelses := make([]*fleet.MachineLSE, 0, 0)

		// A. Check if the MachineLSE(DUT) already exists in the system for
		existingMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if status.Code(err) == codes.Internal {
			return err
		}
		if existingMachinelse != nil {
			return status.Errorf(codes.AlreadyExists, fleetds.AlreadyExists)
		}
		machinelses = append(machinelses, machinelse)

		// B. Check if the DUT has Servo information.
		// Update Labstation MachineLSE with new Servo info.
		newServo := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if newServo != nil {
			// 1. Check if the Labstation MachineLSE exists in the system.
			labstationMachinelse, err := getLabstationMachineLSE(ctx, newServo.GetServoHostname())
			if err != nil {
				return err
			}
			// 2. Check if the ServoHostName and ServoPort are already in use
			_, err = validateServoInfoForDUT(ctx, newServo, machinelse.GetName())
			if err != nil {
				return err
			}
			// 3. Update the Labstation MachineLSE with new Servo information.
			// Append the new Servo entry to the Labstation
			appendServoEntryToLabstation(newServo, labstationMachinelse)
			machinelses = append(machinelses, labstationMachinelse)
		}

		// C. BatchUpdate both DUT and Labstation
		_, err = inventory.BatchUpdateMachineLSEs(ctx, machinelses)
		if err != nil {
			logging.Errorf(ctx, "Failed to BatchUpdate MachineLSEs %s", err)
			return err
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create/update MachineLSE in datastore: %s", err)
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
func updateChromeOSMachineLSEDUT(ctx context.Context, machinelse *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	f := func(ctx context.Context) error {
		machinelses := make([]*fleet.MachineLSE, 0, 0)

		// A. Check if the MachineLSE(DUT) doesnt exist in the system
		oldMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if status.Code(err) == codes.Internal {
			return err
		}
		if oldMachinelse == nil {
			return status.Errorf(codes.NotFound, fleetds.NotFound)
		}
		machinelses = append(machinelses, machinelse)

		// B. Check if the DUT has Servo information.
		// Update Labstation MachineLSE with new Servo info.
		newServo := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if newServo != nil {
			// 1. Check if the Labstation MachineLSE exists in the system.
			newLabstationMachinelse, err := getLabstationMachineLSE(ctx, newServo.GetServoHostname())
			if err != nil {
				return err
			}
			// 2. Check if the ServoHostName and ServoPort are already in use
			_, err = validateServoInfoForDUT(ctx, newServo, machinelse.GetName())
			if err != nil {
				return err
			}
			// 3. Update the Labstation MachineLSE with new Servo information.
			oldServo := oldMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
			// check if the DUT is connected to the same Labstation or different Labstation
			if newServo.GetServoHostname() == oldServo.GetServoHostname() {
				// DUT is connected to the same Labstation,
				// 1. replace the oldServo entry from the Labstation with the newServo entry
				replaceServoEntryInLabstation(oldServo, newServo, newLabstationMachinelse)
				machinelses = append(machinelses, newLabstationMachinelse)
			} else {
				// DUT is connected to a different Labstation,
				// 1. remove the oldServo entry of DUT form oldLabstationMachinelse
				oldLabstationMachinelse, err := inventory.GetMachineLSE(ctx, oldServo.GetServoHostname())
				if err != nil {
					return err
				}
				removeServoEntryFromLabstation(oldServo, oldLabstationMachinelse)
				machinelses = append(machinelses, oldLabstationMachinelse)
				// 2. Append the newServo entry of DUT to the newLabstationMachinelse
				appendServoEntryToLabstation(newServo, newLabstationMachinelse)
				machinelses = append(machinelses, newLabstationMachinelse)
			}
		}

		// C. BatchUpdate both DUT and Labstation/s
		_, err = inventory.BatchUpdateMachineLSEs(ctx, machinelses)
		if err != nil {
			logging.Errorf(ctx, "Failed to BatchUpdate MachineLSEs %s", err)
			return err
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update MachineLSE DUT in datastore: %s", err)
		return nil, err
	}
	return machinelse, nil
}

// validateServoInfoForDUT Checks if the DUT Machinelse has ServoHostname and ServoPort
// already used by a different deployed DUT
func validateServoInfoForDUT(ctx context.Context, servo *chromeosLab.Servo, DUTHostname string) (string, error) {
	logging.Debugf(ctx, "Validating the DUT's servo is already occupied")
	servoID := fleetds.GetServoID(servo.GetServoHostname(), servo.GetServoPort())
	dutMachinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "servo_id", servoID, true)
	if err != nil {
		return "", err
	}
	if dutMachinelses != nil && dutMachinelses[0].GetName() != DUTHostname {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Port: %d in Labstation: %s is already "+
			"in use by DUT: %s. Please provide a different ServoPort.\n",
			servo.GetServoPort(), servo.GetServoHostname(), dutMachinelses[0].GetName()))
		logging.Errorf(ctx, errorMsg.String())
		return dutMachinelses[0].GetName(), status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	if dutMachinelses != nil {
		return dutMachinelses[0].GetName(), nil
	}
	return "", nil
}

// getLabstationMachineLSE get the Labstation MachineLSE
func getLabstationMachineLSE(ctx context.Context, labstationMachinelseName string) (*fleet.MachineLSE, error) {
	labstationMachinelse, err := inventory.GetMachineLSE(ctx, labstationMachinelseName)
	if status.Code(err) == codes.Internal {
		return nil, err
	}
	if labstationMachinelse == nil {
		// There is no Labstation MachineLSE existing in the system
		errorMsg := fmt.Sprintf("Labstation %s not found in the system. "+
			"Please deploy the Labstation %s before deploying the DUT.",
			labstationMachinelseName, labstationMachinelseName)
		logging.Errorf(ctx, errorMsg)
		return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	return labstationMachinelse, nil
}

// appendServoEntryToLabstation append servo entry to the Labstation
func appendServoEntryToLabstation(newServo *chromeosLab.Servo, labstationMachinelse *fleet.MachineLSE) {
	existingServos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	existingServos = append(existingServos, newServo)
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = existingServos
}

// replaceServoEntryInLabstation replaces oldServo entry with newServo entry in the Labstation
func replaceServoEntryInLabstation(oldServo, newServo *chromeosLab.Servo, labstationMachinelse *fleet.MachineLSE) {
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	for i, s := range servos {
		if s.GetServoHostname() == oldServo.GetServoHostname() && s.GetServoPort() == oldServo.GetServoPort() {
			servos[i] = newServo
			break
		}
	}
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = servos
}

// removeServoEntryFromLabstation removes servo entry from the Labstation
func removeServoEntryFromLabstation(servo *chromeosLab.Servo, labstationMachinelse *fleet.MachineLSE) {
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	for i, s := range servos {
		if s.GetServoHostname() == servo.GetServoHostname() && s.GetServoPort() == servo.GetServoPort() {
			servos[i] = servos[len(servos)-1]
			servos = servos[:len(servos)-1]
			break
		}
	}
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = servos
}

// validateCreateMachineLSE validates if a machinelse can be created in the datastore.
func validateCreateMachineLSE(ctx context.Context, machinelse *fleet.MachineLSE) error {
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		newServos := machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
		if len(newServos) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Servos are not allowed to be added in deploying labstations")
		}
	}
	return validateMachineLSEResources(ctx, machinelse)
}

// validateUpdateMachineLSE validates if a machinelse can be updated in the datastore.
func validateUpdateMachineLSE(ctx context.Context, machinelse *fleet.MachineLSE) error {
	// This check is only for a Labstation
	// Check if labstation MachineLSE is updating any servo information
	// It is also not allowed to update the servo Hostname and servo Port of any servo.
	// Servo info is added/updated into Labstation only when a DUT(MachineLSE) is added/updated
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		newServos := machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
		existingLabstationMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if status.Code(err) == codes.Internal {
			return err
		}
		if existingLabstationMachinelse != nil {
			existingServos := existingLabstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			if !testServoEq(newServos, existingServos) {
				return status.Errorf(codes.FailedPrecondition, "Servos are not allowed to be updated in redeploying labstations")
			}
		}
	}
	return validateMachineLSEResources(ctx, machinelse)
}

func validateMachineLSEResources(ctx context.Context, machinelse *fleet.MachineLSE) error {
	var resources []*Resource
	var errorMsg strings.Builder
	machineIDs := machinelse.GetMachines()
	machineLSEPrototypeID := machinelse.GetMachineLsePrototype()
	vlanID := machinelse.GetChromeosMachineLse().GetServerLse().GetSupportedRestrictedVlan()
	rpmID := machinelse.GetChromeosMachineLse().GetDeviceLse().GetRpmInterface().GetRpm()

	if len(machineIDs) != 0 {
		for _, machineID := range machineIDs {
			// TODO: should disallow deploying un-registered machines later
			if machineID != "" {
				resources = append(resources, GetMachineResource(machineID))
			}
		}
	}
	if machineLSEPrototypeID != "" {
		resources = append(resources, GetMachineLSEProtoTypeResource(machineLSEPrototypeID))
	}
	if vlanID != "" {
		resources = append(resources, GetVlanResource(vlanID))
	}
	if rpmID != "" {
		resources = append(resources, GetRPMResource(rpmID))
	}
	return ResourceExist(ctx, resources, &errorMsg)
}

// validateDeleteMachineLSE validates if a MachineLSE can be deleted
func validateDeleteMachineLSE(ctx context.Context, id string) error {
	existingMachinelse, err := inventory.GetMachineLSE(ctx, id)
	if err != nil {
		return err
	}
	if existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		existingServos := existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
		nonDeletedHosts := make([]string, 0, len(existingServos))
		for _, servo := range existingServos {
			dutHostName, err := validateServoInfoForDUT(ctx, servo, "")
			if err != nil {
				nonDeletedHosts = append(nonDeletedHosts, dutHostName)
			}
		}
		if len(nonDeletedHosts) != 0 {
			errorMsg := fmt.Sprintf("Labstation %s cannot be deleted because there are servos in the labstation referenced by other DUTs: %s.", id, strings.Join(nonDeletedHosts, ", "))
			logging.Errorf(ctx, errorMsg)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}
	}
	return nil
}
