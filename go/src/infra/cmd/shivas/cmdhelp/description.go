// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmdhelp

var (
	// ListPageSizeDesc description for List PageSize
	ListPageSizeDesc string = `number of items to get. The service may return fewer than this value.`

	//AddSwitchLongDesc long description for AddSwitchCmd
	AddSwitchLongDesc string = `Create a switch to UFS.

Examples:
shivas add-switch -f switch.json -r {Rack name}
Adds a switch by reading a JSON file input.

shivas add-switch -rack {Rack name} -name {switch name} -capacity {50} -description {description}
Adds a switch by specifying several attributes directly.

shivas add-switch -i
Adds a switch by reading input through interactive mode.`

	// UpdateSwitchLongDesc long description for UpdateSwitchCmd
	UpdateSwitchLongDesc string = `Update a switch by name.

Examples:
shivas update-switch -f switch.json
Update a switch by reading a JSON file input.

shivas update-switch -f drac.json -r {Rack name}
Update a switch by reading a JSON file input and associate the switch with a different rack.

shivas update-switch -i
Update a switch by reading input through interactive mode.`

	// ListSwitchLongDesc long description for ListSwitchCmd
	ListSwitchLongDesc string = `List all switches

Examples:
shivas list switch
Fetches all switches and prints the output in table format

shivas list switch -n 50
Fetches 50 switches and prints the output in table format

shivas list switch -json
Fetches all switches and prints the output in JSON format

shivas list switch -n 50 -json
Fetches 50 switches and prints the output in JSON format
`

	// SwitchFileText description for switch file input
	SwitchFileText string = `[JSON Mode] Path to a file containing switch specification in JSON format.
This file must contain one switch JSON message

Example switch:
{
    "name": "switch-test-example",
    "capacityPort": 456,
    "description": "I am a switch"
}

The protobuf definition of switch is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// VMFileText description for VM file input
	VMFileText string = `Path to a file containing VM specification in JSON format.
This file must contain one VM JSON message

Example VM:
{
	"name": "Windows8.0",
	"osVersion": {
		"value": "8.0",
		"description": "Windows Server"
	},
	"macAddress": "2.44.65.23",
	"hostname": "Windows8.0"
}

The protobuf definition of VM is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine_lse.proto`

	// ListVMLongDesc long description for ListVMCmd
	ListVMLongDesc string = `List all vms for a host

Examples:
shivas list vm -h {Hostname}
Fetches all vms for the host and prints the output in table format

shivas list vm -h {Hostname} -json
Fetches all vms for the host and prints the output in JSON format
`

	// ListKVMLongDesc long description for ListKVMCmd
	ListKVMLongDesc string = `List all kvms

Examples:
shivas list kvm
Fetches all kvms and prints the output in table format

shivas list kvm -n 50
Fetches 50 kvms and prints the output in table format

shivas list kvm -json
Fetches all kvms and prints the output in JSON format

shivas list kvm -n 50 -json
Fetches 50 kvms and prints the output in JSON format
`

	// ListRPMLongDesc long description for ListRPMCmd
	ListRPMLongDesc string = `List all rpms

Examples:
shivas list rpm
Fetches all rpms and prints the output in table format

shivas list rpm -n 50
Fetches 50 rpms and prints the output in table format

shivas list rpm -json
Fetches all rpms and prints the output in JSON format

shivas list rpm -n 50 -json
Fetches 50 rpms and prints the output in JSON format
`

	// ListDracLongDesc long description for ListDracCmd
	ListDracLongDesc string = `List all dracs

Examples:
shivas list drac
Fetches all dracs and prints the output in table format

shivas list drac -n 50
Fetches 50 dracs and prints the output in table format

shivas list drac -json
Fetches all dracs and prints the output in JSON format

shivas list drac -n 50 -json
Fetches 50 dracs and prints the output in JSON format
`

	// ListNicLongDesc long description for ListNicCmd
	ListNicLongDesc string = `List all nics

Examples:
shivas list nic
Fetches all nics and prints the output in table format

shivas list nic -n 50
Fetches 50 nics and prints the output in table format

shivas list nic -json
Fetches all nics and prints the output in JSON format

shivas list nic -n 50 -json
Fetches 50 nics and prints the output in JSON format
`

	// AddMachineLongDesc long description for AddMachineCmd
	AddMachineLongDesc string = `Create a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.) by name.
