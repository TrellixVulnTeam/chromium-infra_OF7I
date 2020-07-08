// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	fleet "infra/unifiedfleet/api/v1/proto"
	chromeosLab "infra/unifiedfleet/api/v1/proto/chromeos/lab"
	UfleetAPI "infra/unifiedfleet/api/v1/rpc"
	UfleetUtil "infra/unifiedfleet/app/util"
)

// Interactive mode messages for user input
const (
	InputDetails              string = "Please enter the details: "
	RequiredField             string = "is a required field. It cannot be blank/empty."
	WrongInput                string = "\n  WRONG INPUT!!\n"
	ChooseOption              string = "\n Choose an option\n"
	ChooseChromePlatform      string = "\n Choose a ChromePlatform\n"
	ChooseMachineLSEPrototype string = "\n Choose a MachineLSE Prototype\n"
	ChooseChameleonType       string = "\n Choose a ChameleonType \n"
	ChooseCameraType          string = "\n Choose a CameraType \n"
	ChooseAntennaConnection   string = "\n Choose an AntennaConnection \n"
	ChooseRouter              string = "\n Choose a Router \n"
	ChooseCableType           string = "\n Choose a Cable \n"
	ChooseCameraboxFacing     string = "\n Choose a Facing for CameraBox \n"
	ChoosePheripheralType     string = "\n Choose a PheripheralType \n"
	ChooseVirtualType         string = "\n Choose a VirtualType \n"
	OptionToEnter             string = "\nDo you want to enter a "
	OptionToEnterMore         string = "\nDo you want to enter one more "
	ChooseLab                 string = "\n Choose a Lab\n"
	BroswerOrOSLab            string = "1=\"Browser Lab\"\n2=\"OS Lab\"\n"
	BrowserOrATLOrACSLab      string = "1=\"Browser Lab\"\n2=\"ATL Lab\"\n3=\"ACS Lab\"\n"
	DutOrLabstationOrServer   string = "1=\"DUT\"\n2=\"Labstation\"\n3=\"Server\"\n"
	DoesNotExist              string = " doesnt not exist in the system. Please check and enter again."
	AlreadyExists             string = " already exists in the system. Please check and enter again."
	ATL                       string = "ATL"
	ACS                       string = "ACS"
	Browser                   string = "Browser"
	Unknown                   string = "Unknown"
	maxPageSize               int32  = 1000
	YesNo                     string = " (y/n)"
	ATLLab                    string = "atl-lab:"
	ACSLab                    string = "acs-lab:"
	BrowserLab                string = "browser-lab:"
	MinMaxError               string = "Maximum value must be greater than or equal to Minimum value."
)

// Input deatils for the input variable
//
// Key - input variable name
// Desc - description of the variable
// Required - if the variable is a required field
type Input struct {
	Key      string
	Desc     string
	Required bool
}

// GetInteractiveInput collects the scanned string list.
//
// Name(string) -> CapacityPort(int)
func GetInteractiveInput() []string {
	inputs := make([]string, 0)
	fmt.Print("Please scan: ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		iput := scanner.Text()
		if iput == "" {
			break
		}
		inputs = append(inputs, iput)
		fmt.Print("Continue (please enter without scanning if you finish): ")
	}
	return inputs
}

// GetSwitchInteractiveInput get switch input in interactive mode
func GetSwitchInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, s *fleet.Switch, update bool) {
	input := &Input{
		Key:      "Name",
		Desc:     UfleetAPI.ValidName,
		Required: true,
	}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(InputDetails)
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "Name":
				if !UfleetAPI.IDRegex.MatchString(value) {
					break
				}
				if !update && SwitchExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, AlreadyExists)
					break
				} else if update && !SwitchExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				s.Name = value
				input = &Input{
					Key: "CapacityPort",
				}
			case "CapacityPort":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					s.CapacityPort = port
				}
				input = nil
			}
			break
		}
	}
}

// GetMachineInteractiveInput get Machine input in interactive mode
//
// Name(string) -> Lab(enum) -> Browser/OS LAB(choice to branch) ->
// -> getBrowserMachine()/getOSMachine() -> Realm(string)
func GetMachineInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, machine *fleet.Machine, update bool) {
	machine.Location = &fleet.Location{}
	input := &Input{
		Key:      "Name",
		Desc:     UfleetAPI.ValidName,
		Required: true,
	}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(InputDetails)
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "Name":
				if !UfleetAPI.IDRegex.MatchString(value) {
					input.Desc = UfleetAPI.ValidName
					break
				}
				if !update && MachineExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, AlreadyExists)
					break
				} else if update && !MachineExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				machine.Name = value
				input = &Input{
					Key:  "Lab",
					Desc: fmt.Sprintf("%s%s", ChooseLab, createKeyValuePairs(fleet.Lab_name)),
				}
			case "Lab":
				if value == "" || value == "0" {
					input = &Input{
						Key:      "Browser/OS LAB",
						Desc:     fmt.Sprintf("%s%s", ChooseLab, BroswerOrOSLab),
						Required: true,
					}
				} else {
					option := getSelectionInput(value, fleet.Lab_name, input)
					if option == -1 {
						break
					}
					machine.Location.Lab = fleet.Lab(option)
					input = &Input{
						Key: "Realm",
					}
					if getLab(machine.Location.Lab) == Browser {
						// Chorome Browser lab
						getBrowserMachine(ctx, ic, scanner, machine)
					} else if getLab(machine.Location.Lab) == ACS ||
						getLab(machine.Location.Lab) == ATL {
						// ChromeOS lab
						getOSMachine(ctx, ic, scanner, machine)
					} else {
						// Unknown or fleet.Lab_LAB_CHROMEOS_SANTIEM
						input = &Input{
							Key:      "Browser/OS LAB",
							Desc:     fmt.Sprintf("%s%s", ChooseLab, BroswerOrOSLab),
							Required: true,
						}
					}
				}
			case "Browser/OS LAB":
				if value == "1" {
					// Chrome Browser lab
					getBrowserMachine(ctx, ic, scanner, machine)
					input = &Input{
						Key: "Realm",
					}
				} else if value == "2" {
					// Chrome OS lab
					getOSMachine(ctx, ic, scanner, machine)
					input = &Input{
						Key: "Realm",
					}
				} else {
					input.Desc = fmt.Sprintf("%s%s%s", WrongInput, ChooseLab, BroswerOrOSLab)
				}
			case "Realm":
				machine.Realm = value
				input = nil
			}
			break
		}
	}
}

