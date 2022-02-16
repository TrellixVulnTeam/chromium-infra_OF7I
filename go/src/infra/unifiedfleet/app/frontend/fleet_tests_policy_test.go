package frontend

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"

	api "infra/unifiedfleet/api/v1/rpc"
)

func TestGetPublicChromiumTestStatus(t *testing.T) {
	t.Parallel()
	ctx := auth.WithState(context.Background(), &authtest.FakeState{
		Identity:       "user:abc@def.com",
		IdentityGroups: []string{"public-chromium-in-chromeos-builders"},
	})
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Check Fleet Policy For Tests", t, func() {
		Convey("happy path", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Board:    "eve",
				Model:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}

			res, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldBeNil)
			So(res.TestStatus.Code, ShouldEqual, api.TestStatus_OK)
		})
		Convey("Private board", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Board:    "private",
				Model:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}

			res, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldBeNil)
			So(res.TestStatus.Code, ShouldEqual, api.TestStatus_NOT_A_PUBLIC_BOARD)
		})
		Convey("Private model", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Board:    "eve",
				Model:    "private",
				Image:    "R100-14495.0.0-rc1",
			}

			res, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldBeNil)
			So(res.TestStatus.Code, ShouldEqual, api.TestStatus_NOT_A_PUBLIC_MODEL)
		})
		Convey("Non allowlisted image", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Board:    "eve",
				Model:    "eve",
				Image:    "invalid",
			}

			res, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldBeNil)
			So(res.TestStatus.Code, ShouldEqual, api.TestStatus_NOT_A_PUBLIC_IMAGE)
		})
		Convey("Private test name and public auth group member", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "private",
				Board:    "eve",
				Model:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}

			res, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldBeNil)
			So(res.TestStatus.Code, ShouldEqual, api.TestStatus_NOT_A_PUBLIC_TEST)
		})
		Convey("Public test name and not a public auth group member", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Board:    "eve",
				Model:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}
			ctx := auth.WithState(context.Background(), &authtest.FakeState{
				Identity: "user:abc@def.com",
			})

			res, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldBeNil)
			So(res.TestStatus.Code, ShouldEqual, api.TestStatus_OK)
		})
		Convey("Private test name and not a public auth group member", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "private",
				Board:    "eve",
				Model:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}
			ctx := auth.WithState(context.Background(), &authtest.FakeState{
				Identity: "user:abc@def.com",
			})

			res, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldBeNil)
			So(res.TestStatus.Code, ShouldEqual, api.TestStatus_OK)
		})
		Convey("Missing Test names", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "",
				Board:    "eve",
				Model:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}

			_, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Test name cannot be empty")
		})
		Convey("Missing Board", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Model:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}

			_, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Board cannot be empty")
		})
		Convey("Missing Models", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Board:    "eve",
				Image:    "R100-14495.0.0-rc1",
			}

			_, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Model cannot be empty")
		})
		Convey("Missing Image", func() {
			req := &api.CheckFleetTestsPolicyRequest{
				TestName: "tast.lacros",
				Board:    "eve",
				Model:    "eve",
			}

			_, err := tf.Fleet.CheckFleetTestsPolicy(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Image cannot be empty")
		})
	})
}
