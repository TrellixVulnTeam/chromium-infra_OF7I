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
	chromeos "infra/unifiedfleet/api/v1/proto/chromeos/lab"
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
	OptionToEnter             string = "\nDo you want to enter a "
	OptionToEnterMore         string = "\nDo you want to enter one more "
	ChooseLab                 string = "\n Choose a Lab\n"
	BroswerOrOSLab            string = "1=\"Browser Lab\"\n2=\"OS Lab\"\n"
	BrowserOrATLOrACSLab      string = "1=\"Browser Lab\"\n2=\"ATL Lab\"\n3=\"ACS Lab\"\n"
	DutOrServer               string = "1=\"DUT\"\n2=\"Server\"\n"
	DoesNotExist              string = " doesnt not exist in the system. Please check and enter again."
	AlreadyExists             string = " already exists in the system. Please check and enter again."
	ATL                       string = "ATL"
	ACS                       string = "ACS"
	Browser                   string = "Browser"
	Unknown                   string = "Unknown"
	maxPageSize               int32  = 1000
	YesNo                     string = " (y/n)"
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
func GetSwitchInteractiveInput(s *fleet.Switch) {
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
				s.Name = value
				input = &Input{
					Key: "CapacityPort",
				}
			case "CapacityPort":
				if value != "" {
					i, err := strconv.ParseInt(value, 10, 32)
					if err != nil {
						input = &Input{
							Key:  "CapacityPort",
							Desc: fmt.Sprintf("%s", WrongInput),
						}
						break
					}
					s.CapacityPort = int32(i)
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
// -> getBrowserMachineInteractiveInput()/getOSMachineInteractiveInput() ->
// -> Realm(string)
func GetMachineInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, machine *fleet.Machine) {
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
				if MachineExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, AlreadyExists)
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
					machine.Location = &fleet.Location{
						Lab: fleet.Lab(option),
					}
					input = &Input{
						Key: "Realm",
					}
					if getLab(machine.Location.Lab) == Browser {
						// Chorome Browser lab
						getBrowserMachineInteractiveInput(ctx, ic, scanner, machine)
					} else if getLab(machine.Location.Lab) == ACS ||
						getLab(machine.Location.Lab) == ATL {
						// ChromeOS lab
						getOSMachineInteractiveInput(ctx, ic, scanner, machine)
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
					getBrowserMachineInteractiveInput(ctx, ic, scanner, machine)
					input = &Input{
						Key: "Realm",
					}
				} else if value == "2" {
					// Chrome OS lab
					getOSMachineInteractiveInput(ctx, ic, scanner, machine)
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

// getOSMachineInteractiveInput get Chrome OS Machine input in interactive mode
//
// Rack(string) -> Aisle(string) -> Row(string) -> Rack Number(string) ->
// -> Shelf(string) -> Position(string)
func getOSMachineInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machine *fleet.Machine) {
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

// getBrowserMachineInteractiveInput get Browser Machine input in interactive mode
//
// Rack(string, resource) -> DisplayName(string) ->
// -> ChromePlatform(string, resource) -> Nics(repeated string, resource) ->
// -> KVM(string, resource) -> KVM Port(int) -> RPM(string, resource) ->
// -> RPM Port(int) -> Switch(string, resource) -> Switch Port(int) ->
// -> Drac(string, resource) -> DeploymentTicket(string) -> Description(string)
func getBrowserMachineInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machine *fleet.Machine) {
	machine.Device = &fleet.Machine_ChromeBrowserMachine{
		ChromeBrowserMachine: &fleet.ChromeBrowserMachine{},
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
				machine.GetChromeBrowserMachine().DisplayName = value
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
					machine.GetChromeBrowserMachine().ChromePlatform = chromePlatforms[option]
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
					machine.GetChromeBrowserMachine().Nics = vals
					input = &Input{
						Key: "KVM",
					}
				}
			case "KVM":
				if value != "" && !KVMExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				machine.GetChromeBrowserMachine().KvmInterface = &fleet.KVMInterface{
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
					machine.GetChromeBrowserMachine().KvmInterface.Port = port
				}
				input = &Input{
					Key: "RPM",
				}
			case "RPM":
				if value != "" && !RPMExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
					break
				}
				machine.GetChromeBrowserMachine().RpmInterface = &fleet.RPMInterface{
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
					machine.GetChromeBrowserMachine().RpmInterface.Port = port
				}
				input = &Input{
					Key: "Drac",
				}
			case "Drac":
				if value != "" && !DracExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, DoesNotExist)
				} else {
					machine.GetChromeBrowserMachine().Drac = value
					input = &Input{
						Key: "DeploymentTicket",
					}
				}
			case "DeploymentTicket":
				machine.GetChromeBrowserMachine().DeploymentTicket = value
				input = &Input{
					Key: "Description",
				}
			case "Description":
				machine.GetChromeBrowserMachine().Description = value
				input = nil
			}
			break
		}
	}
}

