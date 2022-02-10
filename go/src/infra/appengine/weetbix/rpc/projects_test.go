package rpc

import (
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/secrets/testsecrets"

	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

func TestProjects(t *testing.T) {
	Convey("Given a projects server", t, func() {

		ctx := testutil.SpannerTestContext(t)

		// For user identification.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: "user:someone@example.com",
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		// Provides datastore implementation needed for project config.
		ctx = memory.Use(ctx)
		server := &projectServer{}

		Convey("When a list of projects is provided", func() {
			// Setup
			projectChromium := config.CreatePlaceholderConfig()
			projectChrome := config.CreatePlaceholderConfigWithKey("chrome")
			configs := make(map[string]*configpb.ProjectConfig)
			configs["chromium"] = projectChromium
			configs["chrome"] = projectChrome
			config.SetTestProjectConfig(ctx, configs)

			// Run
			request := &pb.ListProjectsRequest{}
			projectsResponse, err := server.List(ctx, request)

			// Verify
			So(err, ShouldBeNil)
			expected := &pb.ListProjectsResponse{Projects: []*pb.Project{
				{
					Name:        "projects/chrome",
					DisplayName: "Chrome",
				},
				{
					Name:        "projects/chromium",
					DisplayName: "Chromium",
				},
			}}
			sort.Slice(expected.Projects, func(i, j int) bool {
				return expected.Projects[i].Name < expected.Projects[j].Name
			})
			sort.Slice(projectsResponse.Projects, func(i, j int) bool {
				return projectsResponse.Projects[i].Name < projectsResponse.Projects[j].Name
			})
			So(projectsResponse, ShouldResembleProto, expected)
		})

		Convey("When a displayName is empty", func() {
			// Setup
			emptyKeyProject := config.CreatePlaceholderConfigWithKey("")
			configs := make(map[string]*configpb.ProjectConfig)
			configs["chrome"] = emptyKeyProject
			config.SetTestProjectConfig(ctx, configs)

			// Run
			request := &pb.ListProjectsRequest{}
			projectsResponse, err := server.List(ctx, request)

			// Verify
			expected := &pb.ListProjectsResponse{Projects: []*pb.Project{
				{
					Name:        "projects/chrome",
					DisplayName: "Chrome",
				},
			}}
			So(err, ShouldBeNil)
			So(projectsResponse, ShouldResembleProto, expected)
		})

	})

}
