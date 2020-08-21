// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmdhelp

import (
	"fmt"
	"strings"

	"infra/cmd/shivas/utils"
	ufsUtil "infra/unifiedfleet/app/util"
)

var (
	// ListPageSizeDesc description for List PageSize
	ListPageSizeDesc string = `number of items to get. The service may return fewer than this value.`

	//AddSwitchLongDesc long description for AddSwitchCmd
	AddSwitchLongDesc string = `Create a switch to UFS.

Examples:
shivas add-switch -f switch.json
Adds a switch by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.

shivas add-switch -rack {Rack name} -name {switch name} -capacity {50} -description {description}
Adds a switch by specifying several attributes directly.

shivas add-switch -i
Adds a switch by reading input through interactive mode.`

	// UpdateSwitchLongDesc long description for UpdateSwitchCmd
	UpdateSwitchLongDesc string = `Update a switch by name.

Examples:
shivas update-switch -f switch.json
Update a switch by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.

shivas update-switch -i
Update a switch by reading input through interactive mode.

shivas update-switch -rack {Rack name} -name {switch name} -capacity {50} -description {description}
Partial updates a switch by parameters. Only specified parameters will be udpated in the switch.`

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
    "name": "eq079.atl97",
    "capacityPort": 48,
    "description": "Arista Networks DCS-7050T-52",
    "tags": ["dell", "8g"],
    "rack": "cr-22"
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
    "hostname": "Windows8.0",
    "tags": ["dell", "8g"],
    "machineLseId" : "adb-1"
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
	AddMachineLongDesc string = `Create a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.) to UFS.

You can create a machine with required parameters to UFS, and later add nics/drac separately by using add-nic/add-drac commands.

You can also provide the optional nics and drac information to create the nics and drac associated with this machine by specifying a json file as input.

Examples:
shivas add-machine -f machinerequest.json
Creates a machine by reading a JSON file input.

shivas add-machine -name machine1 -zone mtv97 -rack rack1 -ticket b/1234 -platform platform1 -kvm kvm1
Creates a machine by parameters without adding nic/drac.`

	// UpdateMachineLongDesc long description for UpdateMachineCmd
	UpdateMachineLongDesc string = `Update a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.) by name.

Examples:
shivas update-machine -f machine.json
Update a machine by reading a JSON file input.

shivas update-machine -i
Update a machine by reading input through interactive mode.

shivas update-machine -name machine1 -zone mtv97 -rack rack1
Partial updates a machine by parameters. Only specified parameters will be udpated in the machine.`

	// ListMachineLongDesc long description for ListMachineCmd
	ListMachineLongDesc string = `List all Machines

Examples:
shivas list machine
Fetches all the machines in table format

shivas list machine -n 5 -json
Fetches 5 machines and prints the output in JSON format
`

	// MachineRegistrationFileText description for machine registration file input
	MachineRegistrationFileText string = `[JSON Mode] Path to a file containing machine request specification in JSON format.
This file must contain required machine field and optional nics/drac field.

Example Browser machine creation request:
{
    "name": "cr85-XXX",
    "serialNumber": "FVSMVXX",
    "location": {
        "rack": "cr85XX",
        "zone": "ZONE_ATL97"
    },
    "tags": ["dell", "8g"],
    "chromeBrowserMachine": {
        "displayName": "cr85-XXX",
        "chromePlatform": "Dell_R720",
        "deploymentTicket": "846026XX",
        "description": "adding a machine cr85-XXX",
        "kvmInterface": {
            "kvm": "ax101-kvm1",
            "port": 34
        },
        "rpmInterface": {
            "rpm": "rpm-23",
            "port": 65
        },
        "nicObjects": [{
            "name": "cr85-XXX:eth0",
            "macAddress": "ec:f4:bb:c4:6e:qq",
            "switchInterface": {
                "switch": "eq082.atl97",
                "portName": "11"
            },
            "tags": ["dell", "8g"]
        }],
        "dracObject": {
            "name": "lin64-38-m0-dracXXX",
            "displayName": "cr85-XXX:drac",
            "macAddress": "5c:f9:dd:fd:6d:ff",
            "switchInterface": {
                "switch": "eq082.atl97",
                "portName": "41"
            },
            "password": "WelcomeDrac***",
            "tags": ["dell", "8g"]
        }
    }
}

Example OS machine creation request:
{
    "name": "machine-OSLAB-example",
    "location": {
        "zone": "ZONE_ATLANTA",
        "aisle": "1",
        "row": "2",
        "rack": "Rack-42",
        "rackNumber": "42",
        "shelf": "3",
        "position": "5"
    },
    "serialNumber": "XXX",
    "chromeosMachine": {}
}


The protobuf definition can be found here:
Machine:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine.proto

Drac:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto

Nic:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/network.proto`

	// MachineFileText description for machine file input
	MachineFileText string = `Path to a file containing machine specification in JSON format.
This file must contain one machine JSON message

Example Browser machine:
{
    "name": "cr85-XXX",
    "serialNumber": "FVSMVXX",
    "location": {
        "rack": "cr85XX",
        "zone": "ZONE_ATL97"
    },
    "tags": ["dell", "8g"],
    "chromeBrowserMachine": {
        "displayName": "cr85-XXX",
        "chromePlatform": "Dell_R720",
        "deploymentTicket": "846026XX",
        "description": "adding a machine cr85-XXX",
        "kvmInterface": {
            "kvm": "ax101-kvm1",
            "port": 34
        },
        "rpmInterface": {
            "rpm": "rpm-23",
            "port": 65
        }
    }
}

Example OS machine:
{
    "name": "machine-OSLAB-example",
    "location": {
        "zone": "ZONE_ATLANTA",
        "aisle": "1",
        "row": "2",
        "rack": "Rack-42",
        "rackNumber": "42",
        "shelf": "3",
        "position": "5"
    },
    "serialNumber" : "XXX",
    "chromeosMachine": {}
}

The protobuf definition of machine is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine.proto`

	// AddHostLongDesc long description for AddHostCmd
	AddHostLongDesc string = `Add a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine

Examples:
shivas add-host -f host.json
Adds a host by reading a JSON file input.
[WARNING]: machines is a required field in json, all other output only fields will be ignored.
Specify additional settings, e.g. vlan, nic, ip via command line parameters along with JSON input

shivas add-host -machine machine0 -name host0 -prototype browser-lab:no-vm  -osversion chrome-version-0 -vm-capacity 3
Adds a host by parameters without adding vms.

shivas add-host -i
Adds a host by reading input through interactive mode.`

	// UpdateHostLongDesc long description for UpdateHostCmd
	UpdateHostLongDesc string = `Update a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine

Examples:
shivas update-host -f host.json
Updates a host by reading a JSON file input.
[WARNING]: machines is a required field in json, all other output only fields will be ignored.
Specify additional settings, e.g. vlan, ip, nic, state via command line parameters along with JSON input

shivas update-host -name cr22 -os windows
Partial update a host by parameters. Only specified parameters will be updated in the host.

shivas update-host -name host0 -delete-vlan
Remove the ip for host

shivas update-host -name host0 -vlan browser-lab:11 -nic eth0
Assign ip to the host

shivas update-host -i
Updates a host by reading input through interactive mode.`

	// MachineLSEFileText description for machinelse/host file input
	MachineLSEFileText string = `[JSON mode] Path to a file containing host specification in JSON format.
This file must contain one machine deployment JSON message

Example host for a browser machine:
{
    "name": "esx-380XXX",
    "machineLsePrototype": "browser-lab:vm",
    "hostname": "esx-380XXX",
    "tags": ["dell", "8g"],
    "nic": "cr151-16-macproXXX:eth0",
    "machines": ["cr205-19-230"],
    "chromeBrowserMachineLse": {
        "vms": [{
            "name": "vm991-m4XXX",
            "osVersion": {
                "value": "macOS_10.13.6_(17G65)",
                "description": "Windows Server"
            },
            "macAddress": "ab:cd:ab:cd:ab:cd",
            "hostname": "vm991-m4XXX",
            "tags": ["dell", "8g"]
        }],
        "vmCapacity": 3,
        "osVersion": {
            "value": "ESXi_6.7.0XXX",
            "description": "Windows Server"
        }
    }
}

Example host(DUT) for an OS machine:
{
    "name": "chromeos3-row2-rack3-host5",
    "machineLsePrototype": "acs-lab:wifi",
    "hostname": "chromeos3-row2-rack3-host5",
    "machines": ["cr205-19-230"],
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
    "machines": ["cr205-19-230"],
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
    "machines": ["cr205-19-230"],
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

shivas list host -n 50
Prints 50 hosts in JSON format

Valid states[" + strings.Join(ufsUtil.ValidStateStr(), ", ") + "]"
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
shivas list machineprototype
Fetches all the machine prototypes in table format

shivas list machineprototype -n 50
Fetches 50 machine prototypes and prints the output in table format

shivas list machineprototype -filter 'tag=acs,camera' -json
Fetches only acs and camera tagged machine prototypes and prints the output in json format
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
    }],
    "tags": ["dell", "8g"]
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
shivas list rackprototype
Fetches all the rack prototypes in table format

shivas list rackprototype -n 50
Fetches 50 rack prototypes and prints the output in table format

shivas list rackprototype -filter 'tag=browser' -json
Fetches only browser tagged rack prototypes and prints the output in json format
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
    }],
    "tags": ["dell", "8g"]
}

The protobuf definition of prototype for rack deployment is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/lse_prototype.proto`

	// AddChromePlatformLongDesc long description for AddChromePlatformCmd
	AddChromePlatformLongDesc string = `Add platform configuration for browser machine.

Examples:
shivas add-platform -f platform.json
Adds a platform by reading a JSON file input.

shivas add-platform -i
Adds a platform by reading input through interactive mode.

shivas add-platform -name DELL_R320 -manufacturer Dell -tags 'dell,8g' -desc 'Dell platform'
Adds a platform by specifying several attributes directly`

	// AddVlanLongDesc long description for AddVlanCmd
	AddVlanLongDesc string = `Add vlans.

Examples:
shivas add-vlan -name browser-lab:100 -cidr-block A.B.C.D/24 -desc "atl97-vlan"
Adds a vlan by specifying several attributes directly`

	// UpdateChromePlatformLongDesc long description for UpdateChromePlatformCmd
	UpdateChromePlatformLongDesc string = `Update platform configuration for browser machine.

Examples:
shivas update-platform -f platform.json
Updates a platform by reading a JSON file input.

shivas update-platform -i
Updates a platform by reading input through interactive mode.

shivas update-platform -name DELL_R320 -manufacturer Dell -tags -'
Updates a platform partially, only specified field values will be updated, with other values remaining the same.
You can clear/empty a field value by providing a - for value as shown for -tags`

	// ListChromePlatformLongDesc long description for ListChromePlatformCmd
	ListChromePlatformLongDesc string = `List all platforms

Examples:
shivas list platform
Fetches all the platforms in table format

shivas list platform -n 50
Fetches 50 platforms and prints the output in table format

shivas list platform -json
Fetches all platforms and prints the output in json format

shivas list platform -n 5 -json
Fetches 5 platforms and prints the output in JSON format
`

	// UpdateVlanLongDesc long description for UpdateVlanCmd
	UpdateVlanLongDesc string = `Update vlan configuration.

only description and state are allowed to update. cidr_block is not allowed to be updated to avoid any potential huge amount dhcp/ip changes of hosts.

Examples:

shivas update-vlan -name vlan_name -desc test -state serving'
Updates a vlan partially, only specified field values will be updated, with other values remaining the same.
You can clear/empty a field value by providing a "-"`

	// ListVlanLongDesc long description for ListVlanCmd
	ListVlanLongDesc string = `List all vlans by some filters

Examples:
shivas list vlan
Fetches all the vlans in table format

shivas list vlan -n 50
Fetches 50 vlans and prints the output in table format

shivas list vlan -filter "state=serving"
Fetches all vlans and prints the output in json format
`

	// ChromePlatformFileText description for ChromePlatform file input
	ChromePlatformFileText string = `Path to a file containing platform configuration for browser machine specification in JSON format.
This file must contain one platform JSON message

Example platform configuration:
{
    "name": "Dell_Signia",
    "manufacturer": "Dell",
    "description": "Dell x86 platform",
    "tags": ["dell", "8g"]
}

The protobuf definition of platform configuration for browser machine is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/chrome_platform.proto`

	// AddNicLongDesc long description for AddNicCmd
	AddNicLongDesc string = `Add a nic to UFS.

Examples:
shivas add-nic -f nic.json
Add a nic by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas add-nic -name machine0:eth0 -switch switch0 -mac-address 123456 -machine machine0 -switch-port 1
Add a nic by specifying several attributes directly.

shivas add-nic -i
Add a nic by reading input through interactive mode.`

	// UpdateNicLongDesc long description for UpdateNicCmd
	UpdateNicLongDesc string = `Update a nic by name.

Examples:
shivas update-nic -f nic.json
Update a nic by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas update-nic -i
Update a nic by reading input through interactive mode.

shivas update-nic -name machine0:eth0 -switch switch0 -mac-address 12345
Partial update a nic by parameters. Only specified parameters will be updated in the nic.`

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
    },
    "tags": ["dell", "8g"],
    "machine": "mac-1"
}