// GetMachinelseInteractiveInput get MachineLSE input in interactive mode
//
// Name(string) -> Broswer/ATL/ACS LAB(choice to branch) ->
// -> getBrowserMachinelseInteractiveInput()/getOSMachinelseInteractiveInput() ->
// -> Machine(repeated string, resource)
func GetMachinelseInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, machinelse *fleet.MachineLSE) {
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
				// TODO(eshwarn) : Have a different format for Machinelse names
				if !UfleetAPI.IDRegex.MatchString(value) {
					input.Desc = UfleetAPI.ValidName
					break
				}
				if MachineLSEExists(ctx, ic, value) {
					input.Desc = fmt.Sprintf("%s%s", value, AlreadyExists)
					break
				}
				machinelse.Name = value
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
					getBrowserMachinelseInteractiveInput(ctx, ic, scanner, machinelse)
				case "2":
					// ATL lab
					getOSMachinelseInteractiveInput(ctx, ic, scanner, machinelse, false)
				case "3":
					// ACS lab
					getOSMachinelseInteractiveInput(ctx, ic, scanner, machinelse, true)
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

// getOSMachinelseInteractiveInput get ChormeOS MachineLSE input in interactive mode
//
// Hostname(string) -> MachineLSEPrototype(string, resource) ->
// -> DUT or Server(choice to branch) ->
// -> getOSDevicelseInteractiveInput()/getOSServerlseInteractiveInput()
func getOSMachinelseInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE, acs bool) {
	machinelse.Lse = &fleet.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &fleet.ChromeOSMachineLSE{},
	}
	var machineLSEPrototypes map[int32]string
	if acs {
		machineLSEPrototypes = getAllMachineLSEPrototypes(ctx, ic, "acs-lab:")
	} else {
		machineLSEPrototypes = getAllMachineLSEPrototypes(ctx, ic, "atl-lab:")
	}
	input := &Input{
		Key: "Hostname",
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
			case "Hostname":
				machinelse.Hostname = value
				if len(machineLSEPrototypes) == 0 {
					getOSDevicelseInteractiveInput(ctx, ic, scanner, machinelse, acs)
					input = nil
				} else {
					input = &Input{
						Key:  "MachineLSEPrototype",
						Desc: fmt.Sprintf("%s%s", ChooseMachineLSEPrototype, createKeyValuePairs(machineLSEPrototypes)),
					}
				}
			case "MachineLSEPrototype":
				if value != "" {
					option := getSelectionInput(value, machineLSEPrototypes, input)
					if option == -1 {
						break
					}
					machinelse.MachineLsePrototype = machineLSEPrototypes[option]
				}
				getOSDevicelseInteractiveInput(ctx, ic, scanner, machinelse, acs)
				input = nil
			}
			break
		}
	}
}