You can also provide the optional nics and drac information to create the nics and drac associated with this machine.
You can also create the nic and drac separately after creating the machine using add-nic/add-drac commands.

Examples:
shivas add-machine -f machinerequest.json
Creates a machine by reading a JSON file input.

shivas add-machine -i
Creates a machine by reading input through interactive mode.`

	// UpdateMachineLongDesc long description for UpdateMachineCmd
	UpdateMachineLongDesc string = `Update a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.) by name.

Examples:
shivas update-machine -f machine.json
Update a machine by reading a JSON file input.

shivas update-machine -i
Update a machine by reading input through interactive mode.`

	// ListMachineLongDesc long description for ListMachineCmd
	ListMachineLongDesc string = `List all Machines

Examples:
shivas list machine
Fetches all the machines in table format

shivas list machine -deployed
Fetches all the deployed machines in table format

shivas list machine -n 50 -deployed -json
Fetches 50 deployed machines and prints the output in JSON format

shivas list machine -n 5 -json
Fetches 5 machines and prints the output in JSON format
`

	// MachineRegistrationFileText description for machine registration file input
	MachineRegistrationFileText string = `Path to a file containing machine creation request specification in JSON format.
This file must contain required machine field and optional nics/drac field.

Example Browser machine creation request:
{
	"machine": {
		"name": "machine-BROWSERLAB-example",
		"location": {
			"lab": "LAB_DATACENTER_MTV97",
			"rack": "RackName"
		},
		"chromeBrowserMachine": {
			"displayName": "ax105-34-230",
			"chromePlatform": "Dell R230",
			"kvmInterface": {
				"kvm": "kvm.mtv97",
				"port": 34
			},
			"rpmInterface": {
				"rpm": "rpm.mtv97",
				"port": 65
			},
			"deploymentTicket": "846026"
		},
		"realm": "Browserlab"
	},
	"nics": [{
			"name": "nic-eth0",
			"macAddress": "00:0d:5d:10:64:8d",
			"switchInterface": {
				"switch": "switch-12",
				"port": 15
			}
		},
		{
			"name": "nic-eth1",
			"macAddress": "22:0d:4f:10:65:9f",
			"switchInterface": {
				"switch": "switch-12",
				"port": 16
			}
		}
	],
	"drac": {
		"name": "drac-23",
		"displayName": "Cisco Drac",
		"macAddress": "10:03:3d:70:64:2d",
		"switchInterface": {
			"switch": "switch-12",
			"port": 17
		},
		"password": "WelcomeDrac***"
	}
}

Example OS machine creation request:
{
	"machine": {
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
}

The protobuf definition can be found here: 
Machine creation request: MachineRegistrationRequest
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/rpc/fleet.proto

Machine:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine.proto

Nic and Drac:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

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
		"kvmInterface": {
			"kvm": "kvm.mtv97",
			"port": 34
		},
		"rpmInterface": {
			"rpm": "rpm.mtv97",
			"port": 65
		},
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine.proto`

	// AddHostLongDesc long description for AddHostCmd
	AddHostLongDesc string = `Add a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine

Examples:
shivas add-host -f host.json -m {Machine name}
Adds a host by reading a JSON file input.
-m option is a required parameter to associate the host to the given machine.

shivas add-host -i
Adds a host by reading input through interactive mode.`

	// UpdateHostLongDesc long description for UpdateHostCmd
	UpdateHostLongDesc string = `Update a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine

Examples:
shivas update-host -f host.json
Updates a host by reading a JSON file input.

shivas update-host -f host.json -m {Machine name}
Update a host by reading a JSON file input and associate the host with a different machine.

shivas update-host -i
Updates a host by reading input through interactive mode.`

	// MachineLSEFileText description for machinelse/host file input
	MachineLSEFileText string = `Path to a file containing host specification in JSON format.
This file must contain one machine deployment JSON message

Example host for a browser machine:
{
	"name": "A-Browser-Host-1",
	"machineLsePrototype": "browser-lab:vm",
	"hostname": "A-Browser-Host-1",
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
		"vmCapacity": 3,
		"osVersion": {
			"value": "3.4",
			"description": "Ubuntu Server"
		},
	}
}

Example host(DUT) for an OS machine:
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
	}
}