The protobuf definition of nic is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/network.proto`

	// AddDracLongDesc long description for AddDracCmd
	AddDracLongDesc string = `Add a drac to UFS.

Examples:
shivas add-drac -f drac.json
Add a drac by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas add-drac -name machine0:drac -switch switch0 -mac-address 123456 -machine machine0 -switch-port 1
Add a drac by specifying several attributes directly.

shivas add-drac -i
Add a drac by reading input through interactive mode.
`

	// UpdateDracLongDesc long description for UpdateDracCmd
	UpdateDracLongDesc string = `Update a drac by name.

Examples:
shivas update-drac -f drac.json
Update a drac by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas update-drac -name machine0:drac -switch switch0 -mac-address 123456
Partial update a drac by parameters. Only specified parameters will be updated in the drac.

shivas update-drac -name drac0 -delete-vlan
Remove the ip for drac0

shivas update-drac -name drac0 -vlan browser-lab:11
Assign ip to the drac

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
    "password": "WelcomeDrac***",
    "tags": ["dell", "8g"],
    "machine": "mac-1"
}

The protobuf definition of drac is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// AddKVMLongDesc long description for AddKVMCmd
	AddKVMLongDesc string = `Add a kvm to UFS.

Examples:
shivas add-kvm -f kvm.json
Add a kvm by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.

shivas add-kvm -rack {Rack name} -name {kvm name} -mac-address {mac} -platform {platform}
Add a kvm by specifying several attributes directly.

shivas add-kvm -i
Add a kvm by reading input through interactive mode.`

	// UpdateKVMLongDesc long description for UpdateKVMCmd
	UpdateKVMLongDesc string = `Update a kvm by name.

Examples:
shivas update-kvm -f kvm.json
Update a kvm by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.
Specify additional settings, e.g. vlan, ip via command line parameters along with JSON input

shivas update-kvm -i
Update a kvm by reading input through interactive mode.

shivas update-kvm -rack {Rack name} -name {kvm name} -mac-address {mac} -platform {platform}
Partial updates a kvm by parameters. Only specified parameters will be udpated in the kvm.`

	// KVMFileText description for kvm file input
	KVMFileText string = `[JSON Mode] Path to a file containing kvm specification in JSON format.
This file must contain one kvm JSON message

Example kvm:
{
    "name": "cx101-kvm1XXX",
    "macAddress": "00:0d:5d:0f:54:ed",
    "chromePlatform": "Raritan_DKX3",
    "capacityPort": 48,
    "tags": ["dell", "8g"],
    "rack": "cr-22"
}

The protobuf definition of kvm is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// AddRackLongDesc long description for AddRackCmd
	AddRackLongDesc string = `Create a rack to UFS.
