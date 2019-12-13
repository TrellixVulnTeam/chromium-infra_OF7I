// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inventory

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"golang.org/x/net/context"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/config"
	dataSV "infra/appengine/crosskylabadmin/app/frontend/internal/datastore/stableversion"
	"infra/appengine/crosskylabadmin/app/frontend/internal/fakes"
	"infra/appengine/crosskylabadmin/app/frontend/internal/gitstore"
	"infra/libs/skylab/inventory"
)

const (
	gpu = "fakeGPU"
	// dut should follow the following rules:
	// 1) entries should be in alphabetical order.
	// 2) indent is 2 spaces, no tabs.
	dut = `duts {
  common {
    environment: ENVIRONMENT_STAGING
    hostname: "dut_hostname"
    id: "dut_id_1"
    labels {
      capabilities {
        carrier: CARRIER_INVALID
        gpu_family: "%s"
        graphics: ""
        power: ""
        storage: ""
      }
      critical_pools: DUT_POOL_SUITES
      model: "link"
      peripherals {
      }
    }
  }
}
`

	emptyStableVersions = `{
	"cros": [],
	"faft": [],
	"firmware": []
}`

	stableVersions = `{
    "cros":[
        {
            "key":{
                "buildTarget":{
                    "name":"auron_paine"
                },
                "modelId":{
                    "value":""
                }
            },
            "version":"R78-12499.40.0"
        }
    ],
    "faft":[
        {
            "key": {
                "buildTarget": {
                    "name": "auron_paine"
                },
                "modelId": {
                    "value": "auron_paine"
                }
            },
            "version": "auron_paine-firmware/R39-6301.58.98"
        }
    ],
    "firmware":[
        {
            "key": {
                "buildTarget": {
                    "name": "auron_paine"
                },
                "modelId": {
                    "value": "auron_paine"
                }
            },
            "version": "Google_Auron_paine.6301.58.98"
        }
    ]
}`

	stableVersionWithEmptyVersions = `{
    "cros":[
        {
            "key":{
                "buildTarget":{
                    "name":"auron_paine"
                },
                "modelId":{
                    "value":""
                }
            },
            "version":""
        }
    ],
    "faft":[
        {
            "key": {
                "buildTarget": {
                    "name": "auron_paine"
                },
                "modelId": {
                    "value": "auron_paine"
                }
            },
            "version": ""
        }
    ],
    "firmware":[
        {
            "key": {
                "buildTarget": {
                    "name": "auron_paine"
                },
                "modelId": {
                    "value": "auron_paine"
                }
            },
            "version": ""
        }
    ]
}`
)

func fakeDeviceConfig(ctx context.Context, ids []DeviceConfigID) map[string]*device.Config {
	deviceConfigs := map[string]*device.Config{}
	for _, id := range ids {
		dcID := getDeviceConfigIDStr(ctx, id)
		deviceConfigs[dcID] = &device.Config{
			Id: &device.ConfigId{
				PlatformId: &device.PlatformId{
					Value: id.PlatformID,
				},
				ModelId: &device.ModelId{
					Value: id.ModelID,
				},
				VariantId: &device.VariantId{
					Value: id.VariantID,
				},
				BrandId: &device.BrandId{
					Value: "",
				},
			},
			GpuFamily: gpu,
		}
	}
	return deviceConfigs
}

