// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmdhelp

import (
	"fmt"
	"strings"

	"infra/cmd/shivas/utils"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsUtil "infra/unifiedfleet/app/util"
)

var (
	// ListPageSizeDesc description for List PageSize
	ListPageSizeDesc string = `number of items to get. The service may return fewer than this value.`

	//AddSwitchLongDesc long description for AddSwitchCmd
	AddSwitchLongDesc string = `Create a switch to UFS.

Examples:
shivas add switch -f switch.json
Adds a switch by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.

shivas add switch -rack {Rack name} -name {switch name} -capacity {50} -description {description}
Adds a switch by specifying several attributes directly.

shivas add switch -i
Adds a switch by reading input through interactive mode.`

	// AddAssetLongDesc long description for AddAssetCmd
	AddAssetLongDesc string = `Add an asset to UFS
Examples:
shivas add asset -f asset.json
Adds an asset by reading a JSON file input.

shivas add asset -name {asset name} -location {asset location} -type DUT
Adds an asset by specifying several attributes directly

shivas add asset -name {asset name} -lab {lab name} -aisle {aisle} -row {row number}-rack {rack name} -type {asset type} -position {asset position}
Alternate location specification for finer details.`

	// UpdateAssetLongDesc long description for UpdateAssetCmd
	UpdateAssetLongDesc string = `Update an asset by name.

Examples:
shivas update asset -f asset.json
Update a asset by reading a JSON file input.

shivas update asset -name asset1 -zone mtv97 -rack rack1
Partial updates a asset by parameters. Only specified parameters will be udpated in the asset.`

	// AddDUTLongDesc long description for AddDUTCmd
	AddDUTLongDesc string = `Add and deploy a DUT.
Examples:
shivas add dut -name {hostname} -asset {asset tag} -servo {servo host}:{servo port} -servo-serial {servo serial} -pool {dut pool}
Adds a DUT to UFS and triggers a swarming job to deploy the DUT.


shivas add dut -name {hostname} -pool {dut pool} -ignore-ufs=true
Triggers a swarming job to deploy the DUT. Avoids updating to UFS.

shivas add dut -f dut.json
Adds a DUT to UFS using a json description file and triggers a swarming job to deploy the DUT.

Note:
1. UFS assigns a servo port if it's not given.
2. Peripherals like camera, audio, wifi, etc,. are only supported through json input.
3. By default, every deploy task runs update-label, verify-recovery-mode and run-pre-deploy-verification actions on the DUT.
`

	// AddLabstationLongDesc long description for AddLabstationCmd
	AddLabstationLongDesc string = `Add and deploy a Labstation.
Examples:
shivas add labstation -name {hostname} -asset {asset tag} -pool {labstation pool}
Adds a Labstation to UFS.

shivas add labstation -name {hostname} -pool {labstation pool} -rpm {rpm host} -rpm-outlet {rpm outlet}
Adds a labstation to UFS with rpm.

shivas add labstation -f labstation.json
Adds a Labstation to UFS using a json description file.

shivas add labstation -f labstation.json
Adds Labstation(s) to UFS using a csv description file.
`
	// UpdateLabstationLongDesc long description for UpdateLabstationCmd
	UpdateLabstationLongDesc string = `Update and/or deploy a Labstation.
Examples:
shivas update labstation -name {hostname} -rpm {rpm host} -outlet {rpm outlet}
Update RPM connected to the Labstation.

shivas update labstation -name {hostname} -rpm -
Delete RPM connected to the Labstation.

shivas update labstation -name {hostname} -pool {labstation pool}
Update pool assigned to the Labstation.

shivas update labstation -f {json/csv file}
Update Labstation(s) with input from JSON/CSV file.
`

	// DUTRegistrationFileText description for json file input
	DUTRegistrationFileText string = `[JSON/MCSV Mode] Path to a file(.json/.csv) containing DUT specification.

[JSON Mode]
The file must contain one DUT specification.
Example DUT:
{
	"name": "chromeos1-row1-rack11-host4",
	"machineLsePrototype": "atl:standard",
	"hostname": "chromeos1-row1-rack11-host4",
	"chromeosMachineLse": {
		"deviceLse": {
			"dut": {
				"hostname": "chromeos1-row1-rack11-host4",
				"peripherals": {
					"servo": {
						"servoHostname": "chromeos1-row1-rack11-labstation",
						"servoPort": 9904
					},
					"rpm": {
						"powerunitName": "chromeos1-row2-rack3-rpm",
						"powerunitOutlet": ".A1"
					}
				},
				"pools": [
						"DUT_POOL_QUOTA"
				]
			}
		}
	},
	"machines": [
		"1156928"
	],
	"deploymentTicket": "crbug.com/123456",
	"description": "Fixed and replaced"
}

Example DUT with peripherals:
{
	"name": "chromeos1-row1-rack11-host4",
	"machineLsePrototype": "atl:standard",
	"hostname": "chromeos1-row1-rack11-host4",
	"chromeosMachineLse": {
		"deviceLse": {
			"dut": {
				"hostname": "chromeos1-row1-rack11-host4",
				"peripherals": {
					"servo": {
						"servoHostname": "chromeos1-row1-rack11-labstation",
						"servoPort": 9904,
					},
					"rpm": {
						"powerunitName": "chromeos1-row2-rack3-rpm",
						"powerunitOutlet": ".A1"
					},
					"chameleon": {
						"chameleon_peripherals": [1],
						"audioBoard": false,
					},
					"connectedCamera": [{
						"cameraType": 1,
					}],
					"audio": {
						"audioBox": false,
						"atrus": false,
						"audioCable": true,
					},
					"wifi": {
						"wificell": true,
						"antennaConn": 1,
						"router": 1,
					},
					"touch": {
						"mimo": false,
					},
					"carrier": "att",
					"camerabox": false,
					"chaos": false,
					"cable": [{
						"type": 1,
					}],
					"cameraboxInfo": {
						"facing": 1,
						"light": 1,
					},
					"smartUsbhub": false
				},
				"pools": [
						"faft_test_debug"
				]
			}
		}
	},
	"machines": [
		"1156928"
	],
	"deploymentTicket": "crbug.com/123456",
	"description": "Fixed and replaced",
}

The protobuf definition of machineLSE is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine_lse.proto

[MCSV Mode]
The file may have multiple or one dut csv record.
The header format and sequence should be: [name,asset,servo_host,servo_port,servo_serial,rpm_host,rpm_outlet,pools]
Example mcsv format:
name,asset,servo_host,servo_port,servo_serial,servo_setup,rpm_host,rpm_outlet,pools
dut-1,asset-1,servo-1,9998,ServoXdw,REGULAR,rpm-1,23,"CTS QUOTA"
dut-2,asset-2,servo-2,9998,ServoYdw,,rpm-2,43,QUOTA
dut-3,asset-3,chromeos6-row2-rack3-host4-servo,,,,,,,`

	// DUTUpdateFileText description for json file input
	DUTUpdateFileText string = `Path to a file(.json) containing DUT specification.

[JSON Mode]
The file must contain one DUT specification. This specification overwrites everything. Empty value for a field will assign
nil/default value.
Example DUT:
{
	"name": "chromeos1-row1-rack11-host4",
	"machineLsePrototype": "atl:standard",
	"hostname": "chromeos1-row1-rack11-host4",
	"chromeosMachineLse": {
		"deviceLse": {
			"dut": {
				"hostname": "chromeos1-row1-rack11-host4",
				"peripherals": {
					"servo": {
						"servoHostname": "chromeos1-row1-rack11-labstation",
						"servoPort": 9998,
					},
					"rpm": {
						"powerunitName": "chromeos1-row2-rack3-rpm",
						"powerunitOutlet": ".A1"
					},
				},
				"pools": [
						"DUT_POOL_QUOTA"
				]
			}
		}
	},
	"machines": [
		"1156928"
	],
	"deploymentTicket": "crbug.com/123456",
	"description": "Fixed and replaced",
}

Example DUT with peripherals:
{
	"name": "chromeos1-row1-rack11-host4",
	"machineLsePrototype": "atl:standard",
	"hostname": "chromeos1-row1-rack11-host4",
	"chromeosMachineLse": {
		"deviceLse": {
			"dut": {
				"hostname": "chromeos1-row1-rack11-host4",
				"peripherals": {
					"servo": {
						"servoHostname": "chromeos1-row1-rack11-labstation",
						"servoPort": 9904,
						"servoSerial": "C126246346"
					},
					"rpm": {
						"powerunitName": "chromeos1-row2-rack3-rpm",
						"powerunitOutlet": ".A1"
					},
					"chameleon": {
						"chameleon_peripherals": [1],
						"audioBoard": false,
					},
					"connectedCamera": [{
						"cameraType": 1,
					}],
					"audio": {
						"audioBox": false,
						"atrus": false,
						"audioCable": true,
					},
					"wifi": {
						"wificell": true,
						"antennaConn": 1,
						"router": 1,
					},
					"touch": {
						"mimo": false,
					},
					"carrier": "att",
					"camerabox": false,
					"chaos": false,
					"cable": [{
						"type": 1,
					}],
					"cameraboxInfo": {
						"facing": 1,
						"light": 1,
					},
					"smartUsbhub": false
				},
				"pools": [
					"DUT_POOL_QUOTA"
				]
			}
		}
	},
	"machines": [
		"1156928"
	],
	"deploymentTicket": "crbug.com/123456",
	"description": "Fixed and replaced",
}

Note: Changing any servo parameters triggers a deploy task. If you don't wish to update anything but trigger a deploy task on it,
clear servoType by setting it to '-'. For example:
		"servo": {
			"servoHostname": "chromeos1-row1-rack11-labstation",
			"servoPort": 9904,
			"servoSerial": "C126246346",
			"servoType": "-"
		}

The protobuf definition of machine lse is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine_lse.proto
The protobuf definition of DeviceUnderTest is a part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/chromeos/lab/device.proto

The file may have multiple or one dut csv record.
The header format and sequence should be: [name,asset,servo_host,servo_port,servo_serial,rpm_host,rpm_outlet,pools]

Example mcsv format:
name,asset,servo_host,servo_port,servo_serial,servo_setup,rpm_host,rpm_outlet,pools
dut-1,asset-1,servo-1,9998,servo-serial-1,DUAL_V4,rpm-1,22,QUOTA
dut-2,,,9998,,,,,
dut-3,asset-1,,,,,,,
dut-4,,,,,,rpm-1,22,

Example mcsv format (delete/clear support. Use - to clear a field where available):
dut-6,,-,9998,servo-serial-1,DUAL_V4,,,
dut-7,,,,,,-,,
dut-8,asset-2,,,,,-,,
dut-9,,,,,,-,,"QUOTA CQ"
`

	// LabstationRegistrationFileText description for json file input
	LabstationRegistrationFileText string = `Path to a file(.json) containing Labstation specification.

[JSON Mode]
The file must contain one DUT specification. This specification overwrites everything. Empty value for a field will assign
nil/default value.
Example Labstation:
{
        "name": "chromeos6-row10-rack22-labstation1",
        "machineLsePrototype": "atl:labstation",
        "chromeosMachineLse": {
                "deviceLse": {
                        "labstation": {
                                "rpm": {
                                        "powerunitName": "chromeos6-row9_10-rack22-rpm3",
                                        "powerunitOutlet": "AA3"
                                },
                                "pools": [
                                        "labstation_main"
                                ]
                        }
                }
        },
        "tags": ["fizz_labstation", "chromeos6-row10"],
        "zone": "ZONE_CHROMEOS6",
        "deploymentTicket": "crbug.com/c/35007",
        "description": "CrOS6-ro10-r22 labstation",
}

The protobuf definition of machine lse is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine_lse.proto

The protobuf definition of Labstation is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/chromeos/lab/device.proto

The file may have multiple or one labstation csv record.
The header format and sequence should be: [name,asset,rpm_host,rpm_outlet,pools]

Example mcsv format:
name,asset,rpm_host,rpm_outlet,pools
labstation-1,asset-1,rpm-1,A2,labstation_main
labstation-2,asset-2,rpm-2,A2,"labstation_main labstation_tryjob"
labstation-3,asset-3,rpm-3,A2,labstation_main

`
	// LabstationUpdateFileText description for json file input
	LabstationUpdateFileText string = `Path to a file(.json) containing Labstation specification.

[JSON Mode]
The file must contain one DUT specification. This specification overwrites everything. Empty value for a field will assign
nil/default value.
Example Labstation:
{
        "name": "chromeos6-row10-rack22-labstation1",
        "machineLsePrototype": "atl:labstation",
        "chromeosMachineLse": {
                "deviceLse": {
                        "labstation": {
                                "rpm": {
                                        "powerunitName": "chromeos6-row9_10-rack22-rpm3",
                                        "powerunitOutlet": "AA3"
                                },
                                "pools": [
                                        "labstation_main"
                                ]
                        }
                }
        },
        "tags": ["fizz_labstation", "chromeos6-row10"],
        "zone": "ZONE_CHROMEOS6",
        "deploymentTicket": "crbug.com/c/35007",
        "description": "CrOS6-ro10-r22 labstation",
}

The protobuf definition of machine lse is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine_lse.proto

The protobuf definition of Labstation is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/chromeos/lab/device.proto

The file may have multiple or one labstation csv record.
The header format and sequence should be: [name,asset,rpm_host,rpm_outlet,pools]

Example mcsv format:
name,asset,rpm_host,rpm_outlet,pools
labstation-1,,rpm-1,A2,labstation_main
labstation-2,asset-1,,,
labstation-3,,,,"labstation_main labstation_tryjob"

Example mcsv format (delete/clear support. Use - to clear a field where available):
name,asset,rpm_host,rpm_outlet,pools
labstation-1,,-,,labstation_main
labstation-2,asset-1,-,,
`

	// UpdateDUTLongDesc long description for UpdateDUTCmd
	UpdateDUTLongDesc string = `Update a DUT by name. This runs a deploy task on the updated DUT by default.

Examples:
shivas update dut -f dut.json
Update a DUT by reading a JSON file input. Triggers deploy task if required.

shivas update dut -f dut.json -force-deploy
Update DUT to UFS if changed. Trigger a deploy task.

shivas update dut -name chromeos6-rack3-row2-host1 -force-deploy
Trigger a deploy task on the given DUT. Nothing is updated.

shivas update dut -name chromeos6-rack3-row2-host1 -servo chromeos6-rack3-row2-labstation1:0 -servo-serial C1024356789
Update servo connected to the DUT.

shivas update dut -name chromeos6-rack3-row2-host1 -rpm chromeos6-row11_12-rack24-rpm1 -outlet .A22
Update rpm connected to the DUT.

shivas update dut -name chromeos6-rack3-row2-host1 -servo -
Delete servo connected to the DUT.

shivas update dut -name chromeos6-rack3-row2-host1 -rpm -
Delete rpm connected to the DUT.

shivas update dut -name chromeos6-rack3-row2-host1 -tags kevin,no-test
Add tags to an existing DUT.

shivas update dut -name chromeos6-rack3-row2-host1 -tags -
Delete tags to an existing DUT.

`
	// UpdateSwitchLongDesc long description for UpdateSwitchCmd
	UpdateSwitchLongDesc string = `Update a switch by name.

Examples:
shivas update switch -f switch.json
Update a switch by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.

shivas update switch -i
Update a switch by reading input through interactive mode.

shivas update switch -rack {Rack name} -name {switch name} -capacity {50} -description {description}
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
	SwitchFileText string = `[JSON/MCSV Mode] Path to a file(.json/.csv) containing switch specification.

[JSON Mode]
This file must contain one switch JSON message
Example switch:
{
    "name": "eq079.atl97",
    "capacityPort": 48,
    "description": "Arista Networks DCS-7050T-52",
    "tags": ["dell", "8g"],
    "rack": "cr-22"
}

[MCSV Mode]
The file may have multiple or one switch csv record
The header format and sequence should be: [name,rack,capacity,desc,tags]
Example mcsv format:
name,rack,capacity,desc,tags
switch-2,rack-2,hello-1,Dell Power
switch-3,rack-2,"hello,world, this is ufs",Apple Pro Power

The protobuf definition of switch is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/peripherals.proto`

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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine_lse.proto`

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

You can create a machine with required parameters to UFS, and later add nics/drac separately by using add nic/add drac commands.

You can also provide the optional nics and drac information to create the nics and drac associated with this machine by specifying a json file as input.

Examples:
shivas add machine -f machinerequest.json
Creates a machine by reading a JSON file input.

shivas add machine -name machine1 -zone mtv97 -rack rack1 -ticket b/1234 -platform platform1 -kvm kvm1
Creates a machine by parameters without adding nic/drac.`

	// UpdateMachineLongDesc long description for UpdateMachineCmd
	UpdateMachineLongDesc string = `Update a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.) by name.