You can create a rack with name and zone to UFS, and later add kvm/switch/rpm separately by using add-switch/add-kvm/add-rpm commands.

You can also provide the optional switches, kvms and rpms information to create the switches, kvms and rpms associated with this rack by specifying a json file as input.

Examples:
shivas add-rack -f rackrequest.json
Creates a rack by reading a JSON file input.

shivas add-rack -name rack-123 -zone lab01 -capacity 10
Creates a rack by parameters without adding kvm/switch/rpm.`

	// UpdateRackLongDesc long description for UpdateRackCmd
	UpdateRackLongDesc string = `Update a rack by name.

Examples:
shivas update-rack -f rack.json
Update a rack by reading a JSON file input.

shivas update-rack -i
Update a rack by reading input through interactive mode.

shivas update-rack -name rack-123 -zone lab01 -capacity 10
Partial updates a rack by parameters. Only specified parameters will be udpated in the rack.`

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

Example browser rack creation request:
{
    "name": "cr82",
    "location": {
        "rack": "cr82",
        "zone": "ZONE_ATL97"
    },
    "capacity_ru": 5,
    "tags": ["dell", "8g"],
    "chromeBrowserRack": {
        "kvmObjects": [{
            "name": "cr82-kvm1",
            "macAddress": "00:0d:5d:11:63:2a",
            "chromePlatform": "Raritan_DKX3",
            "capacityPort": 48,
            "tags": ["dell", "8g"]
        }],
        "switchObjects": [{
            "name": "eq079.atl97",
            "capacityPort": 48,
            "description": "Arista Networks DCS-7050T-52",
            "tags": ["dell", "8g"]
        }],
        "rpmObjects": [{
            "name": "rpm-23",
            "macAddress": "00:0d:5d:10:64:8d",
            "capacityPort": 48,
            "tags": ["dell", "8g"]
        }]
    }
}

Example OS rack:
{
    "name": "cr82XXX",
    "location": {
        "rack": "cr82XXX",
        "zone": "ZONE_CHROMEOS1"
    },
    "capacity_ru": 5,
    "tags": ["dell", "8g"],
    "chromeosRack": {}
}

The protobuf definition can be found here:
Rack:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/rack.proto

Switch, KVM and RPM:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto`

	// RackFileText description for rack file input
	RackFileText string = `Path to a file containing rack specification in JSON format.
This file must contain one rack JSON message

Example Browser rack:
{
    "name": "cr82XXX",
    "location": {
        "rack": "cr82XXX",
        "zone": "ZONE_ATL97"
    },
    "capacity_ru": 5,
    "tags": ["dell", "8g"],
    "chromeBrowserRack": {}
}

Example OS rack:
{
    "name": "cr82XXX",
    "location": {
        "rack": "cr82XXX",
        "zone": "ZONE_CHROMEOS1"
    },
    "capacity_ru": 5,
    "tags": ["dell", "8g"],
    "chromeosRack": {}
}

The protobuf definition of rack is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/rack.proto`

	// ZoneFilterHelpText help text for zone filters for list command
	ZoneFilterHelpText string = fmt.Sprintf("\nValid zone filters: [%s]\n", strings.Join(ufsUtil.ValidZoneStr(), ", "))

	// StateFilterHelpText help text for state filters for list command
	StateFilterHelpText string = fmt.Sprintf("Valid state filters: [%s]\n", strings.Join(ufsUtil.ValidStateStr(), ", "))

	// KeysOnlyText help text for keysOnly option
	KeysOnlyText string = `get only the keys and not the entire object.
Operation will be faster as only primary keys/ids will be retrieved from the service.`

	// FilterText Common filter text for all list filter option
	FilterText string = "filtering option to filter the results.\n"

	// FilterCondition Common filter condition for all list filter option
	FilterCondition string = "\nAll the filter options(separated by comma) are AND and not OR. If you need OR, please run separate list commands."

	// MachineFilterHelp help text for list machine filtering
	MachineFilterHelp string = FilterText + `You can filter machines by kvm/rpm/zone/rack/platform/tag/state` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'nic=nic-1,nic-2'
