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
	"infra/unifiedfleet/app/util"

	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
)

var localPlatforms = []*crimsonconfig.Platform{
	{Name: "platform1"},
	{Name: "platform2"},
	{Name: "platform3"},
}

func mockParsePlatformsFunc(path string) (*crimsonconfig.Platforms, error) {
	return &crimsonconfig.Platforms{
		Platform: localPlatforms,
	}, nil
}

func mockChromePlatform(id, desc string) *proto.ChromePlatform {
	return &proto.ChromePlatform{
		Name:        util.AddPrefix(chromePlatformCollection, id),
		Description: desc,
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
				Name: util.AddPrefix(chromePlatformCollection, "chromePlatform-1"),
			}
			resp, err := tf.Fleet.GetChromePlatform(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
		})
		Convey("Get chromePlatform by non-existing ID", func() {
			req := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(chromePlatformCollection, "chromePlatform-2"),
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
				Name: util.AddPrefix(chromePlatformCollection, "a.b)7&"),
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
				Name: util.AddPrefix(machineCollection, "machine-1"),
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
				Name: util.AddPrefix(chromePlatformCollection, "chromePlatform-1"),
			}
			_, err = tf.Fleet.DeleteChromePlatform(tf.C, dreq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.CannotDelete)

			greq := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(chromePlatformCollection, "chromePlatform-1"),
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
				Name: util.AddPrefix(chromePlatformCollection, "chromePlatform-2"),
			}
			_, err = tf.Fleet.DeleteChromePlatform(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &api.GetChromePlatformRequest{
				Name: util.AddPrefix(chromePlatformCollection, "chromePlatform-2"),
			}
			res, err := tf.Fleet.GetChromePlatform(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})

		Convey("Delete chromePlatform by non-existing ID", func() {
			req := &api.DeleteChromePlatformRequest{
				Name: util.AddPrefix(chromePlatformCollection, "chromePlatform-2"),
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
				Name: util.AddPrefix(chromePlatformCollection, "a.b)7&"),
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
			wants := getLocalPlatformNames()
			gets := getReturnedPlatformNames(*getRes)
			So(gets, ShouldResemble, wants)
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

func getLocalPlatformNames() []string {
	wants := make([]string, len(localPlatforms))
	for i, p := range localPlatforms {
		wants[i] = p.GetName()
	}
	return wants
}

func getReturnedPlatformNames(res datastore.OpResults) []string {
	gets := make([]string, len(res))
	for i, r := range res {
		gets[i] = r.Data.(*proto.ChromePlatform).GetName()
	}
	return gets
}