// getOSMachine get Chrome OS Machine input in interactive mode
//
// Rack(string) -> Aisle(string) -> Row(string) -> Rack Number(string) ->
// -> Shelf(string) -> Position(string)
func getOSMachine(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machine *fleet.Machine) {
	machine.Device = &fleet.Machine_ChromeosMachine{
		ChromeosMachine: &fleet.ChromeOSMachine{},
	}
	input := &Input{
		Key:      "Rack",
		Required: false,
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "Rack":
				if value != "" && !RackExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				machine.Location.Rack = value
				input = &Input{
					Key: "Aisle",
				}
			case "Aisle":
				machine.Location.Aisle = value
				input = &Input{
					Key: "Row",
				}
			case "Row":
				machine.Location.Row = value
				input = &Input{
					Key: "Rack Number",
				}
			case "Rack Number":
				machine.Location.RackNumber = value
				input = &Input{
					Key: "Shelf",
				}
			case "Shelf":
				machine.Location.Shelf = value
				input = &Input{
					Key: "Position",
				}
			case "Position":
				machine.Location.Position = value
				input = nil
			}
			break
		}
	}
}

// getBrowserMachine get Browser Machine input in interactive mode
//
// Rack(string, resource) -> DisplayName(string) ->
// -> ChromePlatform(string, resource) -> Nics(repeated string, resource) ->
// -> KVM(string, resource) -> KVM Port(int) -> RPM(string, resource) ->
// -> RPM Port(int) -> Switch(string, resource) -> Switch Port(int) ->
// -> Drac(string, resource) -> DeploymentTicket(string) -> Description(string)
func getBrowserMachine(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machine *fleet.Machine) {
	browserMachine := &fleet.ChromeBrowserMachine{}
	machine.Device = &fleet.Machine_ChromeBrowserMachine{
		ChromeBrowserMachine: browserMachine,
	}
	chromePlatforms := getAllChromePlatforms(ctx, ic)
	input := &Input{
		Key:      "Rack",
		Required: false,
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "Rack":
				if value != "" && !RackExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
				} else {
					machine.Location.Rack = value
					input = &Input{
						Key: "DisplayName",
					}
				}
			case "DisplayName":
				browserMachine.DisplayName = value
				input = &Input{
					Key:  "ChromePlatform",
					Desc: fmt.Sprintf("%s%s", ChooseChromePlatform, createKeyValuePairs(chromePlatforms)),
				}
			case "ChromePlatform":
				if value != "" {
					option := getSelectionInput(value, chromePlatforms, input)
					if option == -1 {
						break
					}
					browserMachine.ChromePlatform = chromePlatforms[option]
				}
				input = &Input{
					Key:      "Nics (y/n)",
					Desc:     fmt.Sprintf("%s%s?", OptionToEnter, "Nic"),
					Required: true,
				}
				// repeated Nic
			case "Nics (y/n)":
				vals, done := getRepeatedStringInput(ctx, ic, scanner, value, "Nic", input, true)
				if done {
					browserMachine.Nics = vals
					input = &Input{
						Key: "KVM",
					}
				}
			case "KVM":
				if value != "" && !KVMExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				browserMachine.KvmInterface = &fleet.KVMInterface{
					Kvm: value,
				}
				if value != "" {
					input = &Input{
						Key: "KVM Port",
					}
				} else {
					input = &Input{
						Key: "RPM",
					}
				}
			case "KVM Port":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					browserMachine.KvmInterface.Port = port
				}
				input = &Input{
					Key: "RPM",
				}
			case "RPM":
				if value != "" && !RPMExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				browserMachine.RpmInterface = &fleet.RPMInterface{
					Rpm: value,
				}
				if value != "" {
					input = &Input{
						Key: "RPM Port",
					}
				} else {
					input = &Input{
						Key: "Drac",
					}
				}
			case "RPM Port":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					browserMachine.RpmInterface.Port = port
				}
				input = &Input{
					Key: "Drac",
				}
			case "Drac":
				if value != "" && !DracExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
				} else {
					browserMachine.Drac = value
					input = &Input{
						Key: "DeploymentTicket",
					}
				}
			case "DeploymentTicket":
				browserMachine.DeploymentTicket = value
				input = &Input{
					Key: "Description",
				}
			case "Description":
				browserMachine.Description = value
				input = nil
			}
			break
		}
	}
}

