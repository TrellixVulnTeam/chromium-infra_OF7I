package api

import (
	"infra/appengine/cr-rev/frontend/redirect"
	"infra/appengine/cr-rev/models"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func TestServer(t *testing.T) {
	ctx := gaetesting.TestingContext()
	ds := datastore.GetTestable(ctx)
	ds.Consistent(true)
	ds.AutoIndex(true)
	s := NewServer(redirect.NewRules(redirect.NewGitilesRedirect()))
	commits := []*models.Commit{
		{
			ID:             "chromium-chromium/src-0000000000000000000000000000000000000001",
			CommitHash:     "0000000000000000000000000000000000000001",
			Host:           "chromium",
			Repository:     "chromium/src",
			PositionNumber: 1,
			PositionRef:    "svn://svn.chromium.org/chrome",
		},
		{
			ID:             "chromium-chromium/src-0000000000000000000000000000000000000002",
			CommitHash:     "0000000000000000000000000000000000000002",
			Host:           "chromium",
			Repository:     "chromium/src",
			PositionNumber: 2,
			PositionRef:    "refs/heads/main",
		},
		{
			ID:             "chromium-foo-0000000000000000000000000000000000000003",
			CommitHash:     "0000000000000000000000000000000000000003",
			Host:           "chromium",
			Repository:     "foo",
			PositionNumber: 3,
			PositionRef:    "refs/heads/main",
		},
	}
	datastore.Put(ctx, commits)

	Convey("redirect", t, func() {
		Convey("empty request", func() {
			_, err := s.Redirect(ctx, &RedirectRequest{})
			So(err, ShouldBeError)
			s, _ := status.FromError(err)
			So(s.Code(), ShouldEqual, codes.NotFound)
		})
		Convey("matching path found", func() {
			resp, err := s.Redirect(ctx, &RedirectRequest{
				Query: "/1",
			})
			So(err, ShouldBeNil)
			expected := &RedirectResponse{
				GitHash:     "0000000000000000000000000000000000000001",
				Host:        "chromium",
				Repository:  "chromium/src",
				RedirectUrl: "https://chromium.googlesource.com/chromium/src/+/0000000000000000000000000000000000000001",
			}
			So(resp, ShouldResemble, expected)
		})
		Convey("not chromium/src", func() {
			_, err := s.Redirect(ctx, &RedirectRequest{
				Query: "/3",
			})
			So(err, ShouldBeError)
			s, _ := status.FromError(err)
			So(s.Code(), ShouldEqual, codes.NotFound)
		})
	})

	Convey("Numbering", t, func() {
		Convey("empty request", func() {
			_, err := s.Numbering(ctx, &NumberingRequest{})
			So(err, ShouldBeError)
			s, _ := status.FromError(err)
			So(s.Code(), ShouldEqual, codes.NotFound)
		})
		Convey("not found", func() {
			_, err := s.Numbering(ctx, &NumberingRequest{
				PositionNumber: 3,
				Host:           "chromium",
				Repository:     "chromium/src",
				PositionRef:    "svn://svn.chromium.org/chrome",
			})
			So(err, ShouldBeError)
			s, _ := status.FromError(err)
			So(s.Code(), ShouldEqual, codes.NotFound)
		})
		Convey("chromium/src", func() {
			resp, err := s.Numbering(ctx, &NumberingRequest{
				PositionNumber: 1,
				Host:           "chromium",
				Repository:     "chromium/src",
				PositionRef:    "svn://svn.chromium.org/chrome",
			})
			So(err, ShouldBeNil)
			expected := &NumberingResponse{
				GitHash:        "0000000000000000000000000000000000000001",
				PositionNumber: 1,
				Host:           "chromium",
				Repository:     "chromium/src",
			}
			So(resp, ShouldResemble, expected)
		})
		Convey("arbitrary repository", func() {
			resp, err := s.Numbering(ctx, &NumberingRequest{
				PositionNumber: 3,
				Host:           "chromium",
				Repository:     "foo",
				PositionRef:    "refs/heads/main",
			})
			So(err, ShouldBeNil)
			expected := &NumberingResponse{
				GitHash:        "0000000000000000000000000000000000000003",
				PositionNumber: 3,
				Host:           "chromium",
				Repository:     "foo",
			}
			So(resp, ShouldResemble, expected)
		})
	})

	Convey("Commit", t, func() {
		Convey("empty request", func() {
			_, err := s.Commit(ctx, &CommitRequest{})
			So(err, ShouldBeError)
			s, _ := status.FromError(err)
			So(s.Code(), ShouldEqual, codes.NotFound)
		})
		Convey("not found", func() {
			_, err := s.Commit(ctx, &CommitRequest{
				GitHash: "0000000000000000000000000000000000000000",
			})
			So(err, ShouldBeError)
			s, _ := status.FromError(err)
			So(s.Code(), ShouldEqual, codes.NotFound)
		})
		Convey("chromium/src", func() {
			resp, err := s.Commit(ctx, &CommitRequest{
				GitHash: "0000000000000000000000000000000000000001",
			})
			So(err, ShouldBeNil)
			expected := &CommitResponse{
				GitHash:        "0000000000000000000000000000000000000001",
				PositionNumber: 1,
				Host:           "chromium",
				Repository:     "chromium/src",
				RedirectUrl:    "https://chromium.googlesource.com/chromium/src/+/0000000000000000000000000000000000000001",
			}
			So(resp, ShouldResemble, expected)
		})
	})
}