'zone=atl97 & nic=nic-1'
'zone=atl97 & nic=nic-1 & kvm=kvm-1,kvm-2'` + FilterCondition

	// VMSlotFilterHelp help text for list free vm slots filtering
	VMSlotFilterHelp string = FilterText + `You can filter free vm slots by man(manufacturer)/zone/rack/state/machine/os/vdc(virtualdatacenter)` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'man=apple'
'os=macos,10.13.6'
'zone=atl97 & man=apple'` + FilterCondition

	// RackFilterHelp help text for list rack filtering
	RackFilterHelp string = FilterText + `You can filter racks by tag/zone/state` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'kvm=kvm-1,kvm-2'
'zone=atl97 & kvm=kvm-1'` + FilterCondition

	// NicFilterHelp help text for list rack filtering
	NicFilterHelp string = FilterText + `You can filter nics by zone/rack/switch/switchport/mac(macaddress)/machine/tag` + "\n" + ZoneFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'zone=atl97 & rack=rack-1'` + FilterCondition

	// DracFilterHelp help text for list rack filtering
	DracFilterHelp string = FilterText + `You can filter dracs by zone/rack/switch/switchport/mac(macaddress)/machine/tag` + "\n" + ZoneFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'zone=atl97 & rack=rack-1'` + FilterCondition

	// MachineLSEFilterHelp help text for list machinelse filtering
	MachineLSEFilterHelp string = FilterText + `You can filter hosts by machine/machineprototype/rpm/vlan/servo/zone/rack/switch/tag/state/man(manufacturer)/os/vdc(virtualdatacenter)` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'os=esxi,6.7.0'
'zone=atl97 & rack=rack-1'` + FilterCondition

	// VMFilterHelp help text for list vm filtering
	VMFilterHelp string = FilterText + `You can filter hosts by vlan/state/host/zone/tag/os` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'os=esxi,6.7.0'