// GetMachinelseInteractiveInput get MachineLSE input in interactive mode
//
// Name(string) -> Broswer/ATL/ACS LAB(choice to branch) ->
// -> getBrowserMachineLse()/getOSMachineLse() ->
// -> Machine(repeated string, resource)
func GetMachinelseInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, machinelse *fleet.MachineLSE, update bool) {
	input := &Input{
		Key:      "Hostname",
		Desc:     UfleetAPI.ValidName,
		Required: true,
	}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(InputDetails)
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "Hostname":
				if !UfleetAPI.IDRegex.MatchString(value) {
					input.Desc = UfleetAPI.ValidName
					break
				}
				if !update && MachineLSEExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, AlreadyExists)
					break
				} else if update && !MachineLSEExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				// Name and Hostname of a MachineLSE must be same.
				machinelse.Name = value
				machinelse.Hostname = value
				input = &Input{
					Key:      "Broswer/ATL/ACS LAB",
					Desc:     fmt.Sprintf("%s%s", ChooseLab, BrowserOrATLOrACSLab),
					Required: true,
				}
			case "Broswer/ATL/ACS LAB":
				input = &Input{
					Key:      "Machines (y/n)",
					Desc:     fmt.Sprintf("%sMachine?", OptionToEnter),
					Required: true,
				}
				switch value {
				case "1":
					// Browser lab
					getPrototype(ctx, ic, scanner, machinelse, BrowserLab)
					getBrowserMachineLse(ctx, ic, scanner, machinelse)
				case "2":
					// ATL lab
					getPrototype(ctx, ic, scanner, machinelse, ATLLab)
					getOSMachineLse(ctx, ic, scanner, machinelse, false)
				case "3":
					// ACS lab
					getPrototype(ctx, ic, scanner, machinelse, ACSLab)
					getOSMachineLse(ctx, ic, scanner, machinelse, true)
				default:
					input = &Input{
						Key:      "Broswer/ATL/ACS LAB",
						Desc:     fmt.Sprintf("%s%s%s", WrongInput, ChooseLab, BrowserOrATLOrACSLab),
						Required: true,
					}
				}
			// repeated Machines
			case "Machines (y/n)":
				vals, done := getRepeatedStringInput(ctx, ic, scanner, value, "Machine", input, true)
				if done {
					machinelse.Machines = vals
					input = nil
				}
			}
			break
		}
	}
}

// getPrototype get MachineLSE prototype
//
// MachineLSEPrototype(selection) ->
func getPrototype(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE, lab string) {
	machineLSEPrototypes := getAllMachineLSEPrototypes(ctx, ic, lab)
	if len(machineLSEPrototypes) == 0 {
		return
	}
	input := &Input{
		Key:  "MachineLSEPrototype",
		Desc: fmt.Sprintf("%s%s", ChooseMachineLSEPrototype, createKeyValuePairs(machineLSEPrototypes)),
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "MachineLSEPrototype":
				if value != "" {
					option := getSelectionInput(value, machineLSEPrototypes, input)
					if option == -1 {
						break
					}
					machinelse.MachineLsePrototype = machineLSEPrototypes[option]
				}
				input = nil
			}
			break
		}
	}
}

// getOSMachineLse get ChormeOS MachineLSE input in interactive mode
//
// DUT, Labstation or Server(choice to branch) ->
// -> getOSDeviceLse()/getOSServerLse()
func getOSMachineLse(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE, acs bool) {
	machinelse.Lse = &fleet.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &fleet.ChromeOSMachineLSE{},
	}
	input := &Input{
		Key:      "DUT, Labstation or Server",
		Desc:     fmt.Sprintf("%s%s", ChooseOption, DutOrLabstationOrServer),
		Required: true,
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "DUT, Labstation or Server":
				if value == "1" {
					// ChromeOSDeviceLse - DUT
					getOSDeviceLse(ctx, ic, scanner, machinelse, acs, true)
					input = nil
				} else if value == "2" {
					// ChromeOSDeviceLse - Labstation
					getOSDeviceLse(ctx, ic, scanner, machinelse, acs, false)
					input = nil
				} else if value == "3" {
					// ChromeOSServerLse - Server
					getOSServerLse(ctx, ic, scanner, machinelse)
					input = nil
				} else {
					input.Desc = fmt.Sprintf("%s%s%s", WrongInput, ChooseOption, DutOrLabstationOrServer)
				}
			}
			break
		}
	}
}

// getOSDeviceLse get ChromeOSDeviceLSE input in interactive mode
//
// RPM(string, resource) -> RPM Port(int) -> Switch(string, resource) ->
// -> Switch Port(int) -> geDut()/getLabstation()
func getOSDeviceLse(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE, acs, dut bool) {
	deviceLse := &fleet.ChromeOSDeviceLSE{}
	machinelse.GetChromeosMachineLse().ChromeosLse = &fleet.ChromeOSMachineLSE_DeviceLse{
		DeviceLse: deviceLse,
	}
	input := &Input{
		Key: "RPM",
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			// ChromeOSDeviceLSE
			// RPMInterface
			case "RPM":
				if value != "" && !RPMExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				deviceLse.RpmInterface = &fleet.RPMInterface{
					Rpm: value,
				}
				if value != "" {
					input = &Input{
						Key: "RPM Port",
					}
				} else {
					input = &Input{
						Key: "Switch",
					}
				}
			case "RPM Port":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					deviceLse.GetRpmInterface().Port = port
				}
				input = &Input{
					Key: "Switch",
				}
			// SwitchInterface
			case "Switch":
				if value != "" && !SwitchExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				deviceLse.NetworkDeviceInterface = &fleet.SwitchInterface{
					Switch: value,
				}
				if value != "" {
					input = &Input{
						Key: "Switch Port",
					}
				} else {
					if dut {
						// DUT
						getDut(ctx, ic, scanner, machinelse, acs)
					} else {
						// Labstation
						getLabstation(ctx, ic, scanner, machinelse)
					}
					input = nil
				}
			case "Switch Port":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					deviceLse.GetNetworkDeviceInterface().Port = port
				}
				if dut {
					// DUT
					getDut(ctx, ic, scanner, machinelse, acs)
				} else {
					// Labstation
					getLabstation(ctx, ic, scanner, machinelse)
				}
				input = nil
			}
			break
		}
	}
}