// getOSDevicelseInteractiveInput get ChromeOSDeviceLSE input in interactive mode
//
// RPM(string, resource) -> RPM Port(int) -> Switch(string, resource) ->
// -> Switch Port(int) -> Servo Hostname(string) ->
// -> Servo Port(int) -> Servo Serial(string) -> Servo Type(string) ->
// -> Pools(repeated string) -> getACSLabConfig()
func getOSDevicelseInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE, acs bool) {
	machinelse.GetChromeosMachineLse().ChromeosLse = &fleet.ChromeOSMachineLSE_Dut{
		Dut: &fleet.ChromeOSDeviceLSE{},
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
				machinelse.GetChromeosMachineLse().GetDut().RpmInterface = &fleet.RPMInterface{
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
					machinelse.GetChromeosMachineLse().GetDut().GetRpmInterface().Port = port
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
				machinelse.GetChromeosMachineLse().GetDut().NetworkDeviceInterface = &fleet.SwitchInterface{
					Switch: value,
				}
				if value != "" {
					input = &Input{
						Key: "Switch Port",
					}
				} else {
					input = &Input{
						Key: "Servo Hostname",
					}
				}
			case "Switch Port":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					machinelse.GetChromeosMachineLse().GetDut().GetNetworkDeviceInterface().Port = port
				}
				input = &Input{
					Key: "Servo Hostname",
				}
			// DeviceUnderTest Dut Config
			// Pheripherals
			// Servo
			case "Servo Hostname":
				machinelse.GetChromeosMachineLse().GetDut().Config = &chromeos.DeviceUnderTest{
					Hostname: machinelse.Hostname,
				}
				machinelse.GetChromeosMachineLse().GetDut().GetConfig().Peripherals = &chromeos.Peripherals{
					Servo: &chromeos.Servo{
						ServoHostname: value,
					},
					Rpm: &chromeos.RPM{
						PowerunitName:   machinelse.GetChromeosMachineLse().GetDut().GetRpmInterface().GetRpm(),
						PowerunitOutlet: fmt.Sprint(machinelse.GetChromeosMachineLse().GetDut().GetRpmInterface().GetPort()),
					},
				}
				input = &Input{
					Key: "Servo Port",
				}
			case "Servo Port":
				if value != "" {
					port := getIntInput(value, input)
					if port == -1 {
						break
					}
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetServo().ServoPort = port
				}
				input = &Input{
					Key: "Servo Serial",
				}
			case "Servo Serial":
				// TODO(eshwarn) : this is available in Hart indexed by asset tag
				machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetServo().ServoSerial = value
				input = &Input{
					Key: "Servo Type",
				}
			case "Servo Type":
				// TODO(eshwarn) : this is available in Hart as google code name
				machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetServo().ServoType = value
				input = &Input{
					Key:      "Pools (y/n)",
					Desc:     fmt.Sprintf("%sPool?", OptionToEnter),
					Required: true,
				}
			// repeated pools
			case "Pools (y/n)":
				vals, done := getRepeatedStringInput(ctx, ic, scanner, value, "Pool", input, false)
				if done {
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().Pools = vals
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
	machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Chameleon = &chromeos.Chameleon{}
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
				vals, done := getRepeatedEnumInput(scanner, value, "Chameleon Peripheral", chromeos.ChameleonType_name, input)
				if done {
					cps := make([]chromeos.ChameleonType, 0, len(vals))
					for _, val := range vals {
						cps = append(cps, chromeos.ChameleonType(val))
					}
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetChameleon().ChameleonPeripherals = cps
					input = &Input{
						Key:      "Audio Board (y/n)",
						Required: true,
					}
				}
			// bool Audio Board
			case "Audio Board (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetChameleon().AudioBoard = flag
					input = &Input{
						Key:      "Cameras (y/n)",
						Desc:     fmt.Sprintf("%sCamera?", OptionToEnter),
						Required: true,
					}
				}
			// repeated enum Camera
			case "Cameras (y/n)":
				vals, done := getRepeatedEnumInput(scanner, value, "Camera", chromeos.CameraType_name, input)
				if done {
					cameras := make([]*chromeos.Camera, 0, len(vals))
					for _, val := range vals {
						camera := &chromeos.Camera{
							CameraType: chromeos.CameraType(val),
						}
						cameras = append(cameras, camera)
					}
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().ConnectedCamera = cameras
					input = &Input{
						Key:      "Audio Box (y/n)",
						Required: true,
					}
				}
			// bool Audio Box
			case "Audio Box (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Audio = &chromeos.Audio{
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
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetAudio().Atrus = flag
					input = &Input{
						Key:      "Wificell (y/n)",
						Required: true,
					}
				}
			// bool Wificell
			case "Wificell (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Wifi = &chromeos.Wifi{
						Wificell: flag,
					}
					input = &Input{
						Key:  "AntennaConnection",
						Desc: fmt.Sprintf("%s%s", ChooseAntennaConnection, createKeyValuePairs(chromeos.Wifi_AntennaConnection_name)),
					}
				}
			// enum AntennaConnection
			case "AntennaConnection":
				if value != "" {
					option := getSelectionInput(value, chromeos.Wifi_AntennaConnection_name, input)
					if option == -1 {
						break
					}
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetWifi().AntennaConn = chromeos.Wifi_AntennaConnection(option)
				}
				input = &Input{
					Key:  "Router",
					Desc: fmt.Sprintf("%s%s", ChooseRouter, createKeyValuePairs(chromeos.Wifi_Router_name)),
				}
			// enum Router
			case "Router":
				if value != "" {
					option := getSelectionInput(value, chromeos.Wifi_Router_name, input)
					if option == -1 {
						break
					}
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().GetWifi().Router = chromeos.Wifi_Router(option)
				}
				input = &Input{
					Key:      "Touch Mimo (y/n)",
					Required: true,
				}
			// bool Touch Mimo
			case "Touch Mimo (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Touch = &chromeos.Touch{
						Mimo: flag,
					}
					input = &Input{
						Key: "Carrier",
					}
				}
			case "Carrier":
				machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Carrier = value
				input = &Input{
					Key: "Camerabox (y/n)",
				}
			// bool CameraBox
			case "Camerabox (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Camerabox = flag
					input = &Input{
						Key: "Chaos (y/n)",
					}
				}
			// bool chaos
			case "Chaos (y/n)":
				flag, done := getBoolInput(value, input)
				if done {
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Chaos = flag
					input = &Input{
						Key:      "Cable (y/n)",
						Desc:     fmt.Sprintf("%sCable?", OptionToEnter),
						Required: true,
					}
				}
			// repeated enum Cable
			case "Cable (y/n)":
				vals, done := getRepeatedEnumInput(scanner, value, "Cable", chromeos.CableType_name, input)
				if done {
					cables := make([]*chromeos.Cable, 0, len(vals))
					for _, val := range vals {
						cable := &chromeos.Cable{
							Type: chromeos.CableType(val),
						}
						cables = append(cables, cable)
					}
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().Cable = cables
					input = &Input{
						Key:  "Camerabox Info",
						Desc: fmt.Sprintf("%s%s", ChooseCameraboxFacing, createKeyValuePairs(chromeos.Camerabox_Facing_name)),
					}
				}
			// enum CameraBox Info
			case "Camerabox Info":
				if value != "" {
					option := getSelectionInput(value, chromeos.Camerabox_Facing_name, input)
					if option == -1 {
						break
					}
					machinelse.GetChromeosMachineLse().GetDut().GetConfig().GetPeripherals().CameraboxInfo = &chromeos.Camerabox{
						Facing: chromeos.Camerabox_Facing(option),
					}
				}
				input = nil
			}
			break
		}
	}
}