Examples:
shivas update machine -f machine.json
Update a machine by reading a JSON file input.

shivas update machine -i
Update a machine by reading input through interactive mode.

shivas update machine -name machine1 -zone mtv97 -rack rack1
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine.proto

Drac:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/peripherals.proto

Nic:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/network.proto`

	// AssetFileText description for asset file input
	AssetFileText string = `Path to a file containing asset specification in JSON format.
This file must contain one asset JSON message

Example OS asset:
{
	"name":  "test-1",
	"type":  "DUT",
	"model":  "atlas",
	"location":  {
		"aisle":  "1",
		"row":  "2",
		"rack":  "rack-23",
		"rackNumber":  "23",
		"shelf":  "3",
		"position":  "5",
		"barcodeName":  "bar",
		"zone":  "ZONE_CHROMEOS6"
	},
	"info":  {
		"assetTag":  "",
		"serialNumber":  "fer3-rtgd",
		"costCenter":  "cros",
		"googleCodeName":  "kohaku",
		"model":  "atlas",
		"buildTarget":  "zuko",
		"referenceBoard":  "atlas",
		"ethernetMacAddress":  "11:22:33:44:55:66",
		"sku":  "are",
		"phase":  "EVT",
		"hwid":  ""
	},
	"updateTime":  "2020-12-29T23:54:45.437251068Z",
	"realm":  "@internal:ufs/os-atl"
}