// getDut get DeviceUnderTest input in interactive mode
//
// Servo Hostname(string) -> Servo Port(int) -> Servo Serial(string) ->
// -> Servo Type(string) -> RPM PowerunitName(string) ->
// -> RPM PowerunitOutlet(string) -> Pools(repeated string) -> getACSLabConfig()
func getDut(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE, acs bool) {
	dut := &chromeosLab.DeviceUnderTest{
		Hostname: machinelse.Hostname,
		Peripherals: &chromeosLab.Peripherals{
			Servo: &chromeosLab.Servo{},
			Rpm:   &chromeosLab.RPM{},
		},
	}
	machinelse.GetChromeosMachineLse().GetDeviceLse().Device = &fleet.ChromeOSDeviceLSE_Dut{
		Dut: dut,
	}
	input := &Input{
		Key: "Servo Hostname",
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			// DeviceUnderTest Dut Config
			// Peripherals
			// Servo
			case "Servo Hostname":
				if value != "" && !MachineLSEExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("Specified labstation %s is not "+
						"deployed. Please use deploy-machine command to "+
						"deploy the labstation %s first.", value, value)
					break
				}
				dut.GetPeripherals().GetServo().ServoHostname = value
				if value != "" {
					input = &Input{
						Key: "Servo Port",
					}
				} else {
					input = &Input{
						Key: "RPM PowerunitName",
					}
				}
			case "Servo Port":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					dut.GetPeripherals().GetServo().ServoPort = port
				}
				input = &Input{
					Key: "Servo Serial",
				}
			case "Servo Serial":
				// TODO(eshwarn) : this is available in Hart indexed by asset tag
				dut.GetPeripherals().GetServo().ServoSerial = value
				input = &Input{
					Key: "Servo Type",
				}
			case "Servo Type":
				// TODO(eshwarn) : this is available in Hart as google code name
				dut.GetPeripherals().GetServo().ServoType = value
				input = &Input{
					Key: "RPM PowerunitName",
				}
			case "RPM PowerunitName":
				dut.GetPeripherals().GetRpm().PowerunitName = value
				input = &Input{
					Key: "RPM PowerunitOutlet",
				}
			case "RPM PowerunitOutlet":
				dut.GetPeripherals().GetRpm().PowerunitOutlet = value
				input = &Input{
					Key:      "Pools (y/n)",
					Desc:     fmt.Sprintf("%sPool?", OptionToEnter),
					Required: true,
				}
			// repeated pools
			case "Pools (y/n)":
				vals, done := getRepeatedStringInput(nil, nil, scanner, value, "Pool", input, false)
				if done {
					dut.Pools = vals
					if acs {
						getACSLabConfig(scanner, machinelse)
					}
					input = nil
				}
			}
			break
		}
	}
}

