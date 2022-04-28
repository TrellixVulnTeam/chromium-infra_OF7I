// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/api"
	"go.chromium.org/chromiumos/config/go/payload"
	. "go.chromium.org/luci/common/testing/assertions"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

var localPlatforms = []*crimsonconfig.Platform{
	{Name: "fake platform1"},
	{Name: "fake platform2"},
	{Name: "fake platform3"},
}

func mockParsePlatformsFunc(path string) (*crimsonconfig.Platforms, error) {
	return &crimsonconfig.Platforms{
		Platform: localPlatforms,
	}, nil
}

func mockChromePlatform(id, desc string) *ufspb.ChromePlatform {
	return &ufspb.ChromePlatform{
		Name:        util.AddPrefix(util.ChromePlatformCollection, id),
		Description: desc,
	}
}

func mockMachineLSEPrototype(id string) *ufspb.MachineLSEPrototype {
	return &ufspb.MachineLSEPrototype{
		Name: util.AddPrefix(util.MachineLSEPrototypeCollection, id),
	}
}

func mockRackLSEPrototype(id string) *ufspb.RackLSEPrototype {
	return &ufspb.RackLSEPrototype{
		Name: util.AddPrefix(util.RackLSEPrototypeCollection, id),
	}
}

func mockVlan(id string) *ufspb.Vlan {
	return &ufspb.Vlan{
		Name: util.AddPrefix(util.VlanCollection, id),
	}
}

func mockConfigBundle(id string, programId string, name string) *payload.ConfigBundle {
	return &payload.ConfigBundle{
		DesignList: []*api.Design{
			{
				Id: &api.DesignId{
					Value: id,
				},
				ProgramId: &api.ProgramId{
					Value: programId,
				},
				Name: name,
			},
		},
	}
}