The protobuf definition of asset is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/asset.proto`

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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine.proto`

	// AddHostLongDesc long description for AddHostCmd
	AddHostLongDesc string = `Add a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine

Examples:
shivas add host -f host.json
Adds a host by reading a JSON file input.
[WARNING]: machines is a required field in json, all other output only fields will be ignored.
Specify additional settings, e.g. vlan, nic, ip via command line parameters along with JSON input

shivas add host -machine machine0 -name host0 -prototype browser:no-vm  -osversion chrome-version-0 -vm-capacity 3
Adds a host by parameters without adding vms.

shivas add host -i
Adds a host by reading input through interactive mode.`

	// UpdateHostLongDesc long description for UpdateHostCmd
	UpdateHostLongDesc string = `Update a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine

Examples:
shivas update host -f host.json
Updates a host by reading a JSON file input.
[WARNING]: machines is a required field in json, all other output only fields will be ignored.
Specify additional settings, e.g. vlan, ip, nic, state via command line parameters along with JSON input

shivas update host -name cr22 -os windows
Partial update a host by parameters. Only specified parameters will be updated in the host.

shivas update host -name host0 -delete-vlan
Remove the ip for host

shivas update host -name host0 -vlan browser:11 -nic eth0
Assign ip to the host

shivas update host -i
Updates a host by reading input through interactive mode.`

	// MachineLSEFileText description for machinelse/host file input
	MachineLSEFileText string = `[JSON mode] Path to a file containing host specification in JSON format.
This file must contain one machine deployment JSON message

Example host for a browser machine:
{
    "name": "esx-380XXX",
    "machineLsePrototype": "browser:vm",
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
    "machineLsePrototype": "acs:wifi",
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
    "machineLsePrototype": "acs:qwer",
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/machine_lse.proto`

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
shivas add machine-prototype -f machineprototype.json
Adds a machine prototype by reading a JSON file input.

