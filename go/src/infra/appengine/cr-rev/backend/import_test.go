package main

import (
	"infra/appengine/cr-rev/backend/gitiles"
	"infra/appengine/cr-rev/backend/repoimport"
	"infra/appengine/cr-rev/common"
	"testing"

	"infra/appengine/cr-rev/config"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

func TestInitialImport(t *testing.T) {
	ctx := gaetesting.TestingContext()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("invalid host", t, func() {
		cfg := &config.Config{
			Hosts: []*config.Host{
				{
					Name: "invalid/name",
				},
			},
		}
		So(func() {
			setupImport(ctx, cfg)
		}, ShouldPanic)
	})
}

func TestInitialHostImport(t *testing.T) {
	ctx := gaetesting.TestingContext()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	controller := repoimport.NewMockController(mockCtrl)

	Convey("Skip repos", t, func() {
		// Setup gitiles
		fakeGitilesClient := &gitilesProto.GitilesFake{}
		fakeGitilesClient.SetRepository("foo", nil, nil)
		fakeGitilesClient.SetRepository("bar", nil, nil)
		ctx := gitiles.SetClient(ctx, fakeGitilesClient)

		// Setup config
		barRepoConfig := &config.Repository{
			Name: "bar",
			Indexing: &config.Repository_DoNotIndex{
				DoNotIndex: true,
			},
		}
		host := &config.Host{
			Name: "host",
			Repos: []*config.Repository{
				barRepoConfig,
			},
		}

		// setup mock
		controller.EXPECT().Index(common.GitRepository{
			Host: "host",
			Name: "foo",
		}).Times(1)

		initialHostImport(ctx, controller, host)
		// We expect only one Gitiles calls (to list projects):
		So(len(fakeGitilesClient.GetCallLogs()), ShouldEqual, 1)
	})
}