// getACSLabConfig get ACSLab Config input in interactive mode
//
// Chameleon Peripherals(repeated enum) -> Audio Board(bool) ->
// -> Cameras(repeated enum) -> Audio Box(bool) -> Atrus(bool) ->
// -> Wificell(bool) -> AntennaConnection(enum) -> Router(enum) ->
// -> Touch Mimo(bool) -> Carrier(string) -> Camerabox(bool) ->
// -> Chaos(bool) -> Cable(repeated enum) -> Camerabox Info(enum)
func getACSLabConfig(scanner *bufio.Scanner, machinelse *fleet.MachineLSE) {
	peripherals := machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
	peripherals.Chameleon = &chromeosLab.Chameleon{}
	input := &Input{
		Key:      "Chameleon Peripherals (y/n)",
		Desc:     fmt.Sprintf("%sChameleon Peripheral?", OptionToEnter),
		Required: true,
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			// repeated enum Chameleon Peripherals
			case "Chameleon Peripherals (y/n)":
				vals, done := getRepeatedEnumInput(scanner, value, "Chameleon Peripheral", chromeosLab.ChameleonType_name, input)
				if done {
					cps := make([]chromeosLab.ChameleonType, 0, len(vals))
					for _, val := range vals {
						cps = append(cps, chromeosLab.ChameleonType(val))
					}
					peripherals.GetChameleon().ChameleonPeripherals = cps
					input = &Input{
						Key:      "Audio Board (y/n)",
						Required: true,
					}
				}
			// bool Audio Board
			case "Audio Board (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					peripherals.GetChameleon().AudioBoard = flag
					input = &Input{
						Key:      "Cameras (y/n)",
						Desc:     fmt.Sprintf("%sCamera?", OptionToEnter),
						Required: true,
					}
				}
			// repeated enum Camera
			case "Cameras (y/n)":
				vals, done := getRepeatedEnumInput(scanner, value, "Camera", chromeosLab.CameraType_name, input)
				if done {
					cameras := make([]*chromeosLab.Camera, 0, len(vals))
					for _, val := range vals {
						camera := &chromeosLab.Camera{
							CameraType: chromeosLab.CameraType(val),
						}
						cameras = append(cameras, camera)
					}
					peripherals.ConnectedCamera = cameras
					input = &Input{
						Key:      "Audio Box (y/n)",
						Required: true,
					}
				}
			// bool Audio Box
			case "Audio Box (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					peripherals.Audio = &chromeosLab.Audio{
						AudioBox: flag,
					}
					input = &Input{
						Key:      "Atrus (y/n)",
						Required: true,
					}
				}
			// bool Artus
			case "Atrus (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					peripherals.GetAudio().Atrus = flag
					input = &Input{
						Key:      "Wificell (y/n)",
						Required: true,
					}
				}
			// bool Wificell
			case "Wificell (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					peripherals.Wifi = &chromeosLab.Wifi{
						Wificell: flag,
					}
					input = &Input{
						Key:  "AntennaConnection",
						Desc: fmt.Sprintf("%s%s", ChooseAntennaConnection, createKeyValuePairs(chromeosLab.Wifi_AntennaConnection_name)),
					}
				}
			// enum AntennaConnection
			case "AntennaConnection":
				if value != "" {
					option := getSelectionInput(value, chromeosLab.Wifi_AntennaConnection_name, input)
					if option == -1 {
						break
					}
					peripherals.GetWifi().AntennaConn = chromeosLab.Wifi_AntennaConnection(option)
				}
				input = &Input{
					Key:  "Router",
					Desc: fmt.Sprintf("%s%s", ChooseRouter, createKeyValuePairs(chromeosLab.Wifi_Router_name)),
				}
			// enum Router
			case "Router":
				if value != "" {
					option := getSelectionInput(value, chromeosLab.Wifi_Router_name, input)
					if option == -1 {
						break
					}
					peripherals.GetWifi().Router = chromeosLab.Wifi_Router(option)
				}
				input = &Input{
					Key:      "Touch Mimo (y/n)",
					Required: true,
				}
			// bool Touch Mimo
			case "Touch Mimo (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					peripherals.Touch = &chromeosLab.Touch{
						Mimo: flag,
					}
					input = &Input{
						Key: "Carrier",
					}
				}
			case "Carrier":
				peripherals.Carrier = value
				input = &Input{
					Key: "Camerabox (y/n)",
				}
			// bool CameraBox
			case "Camerabox (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					peripherals.Camerabox = flag
					input = &Input{
						Key: "Chaos (y/n)",
					}
				}
			// bool chaos
			case "Chaos (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					peripherals.Chaos = flag
					input = &Input{
						Key:      "Cable (y/n)",
						Desc:     fmt.Sprintf("%sCable?", OptionToEnter),
						Required: true,
					}
				}
			// repeated enum Cable
			case "Cable (y/n)":
				vals, done := getRepeatedEnumInput(scanner, value, "Cable", chromeosLab.CableType_name, input)
				if done {
					cables := make([]*chromeosLab.Cable, 0, len(vals))
					for _, val := range vals {
						cable := &chromeosLab.Cable{
							Type: chromeosLab.CableType(val),
						}
						cables = append(cables, cable)
					}
					peripherals.Cable = cables
					input = &Input{
						Key:  "Camerabox Info",
						Desc: fmt.Sprintf("%s%s", ChooseCameraboxFacing, createKeyValuePairs(chromeosLab.Camerabox_Facing_name)),
					}
				}
			// enum CameraBox Info
			case "Camerabox Info":
				if value != "" {
					option := getSelectionInput(value, chromeosLab.Camerabox_Facing_name, input)
					if option == -1 {
						break
					}
					peripherals.CameraboxInfo = &chromeosLab.Camerabox{
						Facing: chromeosLab.Camerabox_Facing(option),
					}
				}
				input = nil
			}
			break
		}
	}
}

// getLabstation get Labstation input in interactive mode
//
// -> RPM PowerunitName(string) -> RPM PowerunitOutlet(string) ->
// -> Pools(repeated string) -> getServos()
func getLabstation(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE) {
	labstation := &chromeosLab.Labstation{
		// Hostname for a Labstation must be the same as the MachineLSE name.
		// MachineLSE Name is the MachineLSE Hostname
		// we use Labstation Hostname(ServoHostname) from a DUT to update the
		// Labstation with Servo info
		Hostname: machinelse.GetHostname(),
		Rpm:      &chromeosLab.RPM{},
	}
	machinelse.GetChromeosMachineLse().GetDeviceLse().Device = &fleet.ChromeOSDeviceLSE_Labstation{
		Labstation: labstation,
	}
	input := &Input{
		Key: "RPM PowerunitName",
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			// Labstation
			case "RPM PowerunitName":
				labstation.GetRpm().PowerunitName = value
				input = &Input{
					Key: "RPM PowerunitOutlet",
				}
			case "RPM PowerunitOutlet":
				labstation.GetRpm().PowerunitOutlet = value
				input = &Input{
					Key:      "Pools (y/n)",
					Desc:     fmt.Sprintf("%sPool?", OptionToEnter),
					Required: true,
				}
			// repeated pools
			case "Pools (y/n)":
				vals, done := getRepeatedStringInput(nil, nil, scanner, value, "Pool", input, false)
				if done {
					labstation.Pools = vals
					getServos(ctx, ic, machinelse)
					input = nil
				}
			}
			break
		}
	}
}

// getServos get the servos from existing MachineLSE
//
// MachineLSE Labstation update is not allowed to change the servo info in the
// Labstation, so during update call we get the existing servo info from the
// labstation and copy it to the input. For Create call we do nothing.
func getServos(ctx context.Context, ic UfleetAPI.FleetClient, machinelse *fleet.MachineLSE) {
	labstationMachineLse, _ := ic.GetMachineLSE(ctx, &UfleetAPI.GetMachineLSERequest{
		Name: UfleetUtil.AddPrefix(UfleetUtil.MachineLSECollection, machinelse.GetName()),
	})
	if labstationMachineLse != nil {
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = labstationMachineLse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	}
}

// getOSServerLse get ChromeOSServerLSE input in interactive mode
//
// Vlan(string, resource) ->
func getOSServerLse(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE) {
}