Example host(Labstation) for an OS machine:
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
	}
}

Example host(Caching server/Dev server/VM server) for an OS machine:
{
	"name": "A-ChromeOS-Server",
	"machineLsePrototype": "acs-lab:qwer",
	"hostname": "DevServer-1",
	"chromeosMachineLse": {
		"serverLse": {
			"supportedRestrictedVlan": "vlan-1",
			"service_port": 23
		}
	}
}

The protobuf definition of a deployed machine is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine_lse.proto`

	// ListHostLongDesc long description for ListHostCmd
	ListHostLongDesc string = `List all hosts

Examples:
shivas list host
Prints all the hosts in JSON format

shivas list machine -n 50
Prints 50 hosts in JSON format
`

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
shivas list machine-prototype
Fetches all the machine prototypes in table format

shivas list machine-prototype -n 50
Fetches 50 machine prototypes and prints the output in table format

shivas list machine-prototype -lab acs -json
Fetches only ACS lab machine prototypes and prints the output in json format

shivas list machine-prototype -n 5 -lab atl -json
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
shivas list rack-prototype
Fetches all the rack prototypes in table format

shivas list rack-prototype -n 50
Fetches 50 rack prototypes and prints the output in table format

shivas list rack-prototype -lab acs -json
Fetches only ACS lab rack prototypes and prints the output in json format

shivas list rack-prototype -n 5 -lab atl -json
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
shivas list chrome-platform
Fetches all the chrome platforms in table format

shivas list chrome-platform -n 50
Fetches 50 chrome platforms and prints the output in table format

shivas list chrome-platform -json
Fetches all chrome platforms and prints the output in json format

shivas list chrome-platform -n 5 -json
Fetches 5 chrome platforms and prints the output in JSON format
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

	// AddNicLongDesc long description for AddNicCmd
	AddNicLongDesc string = `Add a nic by name.

Examples:
shivas add-nic -f nic.json -m {Machine name}
Add a nic by reading a JSON file input.
-m option is a required parameter to associate the nic to the given machine.

shivas add-nic -i
Add a nic by reading input through interactive mode.`

	// UpdateNicLongDesc long description for UpdateNicCmd
	UpdateNicLongDesc string = `Update a nic by name.

Examples:
shivas update-nic -f nic.json
Update a nic by reading a JSON file input.

shivas update-nic -f nic.json -m {Machine name}
Update a nic by reading a JSON file input and associate the nic with a different machine.

shivas update-nic -i
Update a nic by reading input through interactive mode.`

	// NicFileText description for nic file input
	NicFileText string = `Path to a file containing nic specification in JSON format.
This file must contain one nic JSON message

Example nic:
{
	"name": "nic-23",
	"macAddress": "00:0d:5d:10:64:8d",
	"switchInterface": {
		"switch": "switch-12",
		"port": 15
	}
}

The protobuf definition of nic is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// AddDracLongDesc long description for AddDracCmd
	AddDracLongDesc string = `Add a drac by name.

Examples:
shivas add-drac -f drac.json -m {Machine name}
Add a drac by reading a JSON file input.
-m option is a required parameter to associate the drac to the given machine.

shivas add-drac -i
Add a drac by reading input through interactive mode.`

	// UpdateDracLongDesc long description for UpdateDracCmd
	UpdateDracLongDesc string = `Update a drac by name.

Examples:
shivas update-drac -f drac.json
Update a drac by reading a JSON file input.

shivas update-drac -f drac.json -m {Machine name}
Update a drac by reading a JSON file input and associate the drac with a different machine.

shivas update-drac -i
Update a drac by reading input through interactive mode.`

	// DracFileText description for drac file input
	DracFileText string = `Path to a file containing drac specification in JSON format.
This file must contain one drac JSON message

Example drac:
{
	"name": "drac-23",
	"displayName": "Cisco Drac",
	"macAddress": "00:0d:5d:10:64:8d",
	"switchInterface": {
		"switch": "switch-12",
		"port": 15
	},
	"password": "WelcomeDrac***"
}

The protobuf definition of drac is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// AddKVMLongDesc long description for AddKVMCmd
	AddKVMLongDesc string = `Add a kvm to UFS.

Examples:
shivas add-kvm -new-json-file kvm.json -rack {Rack name}
Add a kvm by reading a JSON file input.

