// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/status"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"

	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
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

func mockChromePlatform(id, desc string) *proto.ChromePlatform {
	return &proto.ChromePlatform{
		Name:        util.AddPrefix(util.ChromePlatformCollection, id),
		Description: desc,
	}
}

func mockMachineLSEPrototype(id string) *proto.MachineLSEPrototype {
	return &proto.MachineLSEPrototype{
		Name: util.AddPrefix(util.MachineLSEPrototypeCollection, id),
	}
}

func mockRackLSEPrototype(id string) *proto.RackLSEPrototype {
	return &proto.RackLSEPrototype{
		Name: util.AddPrefix(util.RackLSEPrototypeCollection, id),
	}
}

func mockVlan(id string) *proto.Vlan {
	return &proto.Vlan{
		Name: util.AddPrefix(util.VlanCollection, id),
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
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform1,
				ChromePlatformId: "ChromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
		})

		Convey("Create existing chromePlatform", func() {
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform3,
				ChromePlatformId: "ChromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.AlreadyExists)
		})

		Convey("Create new chromePlatform - Invalid input nil", func() {
			req := &api.CreateChromePlatformRequest{
				ChromePlatform: nil,
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new chromePlatform - Invalid input empty ID", func() {
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform2,
				ChromePlatformId: "",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new chromePlatform - Invalid input invalid characters", func() {
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform2,
				ChromePlatformId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform1,
				ChromePlatformId: "chromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
			ureq := &api.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform2,
			}
			resp, err = tf.Fleet.UpdateChromePlatform(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform2)
		})

		Convey("Update non-existing chromePlatform", func() {
			ureq := &api.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform3,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Update chromePlatform - Invalid input nil", func() {
			req := &api.UpdateChromePlatformRequest{
				ChromePlatform: nil,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update chromePlatform - Invalid input empty name", func() {
			chromePlatform3.Name = ""
			req := &api.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform3,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update chromePlatform - Invalid input invalid characters", func() {
			req := &api.UpdateChromePlatformRequest{
				ChromePlatform: chromePlatform4,
			}
			resp, err := tf.Fleet.UpdateChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
		req := &api.CreateChromePlatformRequest{
			ChromePlatform:   chromePlatform1,
			ChromePlatformId: "chromePlatform-1",
		}
		resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, chromePlatform1)
		Convey("Get chromePlatform by existing ID", func() {
			req := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-1"),
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
		})
		Convey("Get chromePlatform by non-existing ID", func() {
			req := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get chromePlatform - Invalid input empty name", func() {
			req := &api.GetChromePlatformRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get chromePlatform - Invalid input invalid characters", func() {
			req := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListChromePlatforms(t *testing.T) {
	t.Parallel()
	Convey("ListChromePlatforms", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		chromePlatforms := make([]*proto.ChromePlatform, 0, 4)
		for i := 0; i < 4; i++ {
			chromePlatform1 := mockChromePlatform("", "Camera")
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform1,
				ChromePlatformId: fmt.Sprintf("chromePlatform-%d", i),
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
			chromePlatforms = append(chromePlatforms, resp)
		}

		Convey("ListChromePlatforms - page_size negative", func() {
			req := &api.ListChromePlatformsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListChromePlatforms - page_token invalid", func() {
			req := &api.ListChromePlatformsRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.InvalidPageToken)
		})

		Convey("ListChromePlatforms - Full listing Max PageSize", func() {
			req := &api.ListChromePlatformsRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.ChromePlatforms, ShouldResembleProto, chromePlatforms)
		})

		Convey("ListChromePlatforms - Full listing with no pagination", func() {
			req := &api.ListChromePlatformsRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.ChromePlatforms, ShouldResembleProto, chromePlatforms)
		})

		Convey("ListChromePlatforms - listing with pagination", func() {
			req := &api.ListChromePlatformsRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.ChromePlatforms, ShouldResembleProto, chromePlatforms[:3])

			req = &api.ListChromePlatformsRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListChromePlatforms(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.ChromePlatforms, ShouldResembleProto, chromePlatforms[3:])
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
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform1,
				ChromePlatformId: "chromePlatform-1",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)

			chromeBrowserMachine1 := &proto.Machine{
				Name: util.AddPrefix(util.MachineCollection, "machine-1"),
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-1",
					},
				},
			}
			mreq := &api.CreateMachineRequest{
				Machine:   chromeBrowserMachine1,
				MachineId: "machine-1",
			}
			mresp, merr := tf.Fleet.CreateMachine(tf.C, mreq)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			/* TODO(eshwarn) : Remove comment when kvm create/get is added
			kvm1 := &proto.KVM{
				Name: util.AddPrefix(kvmCollection, "kvm-1"),
				ChromePlatform: "chromePlatform-1",
			}
			kreq := &api.CreateKVMMachineRequest{
				Kvm:   kvm1,
				KvmId: "kvm-1",
			}
			kresp, kerr := tf.Fleet.CreateKVM(tf.C, kreq)
			So(kerr, ShouldBeNil)
			So(kresp, ShouldResembleProto, kvm1)
			*/

			dreq := &api.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-1"),
			}
			_, err = tf.Fleet.DeleteChromePlatform(tf.C, dreq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.CannotDelete)

			greq := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-1"),
			}
			res, err := tf.Fleet.GetChromePlatform(tf.C, greq)
			So(res, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, chromePlatform1)
		})

		Convey("Delete chromePlatform by existing ID without references", func() {
			chromePlatform2 := mockChromePlatform("", "Camera")
			req := &api.CreateChromePlatformRequest{
				ChromePlatform:   chromePlatform2,
				ChromePlatformId: "chromePlatform-2",
			}
			resp, err := tf.Fleet.CreateChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform2)

			dreq := &api.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			_, err = tf.Fleet.DeleteChromePlatform(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			res, err := tf.Fleet.GetChromePlatform(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete chromePlatform by non-existing ID", func() {
			req := &api.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "chromePlatform-2"),
			}
			_, err := tf.Fleet.DeleteChromePlatform(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete chromePlatform - Invalid input empty name", func() {
			req := &api.DeleteChromePlatformRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete chromePlatform - Invalid input invalid characters", func() {
			req := &api.DeleteChromePlatformRequest{
				Name: util.AddPrefix(util.ChromePlatformCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteChromePlatform(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.ImportChromePlatformsRequest{
				Source: &api.ImportChromePlatformsRequest_ConfigSource{
					ConfigSource: &api.ConfigSource{
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
			req := &api.ImportChromePlatformsRequest{
				Source: &api.ImportChromePlatformsRequest_ConfigSource{},
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
			req := &api.ImportOSVersionsRequest{
				Source: &api.ImportOSVersionsRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportOSVersions(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			resp, _, err := configuration.ListOSes(ctx, 100, "")
			So(err, ShouldBeNil)
			// See ListOSes() in fake/crimson.go
			So(resp, ShouldHaveLength, 2)
			So(api.ParseResources(resp, "Value"), ShouldResemble, []string{"os1", "os2"})
			So(api.ParseResources(resp, "Description"), ShouldResemble, []string{"os1_description", "os2_description"})
		})
		Convey("import oses with invalid argument", func() {
			req := &api.ImportOSVersionsRequest{
				Source: &api.ImportOSVersionsRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
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
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype1,
				MachineLSEPrototypeId: "MachineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
		})

		Convey("Create existing machineLSEPrototype", func() {
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype3,
				MachineLSEPrototypeId: "MachineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.AlreadyExists)
		})

		Convey("Create new machineLSEPrototype - Invalid input nil", func() {
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype: nil,
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new machineLSEPrototype - Invalid input empty ID", func() {
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype2,
				MachineLSEPrototypeId: "",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new machineLSEPrototype - Invalid input invalid characters", func() {
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype2,
				MachineLSEPrototypeId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype1,
				MachineLSEPrototypeId: "machineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
			ureq := &api.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype2,
			}
			resp, err = tf.Fleet.UpdateMachineLSEPrototype(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype2)
		})

		Convey("Update non-existing machineLSEPrototype", func() {
			ureq := &api.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Update machineLSEPrototype - Invalid input nil", func() {
			req := &api.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: nil,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update machineLSEPrototype - Invalid input empty name", func() {
			machineLSEPrototype3.Name = ""
			req := &api.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update machineLSEPrototype - Invalid input invalid characters", func() {
			req := &api.UpdateMachineLSEPrototypeRequest{
				MachineLSEPrototype: machineLSEPrototype4,
			}
			resp, err := tf.Fleet.UpdateMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
		req := &api.CreateMachineLSEPrototypeRequest{
			MachineLSEPrototype:   machineLSEPrototype1,
			MachineLSEPrototypeId: "machineLSEPrototype-1",
		}
		resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, machineLSEPrototype1)
		Convey("Get machineLSEPrototype by existing ID", func() {
			req := &api.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-1"),
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
		})
		Convey("Get machineLSEPrototype by non-existing ID", func() {
			req := &api.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get machineLSEPrototype - Invalid input empty name", func() {
			req := &api.GetMachineLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get machineLSEPrototype - Invalid input invalid characters", func() {
			req := &api.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListMachineLSEPrototypes(t *testing.T) {
	t.Parallel()
	Convey("ListMachineLSEPrototypes", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		machineLSEPrototypes := make([]*proto.MachineLSEPrototype, 0, 4)
		for i := 0; i < 4; i++ {
			machineLSEPrototype1 := mockMachineLSEPrototype("")
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype1,
				MachineLSEPrototypeId: fmt.Sprintf("machineLSEPrototype-%d", i),
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
			machineLSEPrototypes = append(machineLSEPrototypes, resp)
		}

		Convey("ListMachineLSEPrototypes - page_size negative", func() {
			req := &api.ListMachineLSEPrototypesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListMachineLSEPrototypes - page_token invalid", func() {
			req := &api.ListMachineLSEPrototypesRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.InvalidPageToken)
		})

		Convey("ListMachineLSEPrototypes - Full listing Max PageSize", func() {
			req := &api.ListMachineLSEPrototypesRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEPrototypes, ShouldResembleProto, machineLSEPrototypes)
		})

		Convey("ListMachineLSEPrototypes - Full listing with no pagination", func() {
			req := &api.ListMachineLSEPrototypesRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEPrototypes, ShouldResembleProto, machineLSEPrototypes)
		})

		Convey("ListMachineLSEPrototypes - listing with pagination", func() {
			req := &api.ListMachineLSEPrototypesRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEPrototypes, ShouldResembleProto, machineLSEPrototypes[:3])

			req = &api.ListMachineLSEPrototypesRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListMachineLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEPrototypes, ShouldResembleProto, machineLSEPrototypes[3:])
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
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype1,
				MachineLSEPrototypeId: "machineLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)

			machineLSE1 := &proto.MachineLSE{
				Name:                util.AddPrefix(util.MachineLSECollection, "machinelse-1"),
				MachineLsePrototype: "machineLSEPrototype-1",
				Hostname:            "machinelse-1",
			}
			machineLSE1, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)

			dreq := &api.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-1"),
			}
			_, err = tf.Fleet.DeleteMachineLSEPrototype(tf.C, dreq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.CannotDelete)

			greq := &api.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-1"),
			}
			res, err := tf.Fleet.GetMachineLSEPrototype(tf.C, greq)
			So(res, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, machineLSEPrototype1)
		})

		Convey("Delete machineLSEPrototype by existing ID without references", func() {
			machineLSEPrototype2 := mockMachineLSEPrototype("")
			req := &api.CreateMachineLSEPrototypeRequest{
				MachineLSEPrototype:   machineLSEPrototype2,
				MachineLSEPrototypeId: "machineLSEPrototype-2",
			}
			resp, err := tf.Fleet.CreateMachineLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype2)

			dreq := &api.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			_, err = tf.Fleet.DeleteMachineLSEPrototype(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &api.GetMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			res, err := tf.Fleet.GetMachineLSEPrototype(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete machineLSEPrototype by non-existing ID", func() {
			req := &api.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "machineLSEPrototype-2"),
			}
			_, err := tf.Fleet.DeleteMachineLSEPrototype(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete machineLSEPrototype - Invalid input empty name", func() {
			req := &api.DeleteMachineLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete machineLSEPrototype - Invalid input invalid characters", func() {
			req := &api.DeleteMachineLSEPrototypeRequest{
				Name: util.AddPrefix(util.MachineLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteMachineLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype1,
				RackLSEPrototypeId: "RackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
		})

		Convey("Create existing rackLSEPrototype", func() {
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype3,
				RackLSEPrototypeId: "RackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.AlreadyExists)
		})

		Convey("Create new rackLSEPrototype - Invalid input nil", func() {
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype: nil,
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new rackLSEPrototype - Invalid input empty ID", func() {
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype2,
				RackLSEPrototypeId: "",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new rackLSEPrototype - Invalid input invalid characters", func() {
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype2,
				RackLSEPrototypeId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype1,
				RackLSEPrototypeId: "rackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
			ureq := &api.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype2,
			}
			resp, err = tf.Fleet.UpdateRackLSEPrototype(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype2)
		})

		Convey("Update non-existing rackLSEPrototype", func() {
			ureq := &api.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Update rackLSEPrototype - Invalid input nil", func() {
			req := &api.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: nil,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update rackLSEPrototype - Invalid input empty name", func() {
			rackLSEPrototype3.Name = ""
			req := &api.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype3,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update rackLSEPrototype - Invalid input invalid characters", func() {
			req := &api.UpdateRackLSEPrototypeRequest{
				RackLSEPrototype: rackLSEPrototype4,
			}
			resp, err := tf.Fleet.UpdateRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
		req := &api.CreateRackLSEPrototypeRequest{
			RackLSEPrototype:   rackLSEPrototype1,
			RackLSEPrototypeId: "rackLSEPrototype-1",
		}
		resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, rackLSEPrototype1)
		Convey("Get rackLSEPrototype by existing ID", func() {
			req := &api.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-1"),
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
		})
		Convey("Get rackLSEPrototype by non-existing ID", func() {
			req := &api.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get rackLSEPrototype - Invalid input empty name", func() {
			req := &api.GetRackLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get rackLSEPrototype - Invalid input invalid characters", func() {
			req := &api.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListRackLSEPrototypes(t *testing.T) {
	t.Parallel()
	Convey("ListRackLSEPrototypes", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		rackLSEPrototypes := make([]*proto.RackLSEPrototype, 0, 4)
		for i := 0; i < 4; i++ {
			rackLSEPrototype1 := mockRackLSEPrototype("")
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype1,
				RackLSEPrototypeId: fmt.Sprintf("rackLSEPrototype-%d", i),
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
			rackLSEPrototypes = append(rackLSEPrototypes, resp)
		}

		Convey("ListRackLSEPrototypes - page_size negative", func() {
			req := &api.ListRackLSEPrototypesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListRackLSEPrototypes - page_token invalid", func() {
			req := &api.ListRackLSEPrototypesRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.InvalidPageToken)
		})

		Convey("ListRackLSEPrototypes - Full listing Max PageSize", func() {
			req := &api.ListRackLSEPrototypesRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RackLSEPrototypes, ShouldResembleProto, rackLSEPrototypes)
		})

		Convey("ListRackLSEPrototypes - Full listing with no pagination", func() {
			req := &api.ListRackLSEPrototypesRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RackLSEPrototypes, ShouldResembleProto, rackLSEPrototypes)
		})

		Convey("ListRackLSEPrototypes - listing with pagination", func() {
			req := &api.ListRackLSEPrototypesRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RackLSEPrototypes, ShouldResembleProto, rackLSEPrototypes[:3])

			req = &api.ListRackLSEPrototypesRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListRackLSEPrototypes(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RackLSEPrototypes, ShouldResembleProto, rackLSEPrototypes[3:])
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
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype1,
				RackLSEPrototypeId: "rackLSEPrototype-1",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)

			rackLSE1 := &proto.RackLSE{
				Name:             util.AddPrefix(util.RackLSECollection, "racklse-1"),
				RackLsePrototype: "rackLSEPrototype-1",
			}
			mreq := &api.CreateRackLSERequest{
				RackLSE:   rackLSE1,
				RackLSEId: "racklse-1",
			}
			mresp, merr := tf.Fleet.CreateRackLSE(tf.C, mreq)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, rackLSE1)

			dreq := &api.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-1"),
			}
			_, err = tf.Fleet.DeleteRackLSEPrototype(tf.C, dreq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.CannotDelete)

			greq := &api.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-1"),
			}
			res, err := tf.Fleet.GetRackLSEPrototype(tf.C, greq)
			So(res, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, rackLSEPrototype1)
		})

		Convey("Delete rackLSEPrototype by existing ID without references", func() {
			rackLSEPrototype2 := mockRackLSEPrototype("")
			req := &api.CreateRackLSEPrototypeRequest{
				RackLSEPrototype:   rackLSEPrototype2,
				RackLSEPrototypeId: "rackLSEPrototype-2",
			}
			resp, err := tf.Fleet.CreateRackLSEPrototype(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype2)

			dreq := &api.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			_, err = tf.Fleet.DeleteRackLSEPrototype(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &api.GetRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			res, err := tf.Fleet.GetRackLSEPrototype(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete rackLSEPrototype by non-existing ID", func() {
			req := &api.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "rackLSEPrototype-2"),
			}
			_, err := tf.Fleet.DeleteRackLSEPrototype(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete rackLSEPrototype - Invalid input empty name", func() {
			req := &api.DeleteRackLSEPrototypeRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete rackLSEPrototype - Invalid input invalid characters", func() {
			req := &api.DeleteRackLSEPrototypeRequest{
				Name: util.AddPrefix(util.RackLSEPrototypeCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRackLSEPrototype(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.CreateVlanRequest{
				Vlan:   vlan1,
				VlanId: "Vlan-1",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})

		Convey("Create existing vlan", func() {
			req := &api.CreateVlanRequest{
				Vlan:   vlan3,
				VlanId: "Vlan-1",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.AlreadyExists)
		})

		Convey("Create new vlan - Invalid input nil", func() {
			req := &api.CreateVlanRequest{
				Vlan: nil,
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new vlan - Invalid input empty ID", func() {
			req := &api.CreateVlanRequest{
				Vlan:   vlan2,
				VlanId: "",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new vlan - Invalid input invalid characters", func() {
			req := &api.CreateVlanRequest{
				Vlan:   vlan2,
				VlanId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.CreateVlanRequest{
				Vlan:   vlan1,
				VlanId: "vlan-1",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
			ureq := &api.UpdateVlanRequest{
				Vlan: vlan2,
			}
			resp, err = tf.Fleet.UpdateVlan(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan2)
		})

		Convey("Update non-existing vlan", func() {
			ureq := &api.UpdateVlanRequest{
				Vlan: vlan3,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Update vlan - Invalid input nil", func() {
			req := &api.UpdateVlanRequest{
				Vlan: nil,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update vlan - Invalid input empty name", func() {
			vlan3.Name = ""
			req := &api.UpdateVlanRequest{
				Vlan: vlan3,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update vlan - Invalid input invalid characters", func() {
			req := &api.UpdateVlanRequest{
				Vlan: vlan4,
			}
			resp, err := tf.Fleet.UpdateVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
		req := &api.CreateVlanRequest{
			Vlan:   vlan1,
			VlanId: "vlan-1",
		}
		resp, err := tf.Fleet.CreateVlan(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, vlan1)
		Convey("Get vlan by existing ID", func() {
			req := &api.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-1"),
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})
		Convey("Get vlan by non-existing ID", func() {
			req := &api.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("Get vlan - Invalid input empty name", func() {
			req := &api.GetVlanRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get vlan - Invalid input invalid characters", func() {
			req := &api.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListVlans(t *testing.T) {
	t.Parallel()
	Convey("ListVlans", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		vlans := make([]*proto.Vlan, 0, 4)
		for i := 0; i < 4; i++ {
			vlan1 := mockVlan("")
			req := &api.CreateVlanRequest{
				Vlan:   vlan1,
				VlanId: fmt.Sprintf("vlan-%d", i),
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
			vlans = append(vlans, resp)
		}

		Convey("ListVlans - page_size negative", func() {
			req := &api.ListVlansRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListVlans - page_token invalid", func() {
			req := &api.ListVlansRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.InvalidPageToken)
		})

		Convey("ListVlans - Full listing Max PageSize", func() {
			req := &api.ListVlansRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Vlans, ShouldResembleProto, vlans)
		})

		Convey("ListVlans - Full listing with no pagination", func() {
			req := &api.ListVlansRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Vlans, ShouldResembleProto, vlans)
		})

		Convey("ListVlans - listing with pagination", func() {
			req := &api.ListVlansRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Vlans, ShouldResembleProto, vlans[:3])

			req = &api.ListVlansRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListVlans(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Vlans, ShouldResembleProto, vlans[3:])
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
			req := &api.CreateVlanRequest{
				Vlan:   vlan2,
				VlanId: "vlan-2",
			}
			resp, err := tf.Fleet.CreateVlan(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan2)

			dreq := &api.DeleteVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			_, err = tf.Fleet.DeleteVlan(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &api.GetVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			res, err := tf.Fleet.GetVlan(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete vlan by non-existing ID", func() {
			req := &api.DeleteVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "vlan-2"),
			}
			_, err := tf.Fleet.DeleteVlan(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete vlan - Invalid input empty name", func() {
			req := &api.DeleteVlanRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete vlan - Invalid input invalid characters", func() {
			req := &api.DeleteVlanRequest{
				Name: util.AddPrefix(util.VlanCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteVlan(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
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
			req := &api.ImportVlansRequest{
				Source: &api.ImportVlansRequest_ConfigSource{
					ConfigSource: &api.ConfigSource{
						ConfigServiceName: "fake-service",
						FileName:          "fakeVlans.cfg",
					},
				},
			}
			tf.Fleet.importPageSize = 25
			res, err := tf.Fleet.ImportVlans(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			vlans, _, err := configuration.ListVlans(ctx, 100, "")
			So(err, ShouldBeNil)
			So(api.ParseResources(vlans, "Name"), ShouldResemble, []string{"browser-lab:144", "browser-lab:20", "browser-lab:40"})
			vlan, err := configuration.GetVlan(ctx, "browser-lab:40")
			So(err, ShouldBeNil)
			expectedCapacity := getCapacity(vlan.GetVlanAddress())
			So(vlan.GetCapacityIp(), ShouldEqual, int32(expectedCapacity))
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"vlan": "browser-lab:40"})
			So(err, ShouldBeNil)
			So(len(ips), ShouldEqual, expectedCapacity)
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
			req := &api.ImportOSVlansRequest{
				Source: &api.ImportOSVlansRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
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
		gets[i] = r.Data.(*proto.ChromePlatform).GetName()
	}
	return gets
}