// getBrowserMachineLse get Browser MachineLSE input in interactive mode
//
// VM capacity(int) -> getVms()
func getBrowserMachineLse(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE) {
	machinelse.Lse = &fleet.MachineLSE_ChromeBrowserMachineLse{
		ChromeBrowserMachineLse: &fleet.ChromeBrowserMachineLSE{},
	}
	input := &Input{
		Key:      "VM Capactiy",
		Required: true,
		Desc:     "The maximum number of the VMs allowed on this Browser Machine LSE.",
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "VM Capactiy":
				if value != "" {
					capacity := getIntInput(value, input)
					if capacity == -1 {
						break
					}
					machinelse.GetChromeBrowserMachineLse().VmCapacity = capacity
				}
				getVms(ctx, ic, scanner, machinelse)
				input = nil
			}
			break
		}
	}
}

// getVms get Vms for Browser MachineLSE input in interactive mode
//
// -> VMs(repeated) -> VM Name(string) -> VM OS Version(string) ->
// -> VM OS Description(string) -> VM Mac Address(string) -> VM Hostname(string)
func getVms(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE) {
	maxVMAllowed := machinelse.GetChromeBrowserMachineLse().GetVmCapacity()
	if maxVMAllowed == 0 {
		fmt.Print("\nvm_capacity is 0. Increase vm_capacity to add VMs.\n")
		return
	}
	input := &Input{
		Key:      "VMs (y/n)",
		Desc:     fmt.Sprintf("%sVM?", OptionToEnter),
		Required: true,
	}
	vms := make([]*fleet.VM, 0, 0)
	var vm *fleet.VM
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			// ChromeBrowserMachineLSE
			// repeated VMs
			case "VMs (y/n)":
				value = strings.ToLower(value)
				if value == "y" {
					input = &Input{
						Key: "VM Name",
					}
				} else if value == "n" {
					input = nil
				} else {
					input = &Input{
						Key:      "VMs (y/n)",
						Desc:     fmt.Sprintf("%s%sVM?", WrongInput, OptionToEnter),
						Required: true,
					}
				}
			case "VM Name":
				vm = &fleet.VM{
					Name: value,
				}
				input = &Input{
					Key: "VM OS Version",
				}
			case "VM OS Version":
				vm.OsVersion = &fleet.OSVersion{
					Value: value,
				}
				input = &Input{
					Key: "VM OS Description",
				}
			case "VM OS Description":
				vm.GetOsVersion().Description = value
				input = &Input{
					Key: "VM Mac Address",
				}
			case "VM Mac Address":
				vm.MacAddress = value
				input = &Input{
					Key: "VM Hostname",
				}
			case "VM Hostname":
				vm.Hostname = value
				vms = append(vms, vm)
				machinelse.GetChromeBrowserMachineLse().Vms = vms
				if len(vms) == int(maxVMAllowed) {
					fmt.Print("\nYou have added the maximum VMs for this MachineLSE.\n" +
						"If you want to add more please increase the vm_capacity.\n")
					return
				}
				input = &Input{
					Key:      "VMs (y/n)",
					Desc:     fmt.Sprintf("%sVM?", OptionToEnterMore),
					Required: true,
				}
			}
			break
		}
	}
}

// GetMachinelsePrototypeInteractiveInput gets MachineLSEPrototype input in interactive mode
//
// Name(string) -> Broswer/ATL/ACS LAB(choice) ->Occupied Capacity(int) ->
// -> getPeripheralRequirements() -> getVirtualRequirements()
func GetMachinelsePrototypeInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, mlsep *fleet.MachineLSEPrototype, update bool) {
	input := &Input{
		Key:      "Broswer/ATL/ACS LAB",
		Desc:     fmt.Sprintf("%s%s", ChooseLab, BrowserOrATLOrACSLab),
		Required: true,
	}
	var prefix string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(InputDetails)
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "Broswer/ATL/ACS LAB":
				input = &Input{
					Key:      "Name",
					Desc:     UfleetAPI.ValidName,
					Required: true,
				}
				switch value {
				case "1":
					// Browser lab
					prefix = BrowserLab
				case "2":
					// ATL lab
					prefix = ATLLab
				case "3":
					// ACS lab
					prefix = ACSLab
				default:
					input = &Input{
						Key:      "Broswer/ATL/ACS LAB",
						Desc:     fmt.Sprintf("%s%s%s", WrongInput, ChooseLab, BrowserOrATLOrACSLab),
						Required: true,
					}
				}
			case "Name":
				if !UfleetAPI.IDRegex.MatchString(value) {
					input.Desc = UfleetAPI.ValidName
					break
				}
				if !update && MachineLSEPrototypeExists(ctx, ic, prefix+value) {
					input.Desc = fmt.Sprintf("%s%s", value, AlreadyExists)
					break
				} else if update && !MachineLSEPrototypeExists(ctx, ic, prefix+value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				mlsep.Name = prefix + value
				input = &Input{
					Key: "Occupied Capacity",
					Desc: "Indicates the Rack Unit capacity of this setup, " +
						"corresponding to a Rack’s Rack Unit capacity.",
				}
			case "Occupied Capacity":
				if value != "" {
					val := getIntInput(value, input)
					if val == -1 {
						break
					}
					mlsep.OccupiedCapacityRu = val
				}
				getPeripheralRequirements(scanner, mlsep)
				getVirtualRequirements(scanner, mlsep)
				input = nil
			}
			break
		}
	}
}