func TestUpdateDeviceConfig(t *testing.T) {
	Convey("Update DUTs with empty device config", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		deviceConfigs := map[string]*device.Config{}
		tf.FakeGitiles.SetInventory(
			config.Get(ctx).Inventory,
			fakes.InventoryData{
				Lab: []byte(fmt.Sprintf(dut, gpu)),
			},
		)
		store := gitstore.NewInventoryStore(tf.FakeGerrit, tf.FakeGitiles)
		err := store.Refresh(ctx)
		So(err, ShouldBeNil)
		url, err := updateDeviceConfig(tf.C, deviceConfigs, store)
		So(err, ShouldBeNil)
		So(url, ShouldNotContainSubstring, config.Get(ctx).Inventory.GerritHost)
	})

	Convey("Update DUTs as device config changes", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		id := DeviceConfigID{
			PlatformID: "",
			ModelID:    "link",
			VariantID:  "",
		}
		deviceConfigs := fakeDeviceConfig(ctx, []DeviceConfigID{id})

		err := tf.FakeGitiles.SetInventory(config.Get(ctx).Inventory, fakes.InventoryData{
			Lab: inventoryBytesFromDUTs([]testInventoryDut{
				{"dut_id_1", "dut_hostname", "link", "DUT_POOL_SUITES"},
			}),
		})
		store := gitstore.NewInventoryStore(tf.FakeGerrit, tf.FakeGitiles)
		err = store.Refresh(ctx)
		So(err, ShouldBeNil)
		url, err := updateDeviceConfig(tf.C, deviceConfigs, store)
		So(err, ShouldBeNil)
		So(url, ShouldContainSubstring, config.Get(ctx).Inventory.GerritHost)
	})

	Convey("Update DUTs with non-existing device config", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		id := DeviceConfigID{
			PlatformID: "",
			ModelID:    "non-link",
			VariantID:  "",
		}
		deviceConfigs := fakeDeviceConfig(ctx, []DeviceConfigID{id})
		err := tf.FakeGitiles.SetInventory(config.Get(ctx).Inventory, fakes.InventoryData{
			Lab: inventoryBytesFromDUTs([]testInventoryDut{
				{"dut_id_1", "dut_hostname", "link", "DUT_POOL_SUITES"},
			}),
		})
		So(err, ShouldBeNil)
		store := gitstore.NewInventoryStore(tf.FakeGerrit, tf.FakeGitiles)
		err = store.Refresh(ctx)
		So(err, ShouldBeNil)
		url, err := updateDeviceConfig(tf.C, deviceConfigs, store)
		So(err, ShouldBeNil)
		So(url, ShouldNotContainSubstring, config.Get(ctx).Inventory.GerritHost)
	})

	Convey("Update DUTs with exactly same device config", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		id := DeviceConfigID{
			PlatformID: "",
			ModelID:    "link",
			VariantID:  "",
		}
		deviceConfigs := fakeDeviceConfig(ctx, []DeviceConfigID{id})
		err := tf.FakeGitiles.SetInventory(
			config.Get(ctx).Inventory, fakes.InventoryData{
				Lab: []byte(fmt.Sprintf(dut, gpu)),
			},
		)
		So(err, ShouldBeNil)
		store := gitstore.NewInventoryStore(tf.FakeGerrit, tf.FakeGitiles)
		err = store.Refresh(ctx)
		So(err, ShouldBeNil)
		url, err := updateDeviceConfig(tf.C, deviceConfigs, store)
		So(err, ShouldBeNil)
		So(url, ShouldNotContainSubstring, config.Get(ctx).Inventory.GerritHost)
	})
}

func TestBatchUpdateDuts(t *testing.T) {
	Convey("Update DUTs with invalid request", t, func() {
		ctx := testingContext()
		ctx = withSplitInventory(ctx)
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		_, err := tf.Inventory.BatchUpdateDuts(ctx, &fleet.BatchUpdateDutsRequest{
			Hostnames: []string{"fake_host"},
			Pool:      "fake_pool",
			DutProperties: []*fleet.DutProperty{
				{
					Hostname: "jetstream-host",
					Pool:     "DUT_POOL_SUITES",
				},
			},
		})
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "deprecated")
	})

	Convey("Update DUTs with pool", t, func() {
		ctx := testingContext()
		ctx = withSplitInventory(ctx)
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		setSplitGitilesDuts(tf.C, tf.FakeGitiles, []testInventoryDut{
			{id: "dut1_id", hostname: "jetstream-host", model: "link", pool: "DUT_POOL_CQ"},
		})

		Convey("Update DUTs to critical pool", func() {
			dp := &fleet.DutProperty{
				Hostname: "jetstream-host",
				Pool:     "DUT_POOL_SUITES",
			}
			resp, err := tf.Inventory.BatchUpdateDuts(ctx, &fleet.BatchUpdateDutsRequest{
				DutProperties: []*fleet.DutProperty{dp},
			})
			So(err, ShouldBeNil)
			So(resp.GetUrl(), ShouldNotBeNil)
			oneDutLab, err := getLastChangeForHost(tf.FakeGerrit, "data/skylab/chromeos-misc/jetstream-host.textpb")
			So(err, ShouldBeNil)
			dut := oneDutLab.Duts[0]
			common := dut.GetCommon()
			So(common.GetHostname(), ShouldEqual, "jetstream-host")
			So(common.GetLabels().GetCriticalPools(), ShouldResemble, []inventory.SchedulableLabels_DUTPool{inventory.SchedulableLabels_DUT_POOL_SUITES})
		})

		Convey("Update DUTs to non-critical pool", func() {
			dp := &fleet.DutProperty{
				Hostname: "jetstream-host",
				Pool:     "performance",
			}
			resp, err := tf.Inventory.BatchUpdateDuts(ctx, &fleet.BatchUpdateDutsRequest{
				DutProperties: []*fleet.DutProperty{dp},
			})
			So(err, ShouldBeNil)
			So(resp.GetUrl(), ShouldNotBeNil)
			oneDutLab, err := getLastChangeForHost(tf.FakeGerrit, "data/skylab/chromeos-misc/jetstream-host.textpb")
			So(err, ShouldBeNil)
			dut := oneDutLab.Duts[0]
			common := dut.GetCommon()
			So(common.GetHostname(), ShouldEqual, "jetstream-host")
			So(common.GetLabels().GetCriticalPools(), ShouldBeNil)
			So(common.GetLabels().GetSelfServePools(), ShouldResemble, []string{"performance"})
		})

		Convey("Update DUTs with a RPM info", func() {
			dp := &fleet.DutProperty{
				Hostname: "jetstream-host",
				Rpm: &fleet.DutProperty_Rpm{
					PowerunitHostname: "powerunit_host_1",
					PowerunitOutlet:   "powerunit_outlet_A",
				},
			}
			resp, err := tf.Inventory.BatchUpdateDuts(ctx, &fleet.BatchUpdateDutsRequest{
				DutProperties: []*fleet.DutProperty{dp},
			})
			So(err, ShouldBeNil)
			So(resp.GetUrl(), ShouldNotBeNil)
			oneDutLab, err := getLastChangeForHost(tf.FakeGerrit, "data/skylab/chromeos-misc/jetstream-host.textpb")
			So(err, ShouldBeNil)
			dut := oneDutLab.Duts[0]
			common := dut.GetCommon()
			So(common.GetHostname(), ShouldEqual, "jetstream-host")
			ph, exist := getAttributeByKey(common, "powerunit_hostname")
			So(exist, ShouldBeTrue)
			So(ph, ShouldEqual, "powerunit_host_1")
			po, exist := getAttributeByKey(common, "powerunit_outlet")
			So(exist, ShouldBeTrue)
			So(po, ShouldEqual, "powerunit_outlet_A")
		})
	})
}