shivas add-kvm -rack {Rack name} -name {kvm name} -mac-address {mac} -platform {platform}
Add a kvm by specifying several attributes directly.

shivas add-kvm -i
Add a kvm by reading input through interactive mode.`

	// UpdateKVMLongDesc long description for UpdateKVMCmd
	UpdateKVMLongDesc string = `Update a kvm by name.

Examples:
shivas update-kvm -f kvm.json
Update a kvm by reading a JSON file input.

shivas update-kvm -f kvm.json -r {Rack name}
Update a kvm by reading a JSON file input and associate the kvm with a different rack.

shivas update-kvm -i
Update a kvm by reading input through interactive mode.`

	// KVMFileText description for kvm file input
	KVMFileText string = `[JSON Mode] Path to a file containing kvm specification in JSON format.
This file must contain one kvm JSON message

Example kvm:
{
	"name": "kvm-23",
	"macAddress": "00:0d:5d:10:64:8d",
	"chromePlatform": "Gigabyte_R181-T92",
	"capacityPort": 48
}

The protobuf definition of kvm is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// AddRackLongDesc long description for AddRackCmd
	AddRackLongDesc string = `Create a rack to UFS.
You can create a rack with name and lab to UFS, and later add kvm/switch/rpm separately by using add-switch/add-kvm/add-rpm commands.

You can also provide the optional switches, kvms and rpms information to create the switches, kvms and rpms associated with this rack by specifying a json file as input.

Examples:
shivas add-rack -json-file rackrequest.json -lab lab01
Creates a rack by reading a JSON file input.

shivas add-rack -name rack-123 -lab lab01 -capacity 10
Creates a rack by parameters without adding kvm/switch/rpm.`

	// UpdateRackLongDesc long description for UpdateRackCmd
	UpdateRackLongDesc string = `Update a rack by name.

Examples:
shivas update-rack -f rack.json
Update a rack by reading a JSON file input.

shivas update-rack -i
Update a rack by reading input through interactive mode.`

	// ListRackLongDesc long description for ListRackCmd
	ListRackLongDesc string = `List all Racks

Examples:
shivas ls rack
Fetches all the racks and prints in table format

shivas ls rack -n 5 -json
Fetches 5 racks and prints the output in JSON format
`

	// RackRegistrationFileText description for rack registration file input
	RackRegistrationFileText string = `[JSON Mode] Path to a file containing rack creation request specification in JSON format.
This file must contain required rack field and optional switches/kvms/rpms fields.

Example rack creation request:
{
	"rack": {
		"name": "rack-BROWSERLAB-example",
		"capacity_ru": 5
	},
	"switches": [{
			"name": "switch-23",
			"capacityPort": 456
		},
		{
			"name": "switch-25",
			"capacityPort": 456
		}
	],
	"kvms": [{
			"name": "kvm-23",
			"macAddress": "00:0d:5d:10:64:8d",
			"chromePlatform": "Gigabyte_R181-T92",
			"capacityPort": 48
		},
		{
			"name": "kvm-25",
			"macAddress": "00:0d:5d:20:64:8d",
			"chromePlatform": "Gigabyte_R181-T92",
			"capacityPort": 44
		}
	],
	"rpms": [{
		"name": "rpm-23",
		"macAddress": "00:0d:5d:10:64:8d",
		"capacityPort": 48
	}, {
		"name": "rpm-25",
		"macAddress": "00:0d:5d:10:68:8d",
		"capacityPort": 45
	}]
}

The protobuf definition can be found here:
Rack creation request: RackRegistrationRequest
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/rpc/fleet.proto

Rack:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/rack.proto

Switch, KVM and RPM:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// RackFileText description for rack file input
	RackFileText string = `Path to a file containing rack specification in JSON format.
This file must contain one rack JSON message

Example Browser rack:
{
	"name": "rack-BROWSERLAB-example",
	"location": {
		"lab": "LAB_DATACENTER_MTV97"
	},
	"capacity_ru": 5,
	"chromeBrowserRack": {},
	"realm": "Browserlab"
}

{
	"name": "rack-OSLAB-example",
	"location": {
		"lab": "LAB_DATACENTER_SANTIAM"
	},
	"capacity_ru": 5,
	"chromeosRack": {},
	"realm": "Browserlab"
}

The protobuf definition of rack is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/rack.proto`
)
