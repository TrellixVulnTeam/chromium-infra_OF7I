package fakelegacy_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sort"
	"testing"

	"infra/chromeperf/pinpoint"
	"infra/chromeperf/pinpoint/fakelegacy"
	"infra/chromeperf/pinpoint/server"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	. "github.com/smartystreets/goconvey/convey"
)

// Path to the directory which contains templates for API responses.
const templateDir = "templates/"

// TODO(chowski): move this to a more common location for reuse.
func shouldBeStatusError(got interface{}, want ...interface{}) string {
	err := got.(error)
	wantCode := want[0].(codes.Code)
	s, ok := status.FromError(err)
	if !ok {
		return fmt.Sprintf("error was not a Status error, found %T", err)
	}
	return ShouldEqual(s.Code(), wantCode)
}

func TestStaticUsage(t *testing.T) {
	const (
		user   = "user@example.com"
		jobID0 = "00000000000000"
		jobID1 = "11111111111111"
	)
	legacyName0 := pinpoint.LegacyJobName(jobID0)
	legacyName1 := pinpoint.LegacyJobName(jobID1)
	fake, err := fakelegacy.NewServer(
		templateDir,
		map[string]*fakelegacy.Job{
			jobID0: {
				ID:        jobID0,
				Status:    fakelegacy.CompletedStatus,
				UserEmail: user,
			},
			jobID1: {
				ID:        jobID1,
				Status:    fakelegacy.CompletedStatus,
				UserEmail: user,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(fake.Handler())
	defer ts.Close()

	grpcPinpoint := server.New(ts.URL, ts.Client())

	ctx := context.Background()
	Convey("GetJob should return known job", t, func() {
		j, err := grpcPinpoint.GetJob(ctx, &pinpoint.GetJobRequest{Name: legacyName0})
		So(err, ShouldBeNil)
		So(j.Name, ShouldEqual, legacyName0)
	})
	Convey("GetJob should return NotFound for unknown job", t, func() {
		_, err := grpcPinpoint.GetJob(ctx, &pinpoint.GetJobRequest{Name: pinpoint.LegacyJobName("86753098675309")})
		So(err, shouldBeStatusError, codes.NotFound)
	})
	Convey("ListJobs should return both known jobs", t, func() {
		list, err := grpcPinpoint.ListJobs(ctx, &pinpoint.ListJobsRequest{})
		So(err, ShouldBeNil)
		So(list.Jobs, ShouldHaveLength, 2)

		sort.Slice(list.Jobs, func(i, j int) bool {
			return list.Jobs[i].Name < list.Jobs[j].Name
		})
		So(list.Jobs[0].Name, ShouldEqual, legacyName0)
		So(list.Jobs[1].Name, ShouldEqual, legacyName1)
	})
}