// getBrowserMachinelseInteractiveInput get Browser MachineLSE input in interactive mode
//
// Hostname(string) -> MachineLSEPrototype(string, resource) -> getVms()
func getBrowserMachinelseInteractiveInput(ctx context.Context, ic UfleetAPI.FleetClient, scanner *bufio.Scanner, machinelse *fleet.MachineLSE) {
	machinelse.Lse = &fleet.MachineLSE_ChromeBrowserMachineLse{
		ChromeBrowserMachineLse: &fleet.ChromeBrowserMachineLSE{},
	}
	input := &Input{
		Key: "Hostname",
	}
	machineLSEPrototypes := getAllMachineLSEPrototypes(ctx, ic, "browser-lab:")
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
				machinelse.Hostname = value
				if len(machineLSEPrototypes) == 0 {
					getVms(ctx, ic, scanner, machinelse)
					input = nil
				} else {
					input = &Input{
						Key:  "MachineLSEPrototype",
						Desc: fmt.Sprintf("%s%s", ChooseMachineLSEPrototype, createKeyValuePairs(machineLSEPrototypes)),
					}
				}
			case "MachineLSEPrototype":
				if value != "" {
					option := getSelectionInput(value, machineLSEPrototypes, input)
					if option == -1 {
						break
					}
					machinelse.MachineLsePrototype = machineLSEPrototypes[option]
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
	pr := getMachineLSEPrototype(ctx, ic, machinelse.GetMachineLsePrototype())
	vrs := pr.GetVirtualRequirements()
	var maxVM int
	for _, vr := range vrs {
		if vr.GetVirtualType() == fleet.VirtualType_VIRTUAL_TYPE_VM {
			maxVM = int(vr.GetMax())
			break
		}
	}
	if maxVM == 0 {
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
				if len(vms) == maxVM {
					fmt.Print("\nYou have added the maximum VMs for this "+
						"MachineLSE as defined by MachineLSEPrototype ",
						machinelse.GetMachineLsePrototype()+"\n")
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