// getPeripheralRequirements get PeripheralRequirements for MachineLSEPrototype input in interactive mode
//
// PeripheralRequirements(repeated) -> PeripheralType(enum) -> min(int) -> max(int)
func getPeripheralRequirements(scanner *bufio.Scanner, mlsep *fleet.MachineLSEPrototype) {
	input := &Input{
		Key:      "PeripheralRequirements (y/n)",
		Desc:     fmt.Sprintf("%sPeripheralRequirement?", OptionToEnter),
		Required: true,
	}
	prs := make([]*fleet.PeripheralRequirement, 0, 0)
	var pr *fleet.PeripheralRequirement
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			// repeated PeripheralRequirements
			case "PeripheralRequirements (y/n)":
				value = strings.ToLower(value)
				if value == "y" {
					input = &Input{
						Key:  "PeripheralType",
						Desc: fmt.Sprintf("%s%s", ChoosePheripheralType, createKeyValuePairs(fleet.PeripheralType_name)),
					}
				} else if value == "n" {
					mlsep.PeripheralRequirements = prs
					input = nil
				} else {
					input = &Input{
						Key:      "PeripheralRequirements (y/n)",
						Desc:     fmt.Sprintf("%s%sPeripheralRequirement?", WrongInput, OptionToEnter),
						Required: true,
					}
				}
			case "PeripheralType":
				if value == "" || value == "0" {
					pr = &fleet.PeripheralRequirement{}
				} else {
					option := getSelectionInput(value, fleet.PeripheralType_name, input)
					if option == -1 {
						break
					}
					pr = &fleet.PeripheralRequirement{
						PeripheralType: fleet.PeripheralType(option),
					}
				}
				input = &Input{
					Key: "Minimum Pheripherals",
					Desc: "The minimum/maximum number of the peripherals " +
						"that is needed by a LSE, e.g. A test needs 1-3 bluetooth " +
						"bt peers to be set up.",
				}
			case "Minimum Pheripherals":
				if value != "" {
					val := getIntInput(value, input)
					if val == -1 {
						break
					}
					pr.Min = val
				}
				input = &Input{
					Key: "Maximum Pheripherals",
				}
			case "Maximum Pheripherals":
				if value != "" {
					val := getIntInput(value, input)
					if val == -1 {
						break
					}
					if val < pr.Min {
						input.Desc = fmt.Sprintf("%s%s", WrongInput, MinMaxError)
						break
					}
					pr.Max = val
				}
				prs = append(prs, pr)
				input = &Input{
					Key:      "PeripheralRequirements (y/n)",
					Desc:     fmt.Sprintf("%sPeripheralRequirement?", OptionToEnterMore),
					Required: true,
				}
			}
			break
		}
	}
}

// getVirtualRequirements get VirtualRequirements for MachineLSEPrototype input in interactive mode
//
// VirtualRequirements(repeated) -> VirtualType(enum) -> min(int) -> max(int)
func getVirtualRequirements(scanner *bufio.Scanner, mlsep *fleet.MachineLSEPrototype) {
	input := &Input{
		Key:      "VirtualRequirements (y/n)",
		Desc:     fmt.Sprintf("%sVirtualRequirement?", OptionToEnter),
		Required: true,
	}
	prs := make([]*fleet.VirtualRequirement, 0, 0)
	var pr *fleet.VirtualRequirement
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			// repeated VirtualRequirements
			case "VirtualRequirements (y/n)":
				value = strings.ToLower(value)
				if value == "y" {
					input = &Input{
						Key:  "VirtualType",
						Desc: fmt.Sprintf("%s%s", ChooseVirtualType, createKeyValuePairs(fleet.VirtualType_name)),
					}
				} else if value == "n" {
					mlsep.VirtualRequirements = prs
					input = nil
				} else {
					input = &Input{
						Key:      "VirtualRequirements (y/n)",
						Desc:     fmt.Sprintf("%s%sVirtualRequirement?", WrongInput, OptionToEnter),
						Required: true,
					}
				}
			case "VirtualType":
				if value == "" || value == "0" {
					pr = &fleet.VirtualRequirement{}
				} else {
					option := getSelectionInput(value, fleet.VirtualType_name, input)
					if option == -1 {
						break
					}
					pr = &fleet.VirtualRequirement{
						VirtualType: fleet.VirtualType(option),
					}
				}
				input = &Input{
					Key:  "Minimum",
					Desc: "The minimum/maximum number of virtual types that can be setup.",
				}
			case "Minimum":
				if value != "" {
					val := getIntInput(value, input)
					if val == -1 {
						break
					}
					pr.Min = val
				}
				input = &Input{
					Key: "Maximum",
				}
			case "Maximum":
				if value != "" {
					val := getIntInput(value, input)
					if val == -1 {
						break
					}
					if val < pr.Min {
						input.Desc = fmt.Sprintf("%s%s", WrongInput, MinMaxError)
						break
					}
					pr.Max = val
				}
				prs = append(prs, pr)
				input = &Input{
					Key:      "VirtualRequirements (y/n)",
					Desc:     fmt.Sprintf("%sVirtualRequirement?", OptionToEnterMore),
					Required: true,
				}
			}
			break
		}
	}
}

