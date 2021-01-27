package handler

import (
	"context"
	"testing"
	"time"

	"infra/appengine/sheriff-o-matic/som/analyzer"
	"infra/monitoring/messages"

	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/gae/impl/dummy"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/gae/service/info"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/logging/gologger"
)

func newTestContext() context.Context {
	c := gaetesting.TestingContext()
	ta := datastore.GetTestable(c)
	ta.Consistent(true)
	c = gologger.StdConfig.Use(c)
	return c
}

type giMock struct {
	info.RawInterface
	token  string
	expiry time.Time
	err    error
}

func (gi giMock) AccessToken(scopes ...string) (token string, expiry time.Time, err error) {
	return gi.token, gi.expiry, gi.err
}

type mockFindit struct {
	res []*messages.FinditResultV2
	err error
}

func (mf *mockFindit) FinditBuildbucket(ctx context.Context, id int64, stepNames []string) ([]*messages.FinditResultV2, error) {
	return mf.res, mf.err
}

func TestAttachFinditResults(t *testing.T) {
	Convey("smoke", t, func() {
		c := gaetesting.TestingContext()
		bf := []*messages.BuildFailure{
			{
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "some step",
					},
				},
			},
		}
		fc := &mockFindit{}
		attachFindItResults(c, bf, fc)
		So(len(bf), ShouldEqual, 1)
	})

	Convey("some results", t, func() {
		c := newTestContext()
		bf := []*messages.BuildFailure{
			{
				Builders: []*messages.AlertedBuilder{
					{
						Name: "some builder",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "some step",
					},
				},
			},
		}
		fc := &mockFindit{
			res: []*messages.FinditResultV2{{
				StepName: "some step",
				Culprits: []*messages.Culprit{
					{
						Commit: &messages.GitilesCommit{
							Host:           "githost",
							Project:        "proj",
							ID:             "0xdeadbeef",
							CommitPosition: 1234,
						},
					},
				},
				IsFinished:  true,
				IsSupported: true,
			}},
		}
		attachFindItResults(c, bf, fc)
		So(len(bf), ShouldEqual, 1)
		So(len(bf[0].Culprits), ShouldEqual, 1)
		So(bf[0].HasFindings, ShouldEqual, true)
	})
}

func TestStoreAlertsSummary(t *testing.T) {
	Convey("success", t, func() {
		c := gaetesting.TestingContext()
		c = info.SetFactory(c, func(ic context.Context) info.RawInterface {
			return giMock{dummy.Info(), "", clock.Now(c), nil}
		})
		a := analyzer.New(5, 100)
		err := storeAlertsSummary(c, a, "some tree", &messages.AlertsSummary{
			Alerts: []*messages.Alert{
				{
					Title: "foo",
					Extension: &messages.BuildFailure{
						RegressionRanges: []*messages.RegressionRange{
							{Repo: "some repo", URL: "about:blank", Positions: []string{}, Revisions: []string{}},
						},
					},
				},
			},
		})
		So(err, ShouldBeNil)
	})
}