func TestDumpStableVersionToDatastore(t *testing.T) {
	Convey("Dump Stable version smoke test", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		tf.setStableVersionFactory("{}")
		is := tf.Inventory
		resp, err := is.DumpStableVersionToDatastore(ctx, nil)
		So(err, ShouldBeNil)
		So(resp, ShouldNotBeNil)
	})
	Convey("Update Datastore from empty stableversions file", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		tf.setStableVersionFactory(emptyStableVersions)
		_, err := tf.Inventory.DumpStableVersionToDatastore(ctx, nil)
		So(err, ShouldBeNil)
	})
	Convey("Update Datastore from non-empty stableversions file", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		tf.setStableVersionFactory(stableVersions)
		_, err := tf.Inventory.DumpStableVersionToDatastore(ctx, nil)
		So(err, ShouldBeNil)
		cros, err := dataSV.GetCrosStableVersion(ctx, "auron_paine")
		So(err, ShouldBeNil)
		So(cros, ShouldEqual, "R78-12499.40.0")
		firmware, err := dataSV.GetFirmwareStableVersion(ctx, "auron_paine", "auron_paine")
		So(err, ShouldBeNil)
		So(firmware, ShouldEqual, "Google_Auron_paine.6301.58.98")
		faft, err := dataSV.GetFaftStableVersion(ctx, "auron_paine", "auron_paine")
		So(err, ShouldBeNil)
		So(faft, ShouldEqual, "auron_paine-firmware/R39-6301.58.98")
	})
	Convey("skip entries with empty version strings", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		tf.setStableVersionFactory(stableVersionWithEmptyVersions)
		defer validate()
		resp, err := tf.Inventory.DumpStableVersionToDatastore(ctx, nil)
		So(err, ShouldBeNil)
		So(resp, ShouldNotBeNil)
		_, err = dataSV.GetCrosStableVersion(ctx, "auron_paine")
		So(err, ShouldNotBeNil)
		_, err = dataSV.GetFirmwareStableVersion(ctx, "auron_paine", "auron_paine")
		So(err, ShouldNotBeNil)
		_, err = dataSV.GetFaftStableVersion(ctx, "auron_paine", "auron_paine")
		So(err, ShouldNotBeNil)
	})
}

func TestStableVersionFileParsing(t *testing.T) {
	Convey("Parse non-empty stableversions", t, func() {
		ctx := testingContext()
		parsed, err := parseStableVersions(stableVersions)
		So(err, ShouldBeNil)
		So(parsed, ShouldNotBeNil)
		So(len(parsed.GetCros()), ShouldEqual, 1)
		So(parsed.GetCros()[0].GetVersion(), ShouldEqual, "R78-12499.40.0")
		So(parsed.GetCros()[0].GetKey(), ShouldNotBeNil)
		So(parsed.GetCros()[0].GetKey().GetBuildTarget(), ShouldNotBeNil)
		So(parsed.GetCros()[0].GetKey().GetBuildTarget().GetName(), ShouldEqual, "auron_paine")
		records := getStableVersionRecords(ctx, parsed)
		So(len(records.cros), ShouldEqual, 1)
		So(len(records.firmware), ShouldEqual, 1)
		So(len(records.faft), ShouldEqual, 1)
	})
}
