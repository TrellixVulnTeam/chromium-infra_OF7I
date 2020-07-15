// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmdhelp

var (
	// ListPageSizeDesc description for List PageSize
	ListPageSizeDesc string = `number of items to get. The service may return fewer than this value.`

	// RegisterSwitchLongDesc long description for RegisterSwitchCmd
	RegisterSwitchLongDesc string = `Register a switch by name.

Examples:
shivas register switch -f switch.json
Register a switch by reading a JSON file input.

shivas register switch -i
Register a switch by reading input through interactive mode.`

	// ReregisterSwitchLongDesc long description for ReregisterSwitchCmd
	ReregisterSwitchLongDesc string = `Reregister/Update a switch by name.

Examples:
shivas reregister switch -f switch.json
Reregister/Update a switch by reading a JSON file input.

shivas reregister switch -i
Reregister/Update a switch by reading input through interactive mode.`

	// ListSwitchLongDesc long description for ListSwitchCmd
	ListSwitchLongDesc string = `List all switches

Examples:
shivas ls switch
Fetches all switches and prints the output in table format

shivas ls switch -n 50
Fetches 50 switches and prints the output in table format

shivas ls switch -json
Fetches all switches and prints the output in JSON format

shivas ls switch -n 50 -json
Fetches 50 switches and prints the output in JSON format
`

	// SwitchFileText description for switch file input
	SwitchFileText string = `Path to a file containing switch specification in JSON format.
This file must contain one switch JSON message

Example switch:
{
    "name": "switch-test-example",
    "capacityPort": 456
}

The protobuf definition of switch is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto#71`

	// ListKVMLongDesc long description for ListKVMCmd
	ListKVMLongDesc string = `List all kvms

Examples:
shivas ls kvm
Fetches all kvms and prints the output in table format

shivas ls kvm -n 50
Fetches 50 kvms and prints the output in table format

shivas ls kvm -json
Fetches all kvms and prints the output in JSON format

shivas ls kvm -n 50 -json
Fetches 50 kvms and prints the output in JSON format
`

	// ListRPMLongDesc long description for ListRPMCmd
	ListRPMLongDesc string = `List all rpms

Examples:
shivas ls rpm
Fetches all rpms and prints the output in table format

shivas ls rpm -n 50
Fetches 50 rpms and prints the output in table format

shivas ls rpm -json
Fetches all rpms and prints the output in JSON format

shivas ls rpm -n 50 -json
Fetches 50 rpms and prints the output in JSON format
`

	// ListDracLongDesc long description for ListDracCmd
	ListDracLongDesc string = `List all dracs

Examples:
shivas ls drac
Fetches all dracs and prints the output in table format

shivas ls drac -n 50
Fetches 50 dracs and prints the output in table format

shivas ls drac -json
Fetches all dracs and prints the output in JSON format

shivas ls drac -n 50 -json
Fetches 50 dracs and prints the output in JSON format
`

	// ListNicLongDesc long description for ListNicCmd
	ListNicLongDesc string = `List all nics

Examples:
shivas ls nic
Fetches all nics and prints the output in table format

shivas ls nic -n 50
Fetches 50 nics and prints the output in table format

shivas ls nic -json
Fetches all nics and prints the output in JSON format

shivas ls nic -n 50 -json
Fetches 50 nics and prints the output in JSON format
`

	// RegisterMachineLongDesc long description for RegisterMachineCmd
	RegisterMachineLongDesc string = `Register a machine(ChromeBook, Bare metal server, Macbook.) by name.

Examples:
shivas register machine -f machine.json
Registers a machine by reading a JSON file input.

shivas register machine -i
Registers a machine by reading input through interactive mode.`

	// ReregisterMachineLongDesc long description for ReregisterMachineCmd
	ReregisterMachineLongDesc string = `Reregister/Update a machine(ChromeBook, Bare metal server, Macbook.) by name.

Examples:
shivas reregister machine -f machine.json
Reregister/Update a machine by reading a JSON file input.

shivas reregister machine -i
Reregister/Update a machine by reading input through interactive mode.`

	// ListMachineLongDesc long description for ListMachineCmd
	ListMachineLongDesc string = `List all Machines

Examples:
shivas ls machine
Fetches all the machines in table format

shivas ls machine -deployed
Fetches all the deployed machines in table format

shivas ls machine -n 50 -deployed -json
Fetches 50 deployed machines and prints the output in JSON format

shivas ls machine -n 5 -json
Fetches 5 machines and prints the output in JSON format
`

	// MachineFileText description for machine file input
	MachineFileText string = `Path to a file containing machine specification in JSON format.
This file must contain one machine JSON message

Example Browser machine:
{
	"name": "machine-BROWSERLAB-example",
	"location": {
		"lab": "LAB_DATACENTER_MTV97",
		"rack": "RackName"
	},
	"chromeBrowserMachine": {
		"displayName": "ax105-34-230",
		"chromePlatform": "Dell R230",
		"nics": ["ax105-34-230-eth0"],
		"kvmInterface": {
			"kvm": "kvm.mtv97",
			"port": 34
		},
		"rpmInterface": {
			"rpm": "rpm.mtv97",
			"port": 65
		},
		"drac": "ax105-34-230-drac",
		"deploymentTicket": "846026"
	},
	"realm": "Browserlab"
}

Example OS machine:
{
	"name": "machine-OSLAB-example",
	"location": {
		"lab": "LAB_CHROME_ATLANTA",
		"aisle": "1",
		"row": "2",
		"rack": "Rack-42",
		"rackNumber": "42",
		"shelf": "3",
		"position": "5"
	},
	"chromeosMachine": {},
	"realm": "OSlab"
}

The protobuf definition of machine is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine.proto#19`

	// DeployMachineLongDesc long description for DeployMachineCmd
	DeployMachineLongDesc string = `Deploy a machine as a DUT, Labstation, DevServer, Caching Server or a VM Server.

Examples:
shivas deploy-machine -f machinelse.json
Deploys a machine by reading a JSON file input.

shivas deploy-machine -i
Deploys a machine by reading input through interactive mode.`

	// RedeployMachineLongDesc long description for RedeployMachineCmd
	RedeployMachineLongDesc string = `Redeploy a machine as a DUT, Labstation, DevServer, Caching Server or a VM Server

Examples:
shivas redeploy-machine -f machinelse.json
Redeploys a machine by reading a JSON file input.

shivas redeploy-machine -i
Redeploys a machine by reading input through interactive mode.`

	// MachinelseFileText description for machinelse file input
	MachinelseFileText string = `Path to a file containing machine deployment specification in JSON format.
This file must contain one machine deployment JSON message

Example Browser machine deployment:
{
	"name": "A-Browser-MachineLSE-1",
	"machineLsePrototype": "browser-lab:vm",
	"hostname": "A-Browser-MachineLSE-1",
	"chromeBrowserMachineLse": {
		"vms": [
			{
				"name": "Windows8.0",
				"osVersion": {
					"value": "8.0",
					"description": "Windows Server"
				},
				"macAddress": "2.44.65.23",
				"hostname": "Windows8-lab1"
			},
			{
				"name": "Linux3.4",
				"osVersion": {
					"value": "3.4",
					"description": "Ubuntu Server"
				},
				"macAddress": "32.45.12.32",
				"hostname": "Ubuntu-lab2"
			}
		],
		"vmCapacity": 3
	},
	"machines": [
		"machine-DellServer-123"
	]
}

Example OS machine deployment for a DUT:
{
	"name": "chromeos3-row2-rack3-host5",
	"machineLsePrototype": "acs-lab:wifi",
	"hostname": "chromeos3-row2-rack3-host5",
	"chromeosMachineLse": {
		"deviceLse": {
			"dut": {
				"hostname": "chromeos3-row2-rack3-host5",
				"peripherals": {
					"servo": {
						"servoHostname": "chromeos3-row6-rack6-labstation6",
						"servoPort": 12,
						"servoSerial": "1234",
						"servoType": "V3"
					},
					"chameleon": {
						"chameleonPeripherals": [
							"CHAMELEON_TYPE_HDMI",
							"CHAMELEON_TYPE_BT_BLE_HID"
						],
						"audioBoard": true
					},
					"rpm": {
						"powerunitName": "rpm-1",
						"powerunitOutlet": "23"
					},
					"connectedCamera": [{
							"cameraType": "CAMERA_HUDDLY"
						},
						{
							"cameraType": "CAMERA_PTZPRO2"
						},
						{
							"cameraType": "CAMERA_HUDDLY"
						}
					],
					"audio": {
						"audioBox": true,
						"atrus": true
					},
					"wifi": {
						"wificell": true,
						"antennaConn": "CONN_OTA",
						"router": "ROUTER_802_11AX"
					},
					"touch": {
						"mimo": true
					},
					"carrier": "Att",
					"camerabox": true,
					"chaos": true,
					"cable": [{
							"type": "CABLE_USBAUDIO"
						},
						{
							"type": "CABLE_USBPRINTING"
						}
					],
					"cameraboxInfo": {
						"facing": "FACING_FRONT"
					}
				},
				"pools": [
					"ATL-LAB_POOL",
					"ACS_QUOTA"
				]
			},
			"rpmInterface": {
				"rpm": "rpm-asset-tag-123",
				"port": 23
			},
			"networkDeviceInterface": {
				"switch": "switch-1",
				"port": 23
			}
		}
	},
	"machines": [
		"ChromeBook-samus"
	]
}

Example OS machine deployment for a Labstation:
{
	"name": "chromeos3-row6-rack6-labstation6",
	"hostname": "chromeos3-row6-rack6-labstation6",
	"chromeosMachineLse": {
		"deviceLse": {
			"labstation": {
				"hostname": "chromeos3-row6-rack6-labstation6",
				"servos": [],
				"rpm": {
					"powerunitName": "rpm-1",
					"powerunitOutlet": "23"
				},
				"pools": [
					"ACS_POOL",
					"ACS_QUOTA"
				]
			},
			"rpmInterface": {
				"rpm": "rpm-asset-tag-123",
				"port": 23
			},
			"networkDeviceInterface": {
				"switch": "switch-1",
				"port": 23
			}
		}
	},
	"machines": [
		"machine-Labstation-samus"
	]
}

Example OS machine deployment for a Caching server/Dev server/VM server:
{
	"name": "A-ChromeOS-Server",
	"machineLsePrototype": "acs-lab:qwer",
	"hostname": "DevServer-1",
	"chromeosMachineLse": {
		"serverLse": {
			"supportedRestrictedVlan": "vlan-1",
			"service_port": 23
		}
	},
	"machines": [
		"machine-DellLinux-Server"
	]
}

The protobuf definition of a deployed machine is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine_lse.proto#24`

	// AddMachineLSEPrototypeLongDesc long description for AddMachineLSEPrototypeCmd
	AddMachineLSEPrototypeLongDesc string = `Add prototype for machine deployment.

Examples:	
shivas add-machine-prototype -f machineprototype.json
Adds a machine prototype by reading a JSON file input.

shivas add-machine-prototype -i
Adds a machine prototype by reading input through interactive mode.`

	// UpdateMachineLSEPrototypeLongDesc long description for UpdateMachineLSEPrototypeCmd
	UpdateMachineLSEPrototypeLongDesc string = `Update prototype for machine deployment.

Examples:
shivas update-machine-prototype -f machineprototype.json
Updates a machine prototype by reading a JSON file input.

shivas update-machine-prototype -i
Updates a machine prototype by reading input through interactive mode.`

	// ListMachineLSEPrototypeLongDesc long description for ListMachineLSEPrototypeCmd
	ListMachineLSEPrototypeLongDesc string = `List all machine prototypes

Examples:
shivas ls machine-prototype
Fetches all the machine prototypes in table format

shivas ls machine-prototype -n 50
Fetches 50 machine prototypes and prints the output in table format

shivas ls machine-prototype -lab acs -json
Fetches only ACS lab machine prototypes and prints the output in json format

shivas ls machine-prototype -n 5 -lab atl -json
Fetches 5 machine prototypes for ATL lab and prints the output in JSON format
`

	// MachineLSEPrototypeFileText description for MachineLSEPrototype file input
	MachineLSEPrototypeFileText string = `Path to a file containing prototype for machine deployment specification in JSON format.
This file must contain one machine prototype JSON message

Example prototype for machine deployment:
{
	"name": "browser-lab:vm",
	"peripheralRequirements": [{
		"peripheralType": "PERIPHERAL_TYPE_SWITCH",
		"min": 5,
		"max": 7
	}],
	"occupiedCapacityRu": 32,
	"virtualRequirements": [{
		"virtualType": "VIRTUAL_TYPE_VM",
		"min": 3,
		"max": 4
	}]
}

The protobuf definition of prototype for machine deployment is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/lse_prototype.proto#29`

	// AddRackLSEPrototypeLongDesc long description for AddRackLSEPrototypeCmd
	AddRackLSEPrototypeLongDesc string = `Add prototype for rack deployment.

Examples:	
shivas add-rack-prototype -f rackprototype.json
Adds a rack prototype by reading a JSON file input.

shivas add-rack-prototype -i
Adds a rack prototype by reading input through interactive mode.`

	// UpdateRackLSEPrototypeLongDesc long description for UpdateRackLSEPrototypeCmd
	UpdateRackLSEPrototypeLongDesc string = `Update prototype for rack deployment.

Examples:
shivas update-rack-prototype -f rackprototype.json
Updates a rack prototype by reading a JSON file input.

shivas update-rack-prototype -i
Updates a rack prototype by reading input through interactive mode.`

	// ListRackLSEPrototypeLongDesc long description for ListRackLSEPrototypeCmd
	ListRackLSEPrototypeLongDesc string = `List all rack prototypes

Examples:
shivas ls rack-prototype
Fetches all the rack prototypes in table format

shivas ls rack-prototype -n 50
Fetches 50 rack prototypes and prints the output in table format

shivas ls rack-prototype -lab acs -json
Fetches only ACS lab rack prototypes and prints the output in json format

shivas ls rack-prototype -n 5 -lab atl -json
Fetches 5 rack prototypes for ATL lab and prints the output in JSON format
`

	// RackLSEPrototypeFileText description for RackLSEPrototype file input
	RackLSEPrototypeFileText string = `Path to a file containing prototype for rack deployment specification in JSON format.
This file must contain one rack prototype JSON message

Example prototype for rack deployment:
{
	"name": "browser-lab:vm",
	"peripheralRequirements": [{
		"peripheralType": "PERIPHERAL_TYPE_SWITCH",
		"min": 5,
		"max": 7
	}]
}

The protobuf definition of prototype for rack deployment is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/lse_prototype.proto`

	// AddChromePlatformLongDesc long description for AddChromePlatformCmd
	AddChromePlatformLongDesc string = `Add chrome platform configuration for browser machine.

Examples:	
shivas add-chrome-platform -f chromeplatform.json
Adds a chrome platform by reading a JSON file input.

shivas add-chrome-platform -i
Adds a chrome platform by reading input through interactive mode.`

	// UpdateChromePlatformLongDesc long description for UpdateChromePlatformCmd
	UpdateChromePlatformLongDesc string = `Update chrome platform configuration for browser machine.

Examples:
shivas update-chrome-platform -f chromeplatform.json
Updates a chrome platform by reading a JSON file input.

shivas update-chrome-platform -i
Updates a chrome platform by reading input through interactive mode.`

	// ListChromePlatformLongDesc long description for ListChromePlatformCmd
	ListChromePlatformLongDesc string = `List all chrome platforms

Examples:
shivas ls chrome-platform
Fetches all the chrome platforms in table format

shivas ls chrome-platform -n 50
Fetches 50 chrome platforms and prints the output in table format

shivas ls chrome-platform -json
Fetches all chrome platforms and prints the output in json format

shivas ls chrome-platform -n 5 -lab atl -json
Fetches 5 chrome platforms for ATL lab and prints the output in JSON format
`

	// ChromePlatformFileText description for ChromePlatform file input
	ChromePlatformFileText string = `Path to a file containing chrome platform configuration for browser machine specification in JSON format.
This file must contain one chrome platform JSON message

Example chrome platform configuration:
{
	"name": "Dell_Signia",
	"manufacturer": "Dell",
	"description": "Dell x86 platform"
}

The protobuf definition of chrome platform configuration for browser machine is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/chrome_platform.proto`
)