'zone=atl97 & vlan=browser-lab:vlan-1'` + FilterCondition

	// KVMFilterHelp help text for list rack filtering
	KVMFilterHelp string = FilterText + `You can filter kvms by zone/rack/platform/tag/mac(macaddress)/state` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'zone=atl97 & rack=rack-1 & platform=p-1'` + FilterCondition

	// RPMFilterHelp help text for list rack filtering
	RPMFilterHelp string = FilterText + `You can filter rpms by zone/rack/tag/state` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'zone=atl97 & rack=rack-1'` + FilterCondition

	// SwitchFilterHelp help text for list rack filtering
	SwitchFilterHelp string = FilterText + `You can filter switches by zone/rack/tag/state` + "\n" + ZoneFilterHelpText + StateFilterHelpText +
		`Filter format Egs:
'zone=atl97'
'zone=atl97 & rack=rack-1'` + FilterCondition

	// MachineLSEPrototypeFilterHelp help text for list MachineLSEPrototype filtering
	MachineLSEPrototypeFilterHelp string = FilterText + `You can filter machineprototypes by tag
Filter format Egs:
'tag=acs,wificell'
'tag=browser` + FilterCondition

	// RackLSEPrototypeFilterHelp help text for list RackLSEPrototype filtering
	RackLSEPrototypeFilterHelp string = FilterText + `You can filter rackprototypes by tag
Filter format Egs:
'tag=acs'
'tag=browser` + FilterCondition

	// ChromePlatformFilterHelp help text for list ChromePlatform filtering
	ChromePlatformFilterHelp string = FilterText + `You can filter platforms by man(manufacturer)/tag
Filter format Egs:
'tag=dell, 8g'
'tag=iphone & man=Apple` + FilterCondition

	// VlanFilterHelp help text for list vlan filtering
	VlanFilterHelp string = FilterText + `You can filter vlans by state` + StateFilterHelpText +
		`Filter format Egs:
'state=serving'` + FilterCondition

	// StateHelp help text for filter '-state'
	StateHelp string = "the state to assign this entity to. Valid state strings: [" + strings.Join(ufsUtil.ValidStateStr(), ", ") + "]"

	//ClearFieldHelpText help text to clear field using field mask in update cmds
	ClearFieldHelpText string = "To clear this field and set it to empty, assign '" + utils.ClearFieldValue + "'"

	//ZoneHelpText help text for zone command line options
	ZoneHelpText string = fmt.Sprintf("the name of the zone. "+
		"You can either use the below strings or prefix \"ZONE_\" to the below strings(for JSON input) to specify the exact enum name. "+
		"Valid zone strings: [%s]", strings.Join(ufsUtil.ValidZoneStr(), ", "))
)