func createKeyValuePairs(m map[int32]string) string {
	keys := make([]int32, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	var s string
	for _, k := range keys {
		s = fmt.Sprintf("%s%d = \"%s\"\n", s, k, m[k])
	}
	return s
}

func getLab(lab fleet.Lab) string {
	switch lab {
	case fleet.Lab_LAB_CHROME_ATLANTA,
		fleet.Lab_LAB_DATACENTER_ATL97,
		fleet.Lab_LAB_DATACENTER_IAD97,
		fleet.Lab_LAB_DATACENTER_MTV96,
		fleet.Lab_LAB_DATACENTER_MTV97,
		fleet.Lab_LAB_DATACENTER_FUCHSIA:
		return Browser
	case fleet.Lab_LAB_CHROMEOS_DESTINY,
		fleet.Lab_LAB_CHROMEOS_PROMETHEUS,
		fleet.Lab_LAB_CHROMEOS_ATLANTIS:
		return ATL
	case fleet.Lab_LAB_CHROMEOS_LINDAVISTA:
		return ACS
	default:
		return Unknown
	}
}

// getAllChromePlatforms gets all ChromePlatforms in the system
func getAllChromePlatforms(ctx context.Context, ic UfleetAPI.FleetClient) map[int32]string {
	m := make(map[int32]string)
	var pageToken string
	var index int32
	for {
		req := &UfleetAPI.ListChromePlatformsRequest{
			PageSize:  int32(maxPageSize),
			PageToken: pageToken,
		}
		res, err := ic.ListChromePlatforms(ctx, req)
		if err != nil {
			return m
		}
		for _, cp := range res.GetChromePlatforms() {
			m[index] = UfleetUtil.RemovePrefix(cp.GetName())
			index++
		}
		pageToken = res.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return m
}

// ParseJSONFile parses json input from the user provided file.
func ParseJSONFile(jsonFile string, pm proto.Message) error {
	rawText, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return errors.Annotate(err, "parse json file").Err()
	}
	err = jsonpb.Unmarshal(strings.NewReader(string(rawText)), pm)
	return err
}

// GetNextPage gets user input for to get next page of items
func GetNextPage(pageToken string) (bool, error) {
	if pageToken == "" {
		fmt.Println("End of list.")
		return false, nil
	}
	fmt.Println("Press Enter to get next page of items.")
	b := bufio.NewReaderSize(os.Stdin, 1)
	input, err := b.ReadByte()
	if err != nil {
		return false, errors.Annotate(err, "Error in getting input").Err()
	}
	// Ctrl-C
	if input == 3 {
		fmt.Println("Exiting...")
		return false, nil
	}
	return true, nil
}

func getBoolInput(value string, input *Input) (bool, bool) {
	value = strings.ToLower(value)
	if value != "y" && value != "n" {
		input.Desc = fmt.Sprintf("%s", WrongInput)
		return false, false
	}
	// User has entered some valid input so we can go to the next input field
	if value == "y" {
		return true, true
	}
	return false, true
}

func getIntInput(value string, input *Input) int32 {
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		input.Desc = fmt.Sprintf("%s", WrongInput)
		return -1
	}
	return int32(i)
}

func getSelectionInput(value string, m map[int32]string, input *Input) int32 {
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil || i < 0 || int(i) >= len(m) {
		input.Desc = fmt.Sprintf("%s%s%s", WrongInput, "\nChoose a "+input.Key+"\n", createKeyValuePairs(m))
		return -1
	}
	return int32(i)
}

func getRepeatedStringInput(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, yn, key string, input *Input, isResource bool) ([]string, bool) {
	yn = strings.ToLower(yn)
	if yn != "y" && yn != "n" {
		input.Desc = fmt.Sprintf("%s%s%s?", WrongInput, OptionToEnter, input.Key)
		return nil, false
	}
	if yn == "n" {
		return nil, true
	}
	values := make([]string, 0, 0)
	input = &Input{
		Key: key,
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case key + YesNo:
				value = strings.ToLower(value)
				if value == "y" {
					input = &Input{
						Key: key,
					}
				} else if value == "n" {
					input = nil
				} else {
					input.Desc = fmt.Sprintf("%s%s%s?", WrongInput, OptionToEnter, key)
				}
			case key:
				if isResource && value != "" && !EntityExists(ctx, ic, key, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				if value != "" {
					values = append(values, value)
				}
				input = &Input{
					Key:      key + YesNo,
					Desc:     fmt.Sprintf("%s%s?", OptionToEnterMore, key),
					Required: true,
				}
			}
			break
		}
	}
	return values, true
}

func getRepeatedEnumInput(scanner *bufio.Scanner, yn, key string, m map[int32]string, input *Input) ([]int32, bool) {
	yn = strings.ToLower(yn)
	if yn != "y" && yn != "n" {
		input.Desc = fmt.Sprintf("%s%s%s?", WrongInput, OptionToEnter, key)
		return nil, false
	}
	if yn == "n" {
		return nil, true
	}
	values := make([]int32, 0, 0)
	input = &Input{
		Key:      key,
		Desc:     fmt.Sprintf("%s%s", "\nChoose a "+key+"\n", createKeyValuePairs(m)),
		Required: true,
	}
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case key + YesNo:
				value = strings.ToLower(value)
				if value == "y" {
					input = &Input{
						Key:      key,
						Desc:     fmt.Sprintf("%s%s", "\nChoose a "+key+"\n", createKeyValuePairs(m)),
						Required: true,
					}
				} else if value == "n" {
					input = nil
				} else {
					input.Desc = fmt.Sprintf("%s%s%s?", WrongInput, OptionToEnter, key)
				}
			case key:
				option := getSelectionInput(value, m, input)
				if option == -1 {
					break
				}
				values = append(values, option)
				input = &Input{
					Key:      key + YesNo,
					Desc:     fmt.Sprintf("%s%s?", OptionToEnterMore, key),
					Required: true,
				}
			}
			break
		}
	}
	return values, true
}

// getAllMachineLSEPrototypes gets all MachineLSEPrototypes in the system
func getAllMachineLSEPrototypes(ctx context.Context, ic UfleetAPI.FleetClient, lab string) map[int32]string {
	m := make(map[int32]string)
	var pageToken string
	var index int32
	for {
		req := &UfleetAPI.ListMachineLSEPrototypesRequest{
			PageSize:  int32(maxPageSize),
			PageToken: pageToken,
		}
		res, err := ic.ListMachineLSEPrototypes(ctx, req)
		if err != nil {
			return m
		}
		for _, cp := range res.GetMachineLSEPrototypes() {
			name := UfleetUtil.RemovePrefix(cp.GetName())
			if strings.Contains(name, lab) {
				m[index] = name
				index++
			}
		}
		pageToken = res.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return m
}
