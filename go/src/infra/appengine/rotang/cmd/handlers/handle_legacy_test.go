package handlers

import (
	"infra/appengine/rotang"
	"net/http/httptest"
	"testing"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/server/router"
)

func TestLegacySheriff(t *testing.T) {
	ctx := newTestContext()

	tests := []struct {
		name       string
		fail       bool
		calFail    bool
		time       time.Time
		calShifts  []rotang.ShiftEntry
		ctx        *router.Context
		file       string
		memberPool []rotang.Member
		cfgs       []*rotang.Configuration
		want       string
	}{{
		name: "Success JSON",
		ctx: &router.Context{
			Context: ctx,
			Writer:  httptest.NewRecorder(),
		},
		file: "sheriff_ios.json",
		time: midnight.Add(6 * fullDay),
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Chrome iOS Build Sheriff",
				},
			},
		},
		calShifts: []rotang.ShiftEntry{
			{
				StartTime: midnight,
				EndTime:   midnight.Add(5 * fullDay),
				OnCall: []rotang.ShiftMember{
					{
						Email: "test1@oncall.com",
					}, {
						Email: "test2@oncall.com",
					},
				},
			}, {
				StartTime: midnight.Add(5 * fullDay),
				EndTime:   midnight.Add(10 * fullDay),
				OnCall: []rotang.ShiftMember{
					{
						Email: "test3@oncall.com",
					}, {
						Email: "test4@oncall.com",
					},
				},
			},
		},
		want: `{"updated_unix_timestamp":1144454400,"emails":["test3@oncall.com","test4@oncall.com"]}
`,
	}, {
		name: "File not supported",
		fail: true,
		ctx: &router.Context{
			Context: ctx,
			Writer:  httptest.NewRecorder(),
		},
		file: "sheriff_not_supported.js",
		time: midnight,
	}, {
		name: "Config not found",
		fail: true,
		ctx: &router.Context{
			Context: ctx,
			Writer:  httptest.NewRecorder(),
		},
		file: "sheriff_ios.js",
		time: midnight,
	}, {
		name:    "Calendar fail",
		fail:    true,
		calFail: true,
		ctx: &router.Context{
			Context: ctx,
			Writer:  httptest.NewRecorder(),
		},
		file: "sheriff_ios.json",
		time: midnight.Add(6 * fullDay),
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Chrome iOS Build Sheriff",
				},
			},
		},
	},
	}

	h := testSetup(t)

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			for _, m := range tst.memberPool {
				if err := h.memberStore(ctx).CreateMember(ctx, &m); err != nil {
					t.Fatalf("%s: CreateMember(ctx, _) failed: %v", tst.name, err)
				}
				defer h.memberStore(ctx).DeleteMember(ctx, m.Email)
			}
			for _, cfg := range tst.cfgs {
				if err := h.configStore(ctx).CreateRotaConfig(ctx, cfg); err != nil {
					t.Fatalf("%s: CreateRotaConfig(ctx, _) failed: %v", tst.name, err)
				}
				defer h.configStore(ctx).DeleteRotaConfig(ctx, cfg.Config.Name)
			}

			h.legacyCalendar.(*fakeCal).Set(tst.calShifts, tst.calFail, false, 0)

			tst.ctx.Context = clock.Set(tst.ctx.Context, testclock.New(tst.time))

			res, err := h.legacySheriff(tst.ctx, tst.file)
			if got, want := (err != nil), tst.fail; got != want {
				t.Fatalf("%s: h.legacySheriff(ctx, %q) = %t want: %t, err: %v", tst.name, tst.file, got, want, err)
			}
			if err != nil {
				return
			}

			if diff := prettyConfig.Compare(tst.want, res); diff != "" {
				t.Fatalf("%s: h.legacySheriff(ctx, %q) differ -want +got, \n%s", tst.name, tst.file, diff)
			}
		})
	}
}