shivas add machine-prototype -i
Adds a machine prototype by reading input through interactive mode.`

	// UpdateMachineLSEPrototypeLongDesc long description for UpdateMachineLSEPrototypeCmd
	UpdateMachineLSEPrototypeLongDesc string = `Update prototype for machine deployment.

Examples:
shivas update machine-prototype -f machineprototype.json
Updates a machine prototype by reading a JSON file input.

shivas update machine-prototype -i
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
    "name": "browser:vm",
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/lse_prototype.proto#29`

	// AddRackLSEPrototypeLongDesc long description for AddRackLSEPrototypeCmd
	AddRackLSEPrototypeLongDesc string = `Add prototype for rack deployment.

Examples:
shivas add rack-prototype -f rackprototype.json
Adds a rack prototype by reading a JSON file input.

shivas add rack-prototype -i
Adds a rack prototype by reading input through interactive mode.`

	// UpdateRackLSEPrototypeLongDesc long description for UpdateRackLSEPrototypeCmd
	UpdateRackLSEPrototypeLongDesc string = `Update prototype for rack deployment.

Examples:
shivas update rack-prototype -f rackprototype.json
Updates a rack prototype by reading a JSON file input.

shivas update rack-prototype -i
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
    "name": "browser:vm",
    "peripheralRequirements": [{
        "peripheralType": "PERIPHERAL_TYPE_SWITCH",
        "min": 5,
        "max": 7
    }],
    "tags": ["dell", "8g"]
}

The protobuf definition of prototype for rack deployment is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/lse_prototype.proto`

	// AddChromePlatformLongDesc long description for AddChromePlatformCmd
	AddChromePlatformLongDesc string = `Add platform configuration for browser machine.

Examples:
shivas add platform -f platform.json
Adds a platform by reading a JSON file input.

shivas add platform -i
Adds a platform by reading input through interactive mode.

shivas add platform -name DELL_R320 -manufacturer Dell -tags 'dell,8g' -desc 'Dell platform'
Adds a platform by specifying several attributes directly`

	// AddVlanLongDesc long description for AddVlanCmd
	AddVlanLongDesc string = `Add vlans.

Examples:
shivas add vlan -name browser:100 -cidr-block A.B.C.D/24 -desc "atl97-vlan"
Adds a vlan by specifying several attributes directly`

	// UpdateChromePlatformLongDesc long description for UpdateChromePlatformCmd
	UpdateChromePlatformLongDesc string = `Update platform configuration for browser machine.

Examples:
shivas update platform -f platform.json
Updates a platform by reading a JSON file input.

shivas update platform -i
Updates a platform by reading input through interactive mode.

shivas update platform -name DELL_R320 -manufacturer Dell -tags -'
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

shivas update vlan -name vlan_name -desc test -state serving'
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/chrome_platform.proto`

	// AddNicLongDesc long description for AddNicCmd
	AddNicLongDesc string = `Add a nic to UFS.

Examples:
shivas add nic -f nic.json
Add a nic by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas add nic -name machine0:eth0 -switch switch0 -mac 123456 -machine machine0 -switch-port 1
Add a nic by specifying several attributes directly.

shivas add nic -i
Add a nic by reading input through interactive mode.`

	// UpdateNicLongDesc long description for UpdateNicCmd
	UpdateNicLongDesc string = `Update a nic by name.

Examples:
shivas update nic -f nic.json
Update a nic by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas update nic -i
Update a nic by reading input through interactive mode.

shivas update nic -name machine0:eth0 -switch switch0 -mac 12345
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/network.proto`

	// AddDracLongDesc long description for AddDracCmd
	AddDracLongDesc string = `Add a drac to UFS.

Examples:
shivas add drac -f drac.json
Add a drac by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas add drac -name machine0:drac -switch switch0 -mac 123456 -machine machine0 -switch-port 1
Add a drac by specifying several attributes directly.

shivas add drac -i
Add a drac by reading input through interactive mode.
`

	// UpdateDracLongDesc long description for UpdateDracCmd
	UpdateDracLongDesc string = `Update a drac by name.

Examples:
shivas update drac -f drac.json
Update a drac by reading a JSON file input.
[WARNING]: machine is a required field in json, all other output only fields will be ignored.

shivas update drac -name machine0:drac -switch switch0 -mac 123456
Partial update a drac by parameters. Only specified parameters will be updated in the drac.

shivas update drac -name drac0 -delete-vlan
Remove the ip for drac0

shivas update drac -name drac0 -vlan browser:11
Assign ip to the drac

shivas update drac -i
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/peripherals.proto`

	// AddKVMLongDesc long description for AddKVMCmd
	AddKVMLongDesc string = `Add a kvm to UFS.

Examples:
shivas add kvm -f kvm.json
Add a kvm by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.

shivas add kvm -rack {Rack name} -name {kvm name} -mac {mac} -platform {platform}
Add a kvm by specifying several attributes directly.

shivas add kvm -i
Add a kvm by reading input through interactive mode.`

	// UpdateKVMLongDesc long description for UpdateKVMCmd
	UpdateKVMLongDesc string = `Update a kvm by name.

Examples:
shivas update kvm -f kvm.json
Update a kvm by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.
Specify additional settings, e.g. vlan, ip via command line parameters along with JSON input

shivas update kvm -i
Update a kvm by reading input through interactive mode.

shivas update kvm -rack {Rack name} -name {kvm name} -mac {mac} -platform {platform}
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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/peripherals.proto`

	// AddRackLongDesc long description for AddRackCmd
	AddRackLongDesc string = `Create a rack to UFS.
You can create a rack with name and zone to UFS, and later add kvm/switch/rpm separately by using add switch/add kvm/add rpm commands.

You can also provide the optional switches, kvms and rpms information to create the switches, kvms and rpms associated with this rack by specifying a json file as input.

Examples:
shivas add rack -f rackrequest.json
Creates a rack by reading a JSON file input.

shivas add rack -name rack-123 -zone lab01 -capacity 10
Creates a rack by parameters without adding kvm/switch/rpm.`

	// UpdateRackLongDesc long description for UpdateRackCmd
	UpdateRackLongDesc string = `Update a rack by name.

Examples:
shivas update rack -f rack.json
Update a rack by reading a JSON file input.

shivas update rack -i
Update a rack by reading input through interactive mode.

shivas update rack -name rack-123 -zone lab01 -capacity 10
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
	RackRegistrationFileText string = `[JSON/MCSV Mode] Path to a file(.json/.csv) containing rack creation request specification.

