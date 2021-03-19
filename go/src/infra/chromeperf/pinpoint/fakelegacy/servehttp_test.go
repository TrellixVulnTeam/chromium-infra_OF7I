package fakelegacy_test

import (
	"context"
	"fmt"
	"net/http/httptest"
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

func TestGetJob(t *testing.T) {
	const (
		jobID = "abcdef01234567"
		user  = "user@example.com"
	)
	legacyName := pinpoint.LegacyJobName(jobID)
	fake, err := fakelegacy.NewServer(
		templateDir,
		map[string]*fakelegacy.Job{
			jobID: {
				ID:        jobID,
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
		j, err := grpcPinpoint.GetJob(ctx, &pinpoint.GetJobRequest{Name: legacyName})
		So(err, ShouldBeNil)
		So(j.Name, ShouldEqual, legacyName)
	})
	Convey("GetJob should return NotFound for unknown job", t, func() {
		_, err := grpcPinpoint.GetJob(ctx, &pinpoint.GetJobRequest{Name: pinpoint.LegacyJobName("86753098675309")})
		So(err, shouldBeStatusError, codes.NotFound)
	})
}