func TestCreateChromePlatform(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromePlatform1 := mockChromePlatform("", "Phone")
	chromePlatform2 := mockChromePlatform("", "Camera")
	chromePlatform3 := mockChromePlatform("", "Sensor")
	Convey("CreateChromePlatform", t, func() {
		Convey("Create new chromePlatform with chromePlatform_id", func() {
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform1,
				ChromePlatformId: "ChromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
		})

		Convey("Create existing chromePlatform", func() {
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform3,
				ChromePlatformId: "ChromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.AlreadyExists)
		})

		Convey("Create new chromePlatform - Invalid input nil", func() {
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform: nil,
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new chromePlatform - Invalid input empty ID", func() {
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform2,
				ChromePlatformId: "",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new chromePlatform - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform2,
				ChromePlatformId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestUpdateChromePlatform(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromePlatform1 := mockChromePlatform("", "Camera")
	chromePlatform2 := mockChromePlatform("chromePlatform-1", "Phone")
	chromePlatform3 := mockChromePlatform("chromePlatform-3", "Sensor")
	chromePlatform4 := mockChromePlatform("a.b)7&", "Printer")
	Convey("UpdateChromePlatform", t, func() {
		Convey("Update existing chromePlatform", func() {
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform1,
				ChromePlatformId: "chromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
			ureq := &ufsAPI.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform2,
			}
			resp, err = tf.Fleet.UpdateChromePlatform(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform2)
		})

		Convey("Update non-existing chromePlatform", func() {
			ureq := &ufsAPI.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform3,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no ChromePlatform with ChromePlatformID chromePlatform-3 in the system")
		})

		Convey("Update chromePlatform - Invalid input nil", func() {
			req := &ufsAPI.UpdateChromePlatformRequest{
				ChromePlatform: nil,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update chromePlatform - Invalid input empty name", func() {
			chromePlatform3.Name = ""
			req := &ufsAPI.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform3,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update chromePlatform - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform4,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestGetChromePlatform(t *testing.T) {
	t.Parallel()
	Convey("GetChromePlatform", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		chromePlatform1 := mockChromePlatform("chromePlatform-1", "Camera")
		req := &ufsAPI.CreateChromePlatformRequest{
			ChromePlatform:   chromePlatform1,
			ChromePlatformId: "chromePlatform-1",
		}
		resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, chromePlatform1)
		Convey("Get chromePlatform by existing ID", func() {
			req := &ufsAPI.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-1"),
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
		})
		Convey("Get chromePlatform by non-existing ID", func() {
			req := &ufsAPI.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get chromePlatform - Invalid input empty name", func() {
			req := &ufsAPI.GetChromePlatformRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get chromePlatform - Invalid input invalid characters", func() {
			req := &ufsAPI.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListChromePlatforms(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromePlatforms := make([]*ufspb.ChromePlatform, 0, 4)
	for i := 0; i < 4; i++ {
		chromePlatform1 := mockChromePlatform("", "Camera")
		chromePlatform1.Name = fmt.Sprintf("chromePlatform-%d", i)
		resp, _ := configuration.CreateChromePlatform(tf.C, chromePlatform1)
		resp.Name = util.AddPrefix(util.ChromePlatformCollection, resp.Name)
		chromePlatforms = append(chromePlatforms, resp)
	}
	Convey("ListChromePlatforms", t, func() {
		Convey("ListChromePlatforms - page_size negative - error", func() {
			req := &ufsAPI.ListChromePlatformsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListChromePlatforms - Full listing with no pagination - happy path", func() {
			req := &ufsAPI.ListChromePlatformsRequest{}
			resp, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.ChromePlatforms, ShouldResembleProto, chromePlatforms)
		})

		Convey("ListChromePlatforms - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListChromePlatformsRequest{
				Filter: "machine=mac-1|kvm=kvm-2",
			}
			_, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})
	})
}

func TestDeleteChromePlatform(t *testing.T) {
	t.Parallel()
	Convey("DeleteChromePlatform", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Delete chromePlatform by existing ID with references", func() {
			chromePlatform1 := mockChromePlatform("", "Camera")
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform1,
				ChromePlatformId: "chromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)

			chromeBrowserMachine1 := &ufspb.Machine{
				Name: util.AddPrefix(util.MachineCollection, "machine-1"),
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-1",
					},
				},
			}
			mresp, merr := registration.CreateMachine(tf.C, chromeBrowserMachine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			/* TODO(eshwarn) : Remove comment when kvm create/get is added
			kvm1 := &ufspb.KVM{
				Name: util.AddPrefix(kvmCollection, "kvm-1"),
				ChromePlatform: "chromePlatform-1",
			}
			kreq := &ufsAPI.CreateKVMMachineRequest{
				Kvm:   kvm1,
				KvmId: "kvm-1",
			}
			kresp, kerr := tf.Fleet.CreateKVM(tf.C, kreq)
			So(kerr, ShouldBeNil)
			So(kresp, ShouldResembleProto, kvm1)
			*/

			dreq := &ufsAPI.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-1"),
			}
			_, err = tf.Fleet.DeleteChromePlatform(tf.C, dreq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.CannotDelete)

			greq := &ufsAPI.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-1"),
			}
			res, err := tf.Fleet.GetChromePlatform(tf.C, greq)
			So(res, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, chromePlatform1)
		})

		Convey("Delete chromePlatform by existing ID without references", func() {
			chromePlatform2 := mockChromePlatform("", "Camera")
			req := &ufsAPI.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform2,
				ChromePlatformId: "chromePlatform-2",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform2)

			dreq := &ufsAPI.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			_, err = tf.Fleet.DeleteChromePlatform(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &ufsAPI.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			res, err := tf.Fleet.GetChromePlatform(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete chromePlatform by non-existing ID", func() {
			req := &ufsAPI.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			_, err := tf.Fleet.DeleteChromePlatform(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete chromePlatform - Invalid input empty name", func() {
			req := &ufsAPI.DeleteChromePlatformRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete chromePlatform - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestImportChromePlatforms(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import chrome platforms", t, func() {
		Convey("happy path", func() {
			req := &ufsAPI.ImportChromePlatformsRequest{
				Source: &ufsAPI.ImportChromePlatformsRequest_ConfigSource{
					ConfigSource: &ufsAPI.ConfigSource{
						ConfigServiceName: "",
						FileName:          "test.config",
					},
				},
			}
			parsePlatformsFunc = mockParsePlatformsFunc
			res, err := tf.Fleet.ImportChromePlatforms(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			getRes, err := configuration.GetAllChromePlatforms(ctx)
			So(err, ShouldBeNil)
			So(getRes, ShouldHaveLength, len(localPlatforms))
			gets := getReturnedPlatformNames(*getRes)
			So(gets, ShouldResemble, []string{"fake_platform1", "fake_platform2", "fake_platform3"})
		})
		Convey("import platforms with invalid argument", func() {
			req := &ufsAPI.ImportChromePlatformsRequest{
				Source: &ufsAPI.ImportChromePlatformsRequest_ConfigSource{},
			}
			_, err := tf.Fleet.ImportChromePlatforms(ctx, req)
			So(err, ShouldNotBeNil)
			s, ok := status.FromError(err)
			So(ok, ShouldBeTrue)
			So(s.Code(), ShouldEqual, code.Code_INVALID_ARGUMENT)
		})
	})
}

func TestImportOSVersions(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import os versions", t, func() {
		Convey("happy path", func() {
			req := &ufsAPI.ImportOSVersionsRequest{
				Source: &ufsAPI.ImportOSVersionsRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportOSVersions(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			resp, _, err := configuration.ListOSes(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			// See ListOSes() in fake/crimson.go
			So(resp, ShouldHaveLength, 2)
			So(ufsAPI.ParseResources(resp, "Value"), ShouldResemble, []string{"os1", "os2"})
			So(ufsAPI.ParseResources(resp, "Description"), ShouldResemble, []string{"os1_description", "os2_description"})
		})
		Convey("import oses with invalid argument", func() {
			req := &ufsAPI.ImportOSVersionsRequest{
				Source: &ufsAPI.ImportOSVersionsRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
						Host: "",
					},
				},
			}
			_, err := tf.Fleet.ImportOSVersions(ctx, req)
			So(err, ShouldNotBeNil)
			s, ok := status.FromError(err)
			So(ok, ShouldBeTrue)
			So(s.Code(), ShouldEqual, code.Code_INVALID_ARGUMENT)
		})
	})
}

func TestCreateMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machineLSEPrototype1 := mockMachineLSEPrototype("")
	machineLSEPrototype2 := mockMachineLSEPrototype("")
	machineLSEPrototype3 := mockMachineLSEPrototype("")
	Convey("CreateMachineLSEPrototype", t, func() {
		Convey("Create new machineLSEPrototype with machineLSEPrototype_id", func() {
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype1,
				MachineLSEPrototypeId: "MachineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
		})

		Convey("Create existing machineLSEPrototype", func() {
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype3,
				MachineLSEPrototypeId: "MachineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.AlreadyExists)
		})

		Convey("Create new machineLSEPrototype - Invalid input nil", func() {
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype: nil,
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new machineLSEPrototype - Invalid input empty ID", func() {
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype2,
				MachineLSEPrototypeId: "",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new machineLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype2,
				MachineLSEPrototypeId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestUpdateMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machineLSEPrototype1 := mockMachineLSEPrototype("")
	machineLSEPrototype2 := mockMachineLSEPrototype("machineLSEPrototype-1")
	machineLSEPrototype3 := mockMachineLSEPrototype("machineLSEPrototype-3")
	machineLSEPrototype4 := mockMachineLSEPrototype("a.b)7&")
	Convey("UpdateMachineLSEPrototype", t, func() {
		Convey("Update existing machineLSEPrototype", func() {
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype1,
				MachineLSEPrototypeId: "machineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
			ureq := &ufsAPI.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype2,
			}
			resp, err = tf.Fleet.UpdateMachineLSEPrototype(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype2)
		})

		Convey("Update non-existing machineLSEPrototype", func() {
			ureq := &ufsAPI.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Update machineLSEPrototype - Invalid input nil", func() {
			req := &ufsAPI.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: nil,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update machineLSEPrototype - Invalid input empty name", func() {
			machineLSEPrototype3.Name = ""
			req := &ufsAPI.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update machineLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype4,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestGetMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	Convey("GetMachineLSEPrototype", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		machineLSEPrototype1 := mockMachineLSEPrototype("machineLSEPrototype-1")
		req := &ufsAPI.CreateMachineLSEPrototypeRequest{
			MachineLSEPrototype:   machineLSEPrototype1,
			MachineLSEPrototypeId: "machineLSEPrototype-1",
		}
		resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, machineLSEPrototype1)
		Convey("Get machineLSEPrototype by existing ID", func() {
			req := &ufsAPI.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-1"),
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
		})
		Convey("Get machineLSEPrototype by non-existing ID", func() {
			req := &ufsAPI.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get machineLSEPrototype - Invalid input empty name", func() {
			req := &ufsAPI.GetMachineLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get machineLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListMachineLSEPrototypes(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machineLSEPrototypes := make([]*ufspb.MachineLSEPrototype, 0, 4)
	for i := 0; i < 4; i++ {
		machineLSEPrototype1 := mockMachineLSEPrototype("")
		machineLSEPrototype1.Name = fmt.Sprintf("machineLSEPrototype-%d", i)
		resp, _ := configuration.CreateMachineLSEPrototype(tf.C, machineLSEPrototype1)
		resp.Name = util.AddPrefix(util.MachineLSEPrototypeCollection, resp.Name)
		machineLSEPrototypes = append(machineLSEPrototypes, resp)
	}
	Convey("ListMachineLSEPrototypes", t, func() {
		Convey("ListMachineLSEPrototypes - page_size negative", func() {
			req := &ufsAPI.ListMachineLSEPrototypesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListMachineLSEPrototypes - Full listing", func() {
			req := &ufsAPI.ListMachineLSEPrototypesRequest{}
			resp, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEPrototypes, ShouldResembleProto, machineLSEPrototypes)
		})

		Convey("ListMachineLSEPrototypes - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListMachineLSEPrototypesRequest{
				Filter: "machine=mac-1|kvm=kvm-2",
			}
			_, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})
	})
}

func TestDeleteMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	Convey("DeleteMachineLSEPrototype", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Delete machineLSEPrototype by existing ID with references", func() {
			machineLSEPrototype1 := mockMachineLSEPrototype("")
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype1,
				MachineLSEPrototypeId: "machineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)

			machineLSE1 := &ufspb.MachineLSE{
				Name:                util.AddPrefix(util.MachineLSECollection, "machinelse-1"),
				MachineLsePrototype: "machineLSEPrototype-1",
				Hostname:            "machinelse-1",
			}
			machineLSE1, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)

			dreq := &ufsAPI.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-1"),
			}
			_, err = tf.Fleet.DeleteMachineLSEPrototype(tf.C, dreq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.CannotDelete)

			greq := &ufsAPI.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-1"),
			}
			res, err := tf.Fleet.GetMachineLSEPrototype(tf.C, greq)
			So(res, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, machineLSEPrototype1)
		})

		Convey("Delete machineLSEPrototype by existing ID without references", func() {
			machineLSEPrototype2 := mockMachineLSEPrototype("")
			req := &ufsAPI.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype2,
				MachineLSEPrototypeId: "machineLSEPrototype-2",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype2)

			dreq := &ufsAPI.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			_, err = tf.Fleet.DeleteMachineLSEPrototype(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &ufsAPI.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			res, err := tf.Fleet.GetMachineLSEPrototype(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete machineLSEPrototype by non-existing ID", func() {
			req := &ufsAPI.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			_, err := tf.Fleet.DeleteMachineLSEPrototype(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete machineLSEPrototype - Invalid input empty name", func() {
			req := &ufsAPI.DeleteMachineLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete machineLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestCreateRackLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rackLSEPrototype1 := mockRackLSEPrototype("")
	rackLSEPrototype2 := mockRackLSEPrototype("")
	rackLSEPrototype3 := mockRackLSEPrototype("")
	Convey("CreateRackLSEPrototype", t, func() {
		Convey("Create new rackLSEPrototype with rackLSEPrototype_id", func() {
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype1,
				RackLSEPrototypeId: "RackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
		})

		Convey("Create existing rackLSEPrototype", func() {
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype3,
				RackLSEPrototypeId: "RackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.AlreadyExists)
		})

		Convey("Create new rackLSEPrototype - Invalid input nil", func() {
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype: nil,
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new rackLSEPrototype - Invalid input empty ID", func() {
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype2,
				RackLSEPrototypeId: "",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new rackLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype2,
				RackLSEPrototypeId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestUpdateRackLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rackLSEPrototype1 := mockRackLSEPrototype("")
	rackLSEPrototype2 := mockRackLSEPrototype("rackLSEPrototype-1")
	rackLSEPrototype3 := mockRackLSEPrototype("rackLSEPrototype-3")
	rackLSEPrototype4 := mockRackLSEPrototype("a.b)7&")
	Convey("UpdateRackLSEPrototype", t, func() {
		Convey("Update existing rackLSEPrototype", func() {
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype1,
				RackLSEPrototypeId: "rackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
			ureq := &ufsAPI.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype2,
			}
			resp, err = tf.Fleet.UpdateRackLSEPrototype(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype2)
		})

		Convey("Update non-existing rackLSEPrototype", func() {
			ureq := &ufsAPI.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Update rackLSEPrototype - Invalid input nil", func() {
			req := &ufsAPI.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: nil,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update rackLSEPrototype - Invalid input empty name", func() {
			rackLSEPrototype3.Name = ""
			req := &ufsAPI.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update rackLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype4,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestGetRackLSEPrototype(t *testing.T) {
	t.Parallel()
	Convey("GetRackLSEPrototype", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		rackLSEPrototype1 := mockRackLSEPrototype("rackLSEPrototype-1")
		req := &ufsAPI.CreateRackLSEPrototypeRequest{
			RackLSEPrototype:   rackLSEPrototype1,
			RackLSEPrototypeId: "rackLSEPrototype-1",
		}
		resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, rackLSEPrototype1)
		Convey("Get rackLSEPrototype by existing ID", func() {
			req := &ufsAPI.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-1"),
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
		})
		Convey("Get rackLSEPrototype by non-existing ID", func() {
			req := &ufsAPI.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get rackLSEPrototype - Invalid input empty name", func() {
			req := &ufsAPI.GetRackLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get rackLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListRackLSEPrototypes(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rackLSEPrototypes := make([]*ufspb.RackLSEPrototype, 0, 4)
	for i := 0; i < 4; i++ {
		rackLSEPrototype1 := mockRackLSEPrototype("")
		rackLSEPrototype1.Name = fmt.Sprintf("rackLSEPrototype-%d", i)
		resp, _ := configuration.CreateRackLSEPrototype(tf.C, rackLSEPrototype1)
		resp.Name = util.AddPrefix(util.RackLSEPrototypeCollection, resp.Name)
		rackLSEPrototypes = append(rackLSEPrototypes, resp)
	}
	Convey("ListRackLSEPrototypes", t, func() {
		Convey("ListRackLSEPrototypes - page_size negative - error", func() {
			req := &ufsAPI.ListRackLSEPrototypesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListRackLSEPrototypes - Full listing - happy path", func() {
			req := &ufsAPI.ListRackLSEPrototypesRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RackLSEPrototypes, ShouldResembleProto, rackLSEPrototypes)
		})

		Convey("ListRackLSEPrototypes - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListRackLSEPrototypesRequest{
				Filter: "machine=mac-1|kvm=kvm-2",
			}
			_, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})
	})
}

func TestDeleteRackLSEPrototype(t *testing.T) {
	t.Parallel()
	Convey("DeleteRackLSEPrototype", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Delete rackLSEPrototype by existing ID with references", func() {
			rackLSEPrototype1 := mockRackLSEPrototype("")
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype1,
				RackLSEPrototypeId: "rackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)

			rackLSE1 := &ufspb.RackLSE{
				Name:             util.AddPrefix(util.RackLSECollection, "racklse-1"),
				RackLsePrototype: "rackLSEPrototype-1",
			}
			mreq := &ufsAPI.CreateRackLSERequest{
				RackLSE:   rackLSE1,
				RackLSEId: "racklse-1",
			}
			mresp, merr := tf.Fleet.CreateRackLSE(tf.C, mreq)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, rackLSE1)

			dreq := &ufsAPI.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-1"),
			}
			_, err = tf.Fleet.DeleteRackLSEPrototype(tf.C, dreq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.CannotDelete)

			greq := &ufsAPI.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-1"),
			}
			res, err := tf.Fleet.GetRackLSEPrototype(tf.C, greq)
			So(res, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, rackLSEPrototype1)
		})

		Convey("Delete rackLSEPrototype by existing ID without references", func() {
			rackLSEPrototype2 := mockRackLSEPrototype("")
			req := &ufsAPI.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype2,
				RackLSEPrototypeId: "rackLSEPrototype-2",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype2)

			dreq := &ufsAPI.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			_, err = tf.Fleet.DeleteRackLSEPrototype(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &ufsAPI.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			res, err := tf.Fleet.GetRackLSEPrototype(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete rackLSEPrototype by non-existing ID", func() {
			req := &ufsAPI.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			_, err := tf.Fleet.DeleteRackLSEPrototype(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete rackLSEPrototype - Invalid input empty name", func() {
			req := &ufsAPI.DeleteRackLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete rackLSEPrototype - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestCreateVlan(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	vlan1 := mockVlan("")
	vlan2 := mockVlan("")
	vlan3 := mockVlan("")
	Convey("CreateVlan", t, func() {
		Convey("Create new vlan with vlan_id", func() {
			vlan1.VlanAddress = "192.168.255.248/27"
			req := &ufsAPI.CreateVlanRequest{
				Vlan:   vlan1,
				VlanId: "Vlan-1",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})

		Convey("Create existing vlan", func() {
			vlan3.VlanAddress = "192.168.255.248/27"
			req := &ufsAPI.CreateVlanRequest{
				Vlan:   vlan3,
				VlanId: "Vlan-1",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already exists")
		})

		Convey("Create new vlan - Invalid input nil", func() {
			req := &ufsAPI.CreateVlanRequest{
				Vlan: nil,
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new vlan - Invalid input empty ID", func() {
			req := &ufsAPI.CreateVlanRequest{
				Vlan:   vlan2,
				VlanId: "",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new vlan - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateVlanRequest{
				Vlan:   vlan2,
				VlanId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestUpdateVlan(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	vlan1 := mockVlan("")
	vlan2 := mockVlan("vlan-1")
	vlan3 := mockVlan("vlan-3")
	vlan4 := mockVlan("a.b)7&")
	Convey("UpdateVlan", t, func() {
		Convey("Update existing vlan", func() {
			vlan1.VlanAddress = "3.3.3.3/27"
			req := &ufsAPI.CreateVlanRequest{
				Vlan:   vlan1,
				VlanId: "vlan-1",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
			ureq := &ufsAPI.UpdateVlanRequest{
				Vlan: vlan2,
			}
			resp, err = tf.Fleet.UpdateVlan(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan2)
		})

		Convey("Update non-existing vlan", func() {
			ureq := &ufsAPI.UpdateVlanRequest{
				Vlan: vlan3,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Update vlan - Invalid input nil", func() {
			req := &ufsAPI.UpdateVlanRequest{
				Vlan: nil,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update vlan - Invalid input empty name", func() {
			vlan3.Name = ""
			req := &ufsAPI.UpdateVlanRequest{
				Vlan: vlan3,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update vlan - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateVlanRequest{
				Vlan: vlan4,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestGetVlan(t *testing.T) {
	t.Parallel()
	Convey("GetVlan", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		vlan1 := mockVlan("vlan-1")
		vlan1.VlanAddress = "3.3.3.4/27"
		req := &ufsAPI.CreateVlanRequest{
			Vlan:   vlan1,
			VlanId: "vlan-1",
		}
		resp, err := tf.Fleet.CreateVlan(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, vlan1)
		Convey("Get vlan by existing ID", func() {
			req := &ufsAPI.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-1"),
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})
		Convey("Get vlan by non-existing ID", func() {
			req := &ufsAPI.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get vlan - Invalid input empty name", func() {
			req := &ufsAPI.GetVlanRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get vlan - Invalid input invalid characters", func() {
			req := &ufsAPI.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListVlans(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	vlans := make([]*ufspb.Vlan, 0, 4)
	for i := 0; i < 4; i++ {
		vlan1 := mockVlan("")
		vlan1.Name = fmt.Sprintf("vlan-%d", i)
		resp, _ := configuration.CreateVlan(tf.C, vlan1)
		resp.Name = util.AddPrefix(util.VlanCollection, resp.Name)
		vlans = append(vlans, resp)
	}
	Convey("ListVlans", t, func() {
		Convey("ListVlans - page_size negative - error", func() {
			req := &ufsAPI.ListVlansRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListVlans - Full listing - happy path", func() {
			req := &ufsAPI.ListVlansRequest{}
			resp, err := tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Vlans, ShouldResembleProto, vlans)
		})

		Convey("ListVlans - page_size negative - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListVlansRequest{
				Filter: "machine=mac-1|kvm=kvm-2",
			}
			_, err := tf.Fleet.ListVlans(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})
	})
}

func TestDeleteVlan(t *testing.T) {
	t.Parallel()
	Convey("DeleteVlan", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Delete vlan by existing ID without references", func() {
			vlan2 := mockVlan("")
			vlan2.VlanAddress = "192.168.110.0/27"
			req := &ufsAPI.CreateVlanRequest{
				Vlan:   vlan2,
				VlanId: "vlan-2",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan2)

			dreq := &ufsAPI.DeleteVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			_, err = tf.Fleet.DeleteVlan(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &ufsAPI.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			res, err := tf.Fleet.GetVlan(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete vlan by non-existing ID", func() {
			req := &ufsAPI.DeleteVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			_, err := tf.Fleet.DeleteVlan(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete vlan - Invalid input empty name", func() {
			req := &ufsAPI.DeleteVlanRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete vlan - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestImportVlans(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import vlans", t, func() {
		Convey("happy path", func() {
			req := &ufsAPI.ImportVlansRequest{
				Source: &ufsAPI.ImportVlansRequest_ConfigSource{
					ConfigSource: &ufsAPI.ConfigSource{
						ConfigServiceName: "fake-service",
						FileName:          "fakeVlans.cfg",
					},
				},
			}
			tf.Fleet.importPageSize = 25
			res, err := tf.Fleet.ImportVlans(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			vlans, _, err := configuration.ListVlans(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(vlans, "Name"), ShouldResemble, []string{"browser:144", "browser:20", "browser:40"})
			vlan, err := configuration.GetVlan(ctx, "browser:40")
			So(err, ShouldBeNil)
			So(vlan.GetCapacityIp(), ShouldEqual, 1024)
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"vlan": "browser:40"})
			So(err, ShouldBeNil)
			So(len(ips), ShouldEqual, 1024)
		})
	})
}

func TestOSImportVlans(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import OS vlan-related infos", t, func() {
		Convey("happy path", func() {
			req := &ufsAPI.ImportOSVlansRequest{
				Source: &ufsAPI.ImportOSVlansRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			tf.Fleet.importPageSize = 25
			res, err := tf.Fleet.ImportOSVlans(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
		})
	})
}

func getReturnedPlatformNames(res datastore.OpResults) []string {
	gets := make([]string, len(res))
	for i, r := range res {
		gets[i] = r.Data.(*ufspb.ChromePlatform).GetName()
	}
	return gets
}

func TestUpdateConfigBundle(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	t.Run("update non-existent ConfigBundle", func(t *testing.T) {
		cb1 := mockConfigBundle("design1", "program1", "name1")
		cb1Bytes, err := proto.Marshal(cb1)
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}
		want := &ufsAPI.UpdateConfigBundleResponse{
			ConfigBundle: cb1Bytes,
		}

		got, err := tf.Fleet.UpdateConfigBundle(tf.C, &ufsAPI.UpdateConfigBundleRequest{
			ConfigBundle: cb1Bytes,
			UpdateMask:   nil,
			AllowMissing: true,
		})
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("UpdateConfigBundle returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update existent ConfigBundle", func(t *testing.T) {
		cb2 := mockConfigBundle("design2", "program2", "name2")
		cb2Bytes, err := proto.Marshal(cb2)
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}

		_, _ = tf.Fleet.UpdateConfigBundle(tf.C, &ufsAPI.UpdateConfigBundleRequest{
			ConfigBundle: cb2Bytes,
			UpdateMask:   nil,
			AllowMissing: true,
		})

		// Update cb2
		cb2update := mockConfigBundle("design2", "program2", "name2update")
		cb2updateBytes, err := proto.Marshal(cb2update)
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}
		want := &ufsAPI.UpdateConfigBundleResponse{
			ConfigBundle: cb2updateBytes,
		}

		got, err := tf.Fleet.UpdateConfigBundle(tf.C, &ufsAPI.UpdateConfigBundleRequest{
			ConfigBundle: cb2updateBytes,
			UpdateMask:   nil,
			AllowMissing: true,
		})
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("UpdateConfigBundle returned unexpected diff (-want +got):\n%s", diff)
		}
	})

	t.Run("update existent ConfigBundle", func(t *testing.T) {
		cb3 := mockConfigBundle("", "", "")
		cb3Bytes, err := proto.Marshal(cb3)
		if err != nil {
			t.Fatalf("UpdateConfigBundle failed: %s", err)
		}

		got, err := tf.Fleet.UpdateConfigBundle(tf.C, &ufsAPI.UpdateConfigBundleRequest{
			ConfigBundle: cb3Bytes,
			UpdateMask:   nil,
			AllowMissing: true,
		})
		if err == nil {
			t.Errorf("UpdateConfigBundle succeeded with empty IDs")
		}
		if c := status.Code(err); c != codes.InvalidArgument {
			t.Errorf("Unexpected error when calling GetConfigBundle: %s", err)
		}

		var respNil *ufsAPI.UpdateConfigBundleResponse = nil
		if diff := cmp.Diff(respNil, got); diff != "" {
			t.Errorf("UpdateConfigBundle returned unexpected diff (-want +got):\n%s", diff)
		}
	})
}