[JSON Mode]
This file must contain required rack field and optional switches/kvms/rpms fields.
Example json browser rack creation request:
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

Example json OS rack:
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

[MCSV Mode]
The file may have multiple or one rack csv record.
The header format and sequence should be: [name,zone,capacity_ru,desc,tags]
Example mcsv format:
name,zone,capacity_ru,desc,tags
rack-2,chromeos2,12,hello-1,Dell Power
rack-3,chromeos2,13,"hello,world, this is ufs",Apple Pro Power

The protobuf definition can be found here:
Rack:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/rack.proto

Switch, KVM and RPM:
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/peripherals.proto`

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
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/rack.proto`

	// ZoneFilterHelpText help text for zone filters for list command
	ZoneFilterHelpText string = fmt.Sprintf("\nValid zone filters: [%s]\n", strings.Join(ufsUtil.ValidZoneStr(), ", "))

	// DeviceTypeFilterHelpText help text for devicetype filters for list command
	DeviceTypeFilterHelpText string = fmt.Sprintf("\nValid devicetype filters: [%s]\n", strings.Join(ufsUtil.ValidDeviceTypeStr(), ", "))

	// AssetTypesHelpText help text for asset type filters
	AssetTypesHelpText string = fmt.Sprintf("\nValid type filters [%s]", strings.Join(ufsUtil.ValidAssetTypeStr(), ", "))

	// StateFilterHelpText help text for state filters for list command
	StateFilterHelpText string = fmt.Sprintf("Valid state filters: [%s]\n", strings.Join(ufsUtil.ValidStateStr(), ", "))

	// KeysOnlyText help text for keysOnly option
	KeysOnlyText string = `prints only the keys in table format (without title)
-keys -json prints the entire JSON object, but only name/id field will be filled, other fields will be empty
-keys -json -noemit prints JSON object with only name/id field.
Operation will be faster as only name/id will be retrieved from the service.`

	// StateHelp help text for filter '-state'
	StateHelp string = "the state to assign this entity to. Valid state strings: [" + strings.Join(ufsUtil.ValidStateStr(), ", ") + "]"

	//ClearFieldHelpText help text to clear field using field mask in update cmds
	ClearFieldHelpText string = "To clear this field and set it to empty, assign '" + utils.ClearFieldValue + "'"

	//ZoneHelpText help text for zone command line options
	ZoneHelpText string = fmt.Sprintf("the name of the zone. "+
		"You can either use the below strings or prefix \"ZONE_\" to the below strings(for JSON input) to specify the exact enum name. "+
		"Valid zone strings: [%s]", strings.Join(ufsUtil.ValidZoneStr(), ", "))

	//ChameleonTypeHelpText help text for chameleontype command line options
	ChameleonTypeHelpText string = fmt.Sprintf("the name of the chameleontype. Can specify multiple comma separated values. "+
		"Valid ChameleonType strings: [%s]", strings.Join(ufsUtil.ValidChameleonTypeStr(), ", "))

	//CameraTypeHelpText help text for cameratype command line options
	CameraTypeHelpText string = fmt.Sprintf("the name of the cameratype. Can specify multiple comma separated values. "+
		"Valid CameraType strings: [%s]", strings.Join(ufsUtil.ValidCameraTypeStr(), ", "))

	//AntennaConnectionHelpText help text for antennaconnection command line options
	AntennaConnectionHelpText string = fmt.Sprintf("the name of the wifi antennaconnection. "+
		"Valid AntennaConnection strings: [%s]", strings.Join(ufsUtil.ValidAntennaConnectionStr(), ", "))

	//RouterHelpText help text for router command line options
	RouterHelpText string = fmt.Sprintf("the name of the wifi router. "+
		"Valid Router strings: [%s]", strings.Join(ufsUtil.ValidRouterStr(), ", "))

	//FacingHelpText help text for facing command line options
	FacingHelpText string = fmt.Sprintf("the name of the camerabox info facing. "+
		"Valid Facing strings: [%s]", strings.Join(ufsUtil.ValidFacingStr(), ", "))

	//LightHelpText help text for light command line options
	LightHelpText string = fmt.Sprintf("the name of the camerabox info light. "+
		"Valid Light strings: [%s]", strings.Join(ufsUtil.ValidLightStr(), ", "))

	//CableTypeHelpText help text for cabletype command line options
	CableTypeHelpText string = fmt.Sprintf("the name of the cabletype. Can specify multiple comma separated values. "+
		"Valid CableType strings: [%s]", strings.Join(ufsUtil.ValidCableTypeStr(), ", "))

	// AddRPMLongDesc long description for AddRPMCmd
	AddRPMLongDesc string = `Add a rpm to UFS.

Examples:
shivas add rpm -f rpm.json
Add a rpm by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.

shivas add rpm -rack {Rack name} -name {rpm name} -mac {mac} -capacity {50} -desc {description}
Add a rpm by specifying several attributes directly.

shivas add rpm -i
Add a rpm by reading input through interactive mode.`

	// UpdateRPMLongDesc long description for UpdateRPMCmd
	UpdateRPMLongDesc string = `Update a rpm by name.

Examples:
shivas update rpm -f rpm.json
Update a rpm by reading a JSON file input.
[WARNING]: rack is a required field in json, all other output only fields will be ignored.
Specify additional settings, e.g. vlan, ip via command line parameters along with JSON input

shivas update rpm -i
Update a rpm by reading input through interactive mode.

shivas update rpm -rack {Rack name} -name {rpm name} -mac {mac}
Partial updates a rpm by parameters. Only specified parameters will be udpated in the rpm.`

	// RPMFileText description for rpm file input
	RPMFileText string = `[JSON/MCSV Mode] Path to a file(.json/.csv) containing rpm specification.

[JSON Mode]
This file must contain one rpm JSON message
Example json rpm:
{
    "name": "cx101-rpm1XXX",
    "macAddress": "00:0d:5d:0f:54:ed",
    "tags": ["dell", "8g"],
    "capacityPort": 48,
    "rack": "cr-22"
}

[MCSV Mode]
The file may have multiple or one rpm csv record.
The header format and sequence should be: [name,rack,mac,capacity,desc,tags]
Example mcsv format:
name,rack,mac,capacity,desc,tags
rpm-2,chromeos2,11:11:11:11:11:11,12,hello-1,Dell Power
rpm-3,chromeos2,22:22:22:22:22:22,13,"hello,world, this is ufs",Apple Pro Power

The protobuf definition of rpm is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/models/peripherals.proto`
)

// ServoSetupTypeAllowedValuesString returns a string description of all allowed values for servo setup type.
func ServoSetupTypeAllowedValuesString() string {
	servoSetupTypeAllowedValueList := []string{}
	for name := range chromeosLab.ServoSetupType_value {
		servoSetupTypeAllowedValueList = append(servoSetupTypeAllowedValueList, strings.TrimPrefix(name, "SERVO_SETUP_"))
	}
	return fmt.Sprintf("[%s]", strings.Join(servoSetupTypeAllowedValueList, ", "))
}
