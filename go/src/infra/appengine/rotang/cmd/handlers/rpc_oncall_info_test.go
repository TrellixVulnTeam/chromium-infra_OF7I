package handlers

import (
	"context"
	"infra/appengine/rotang"
	"sort"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"
	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"

	apb "infra/appengine/rotang/proto/rotangapi"
)

func TestRPCShifts(t *testing.T) {
	ctx := newTestContext()

	tests := []struct {
		name    string
		fail    bool
		ctx     context.Context
		in      string
		cfgs    []*rotang.Configuration
		members []rotang.Member
		shifts  []rotang.ShiftEntry
		time    time.Time
		want    string
	}{{
		name: "Success",
		in: `
			name: "Test Rota"
			start: {
				seconds: 1143849600,
			},
			end:   {
				seconds: 1144022400,
			},
		`,
		time: midnight,
		ctx:  ctx,
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Test Rota",
					Shifts: rotang.ShiftConfig{
						Generator: "Fair",
						Shifts: []rotang.Shift{
							{
								Name: "MTV All Day",
							},
						},
					},
				},
				Members: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					}, {
						Email:     "mtv2@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		members: []rotang.Member{
			{
				Email: "mtv1@oncall.com",
				Name:  "Mtv1 Mtvsson",
				TZ:    *time.UTC,
			}, {
				Email: "mtv2@oncall.com",
				Name:  "Mtv2 Mtvsson",
				TZ:    *time.UTC,
			},
		},
		shifts: []rotang.ShiftEntry{
			{
				Name:      "MTV All Day",
				StartTime: midnight.Add(-1 * fullDay),
				EndTime:   midnight.Add(fullDay),
				OnCall: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		want: `
		shifts: <
			name: "MTV All Day"
			oncallers: <
					email: "mtv1@oncall.com",
					name: "Mtv1 Mtvsson",
					tz: "UTC",
			>,
			start: {
				seconds: 1143849600,
			},
			end: {
				seconds: 1144022400,
			},
    >
		`,
	}, {
		name: "No rota name",
		fail: true,
		ctx:  ctx,
	}, {
		name: "Non existing rota",
		fail: true,
		ctx:  ctx,
		in:   `name: "Non Exist"`,
	}, {
		name: "No Shifts",
		in: `
			name: "Test Rota"
			start: {
				seconds: 1143849600,
			},
			end:   {
				seconds: 1144022400,
			},
		`,
		time: midnight,
		ctx:  ctx,
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Test Rota",
					Shifts: rotang.ShiftConfig{
						Generator: "Fair",
						Shifts: []rotang.Shift{
							{
								Name: "MTV All Day",
							},
						},
					},
				},
				Members: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					}, {
						Email:     "mtv2@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		members: []rotang.Member{
			{
				Email: "mtv1@oncall.com",
				Name:  "Mtv1 Mtvsson",
				TZ:    *time.UTC,
			}, {
				Email: "mtv2@oncall.com",
				Name:  "Mtv2 Mtvsson",
				TZ:    *time.UTC,
			},
		},
		shifts: []rotang.ShiftEntry{
			{
				Name:      "MTV All Day",
				StartTime: midnight.Add(2 * fullDay),
				EndTime:   midnight.Add(4 * fullDay),
				OnCall: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
	}}

	h := testSetup(t)

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			for _, m := range tst.members {
				if err := h.memberStore(ctx).CreateMember(ctx, &m); err != nil {
					t.Fatalf("%s: CreateMember(ctx, _) failed: %v", tst.name, err)
				}
				defer h.memberStore(ctx).DeleteMember(ctx, m.Email)
			}
			for _, c := range tst.cfgs {
				if err := h.configStore(ctx).CreateRotaConfig(ctx, c); err != nil {
					t.Fatalf("%s: CreateRotaconfig(ctx, _) failed: %v", tst.name, err)
				}
				defer h.configStore(ctx).DeleteRotaConfig(ctx, c.Config.Name)
				if err := h.shiftStore(ctx).AddShifts(ctx, c.Config.Name, tst.shifts); err != nil {
					t.Fatalf("%s: AddShifts(ctx, %q, _) failed: %v", tst.name, c.Config.Name, err)
				}
				defer h.shiftStore(ctx).DeleteAllShifts(ctx, c.Config.Name)
			}

			tst.ctx = clock.Set(tst.ctx, testclock.New(tst.time))

			var inPB apb.ShiftsRequest
			if err := proto.UnmarshalText(tst.in, &inPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			var wantPB apb.ShiftsResponse
			if err := proto.UnmarshalText(tst.want, &wantPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			res, err := h.Shifts(tst.ctx, &inPB)
			if got, want := (err != nil), tst.fail; got != want {
				t.Fatalf("%s: h.Oncall(ctx, _) = %t want: %t, err: %v", tst.name, got, want, err)
			}

			if err != nil {
				return
			}

			if diff := pretty.Compare(wantPB, res); diff != "" {
				t.Fatalf("%s: h.Oncall(ctx, _) differ -want +got: %s", tst.name, diff)
			}
		})
	}
}

type tzByName []*apb.TZGroup

func (t tzByName) Less(i, j int) bool {
	return t[i].BusinessGroup < t[j].BusinessGroup
}

func (t tzByName) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t tzByName) Len() int {
	return len(t)
}

func TestRPCMigrationInfo(t *testing.T) {
	ctx := newTestContext()

	mtvTime, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("time.LoadLocation(_) failed: %v", err)
	}
	nyTime, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("time.LoadLocation(_) failed: %v", err)
	}
	euTime, err := time.LoadLocation("Europe/Brussels")
	if err != nil {
		t.Fatalf("time.LoadLocation(_) failed: %v", err)
	}
	apacTime, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatalf("time.LoadLocation(_) failed: %v", err)
	}

	tests := []struct {
		name    string
		fail    bool
		ctx     context.Context
		user    string
		in      string
		cfgs    []*rotang.Configuration
		members []rotang.Member
		shifts  []rotang.ShiftEntry
		time    time.Time
		want    string
	}{
		{
			name: "Success",
			in: `
				name: "Test Rota",
			`,
			ctx:  ctx,
			user: "owner@owner.com",
			cfgs: []*rotang.Configuration{
				{
					Config: rotang.Config{
						Name:   "Test Rota",
						Owners: []string{"owner@owner.com"},
						Shifts: rotang.ShiftConfig{
							Generator: "Fair",
							Shifts: []rotang.Shift{
								{
									Name: "MTV All Day",
								},
							},
						},
					},
					Members: []rotang.ShiftMember{
						{
							Email:     "mtv1@oncall.com",
							ShiftName: "MTV All Day",
						}, {
							Email:     "mtv2@oncall.com",
							ShiftName: "MTV All Day",
						},
					},
				},
			},
			members: []rotang.Member{
				{
					Email: "mtv1@oncall.com",
					Name:  "Mtv1 Mtvsson",
					TZ:    *time.UTC,
				}, {
					Email: "mtv2@oncall.com",
					Name:  "Mtv2 Mtvsson",
					TZ:    *time.UTC,
				},
			},
			shifts: []rotang.ShiftEntry{
				{
					Name:      "MTV All Day",
					StartTime: midnight.Add(-1 * fullDay),
					EndTime:   midnight.Add(fullDay),
					OnCall: []rotang.ShiftMember{
						{
							Email:     "mtv1@oncall.com",
							ShiftName: "MTV All Day",
						},
					},
				},
			},
			want: `name: "Test Rota"
					owners: "owner@owner.com"
					members: <
						business_group: "default"
						members: <
							email: "mtv1@oncall.com"
							name: "Mtv1 Mtvsson"
							tz: "UTC"
						>
						members: <
							email: "mtv2@oncall.com"
							name: "Mtv2 Mtvsson"
							tz: "UTC"
						>
					>
					shifts: <
							name: "MTV All Day"
						oncallers: <
							email: "mtv1@oncall.com"
							name: "Mtv1 Mtvsson"
							tz: "UTC"
						>
						start: <
							seconds: 1143849600
						>
						end: <
							seconds: 1144022400
						>
					>
			`,
		}, {
			name: "Not owner",
			fail: true,
			in: `
				name: "Test Rota",
			`,
			ctx:  ctx,
			user: "notowner@owner.com",
			cfgs: []*rotang.Configuration{
				{
					Config: rotang.Config{
						Name:   "Test Rota",
						Owners: []string{"owner@owner.com"},
						Shifts: rotang.ShiftConfig{
							Generator: "Fair",
							Shifts: []rotang.Shift{
								{
									Name: "MTV All Day",
								},
							},
						},
					},
					Members: []rotang.ShiftMember{
						{
							Email:     "mtv1@oncall.com",
							ShiftName: "MTV All Day",
						}, {
							Email:     "mtv2@oncall.com",
							ShiftName: "MTV All Day",
						},
					},
				},
			},
			members: []rotang.Member{
				{
					Email: "mtv1@oncall.com",
					Name:  "Mtv1 Mtvsson",
					TZ:    *time.UTC,
				}, {
					Email: "mtv2@oncall.com",
					Name:  "Mtv2 Mtvsson",
					TZ:    *time.UTC,
				},
			},
			shifts: []rotang.ShiftEntry{
				{
					Name:      "MTV All Day",
					StartTime: midnight.Add(-1 * fullDay),
					EndTime:   midnight.Add(fullDay),
					OnCall: []rotang.ShiftMember{
						{
							Email:     "mtv1@oncall.com",
							ShiftName: "MTV All Day",
						},
					},
				},
			},
		}, {
			name: "TZ aware",
			in: `
				name: "Test Rota",
			`,
			ctx:  ctx,
			user: "owner@owner.com",
			cfgs: []*rotang.Configuration{
				{
					Config: rotang.Config{
						Name:   "Test Rota",
						Owners: []string{"owner@owner.com"},
						Shifts: rotang.ShiftConfig{
							Generator: "TZRecent",
							Shifts: []rotang.Shift{
								{
									Name: "MTV All Day",
								},
							},
						},
					},
					Members: []rotang.ShiftMember{
						{
							Email:     "mtv1@oncall.com",
							ShiftName: "MTV All Day",
						}, {
							Email:     "ny1@oncall.com",
							ShiftName: "MTV All Day",
						}, {
							Email:     "syd1@oncall.com",
							ShiftName: "MTV All Day",
						}, {
							Email:     "eu1@oncall.com",
							ShiftName: "MTV All Day",
						},
					},
				},
			},
			members: []rotang.Member{
				{
					Email: "mtv1@oncall.com",
					Name:  "Mtv1",
					TZ:    *mtvTime,
				}, {
					Email: "ny1@oncall.com",
					Name:  "NY1",
					TZ:    *nyTime,
				}, {
					Email: "syd1@oncall.com",
					Name:  "SYD1",
					TZ:    *apacTime,
				}, {
					Email: "eu1@oncall.com",
					Name:  "EU",
					TZ:    *euTime,
				},
			},
			shifts: []rotang.ShiftEntry{
				{
					Name:      "MTV All Day",
					StartTime: midnight.Add(-1 * fullDay),
					EndTime:   midnight.Add(fullDay),
					OnCall: []rotang.ShiftMember{
						{
							Email:     "mtv1@oncall.com",
							ShiftName: "MTV All Day",
						},
					},
				},
			},
			want: `name: "Test Rota"
				 owners: "owner@owner.com"
					tz_consider: true
					members: <
						business_group: "AMER-WEST"
						members: <
							email: "mtv1@oncall.com"
							name: "Mtv1"
							tz: "America/Los_Angeles"
						>
					>
					members: <
						business_group: "AMER-EAST"
						members: <
							email: "ny1@oncall.com"
							name: "NY1"
							tz: "America/New_York"
						>
					>
					members: <
						business_group: "APAC"
						members: <
							email: "syd1@oncall.com"
							name: "SYD1"
							tz: "Australia/Sydney"
						>
					>
					members: <
						business_group: "EMEA"
						members: <
							email: "eu1@oncall.com"
							name: "EU"
							tz: "Europe/Brussels"
						>
					>
						shifts: <
							name: "MTV All Day"
							oncallers: <
								email: "mtv1@oncall.com"
								name: "Mtv1"
								tz: "America/Los_Angeles"
							>
							start: <
								seconds: 1143849600
							>
							end: <
								seconds: 1144022400
							>
						>
		`,
		},
	}

	h := testSetup(t)

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			for _, m := range tst.members {
				if err := h.memberStore(ctx).CreateMember(ctx, &m); err != nil {
					t.Fatalf("%s: CreateMember(ctx, _) failed: %v", tst.name, err)
				}
				defer h.memberStore(ctx).DeleteMember(ctx, m.Email)
			}
			for _, c := range tst.cfgs {
				if err := h.configStore(ctx).CreateRotaConfig(ctx, c); err != nil {
					t.Fatalf("%s: CreateRotaconfig(ctx, _) failed: %v", tst.name, err)
				}
				defer h.configStore(ctx).DeleteRotaConfig(ctx, c.Config.Name)
				if err := h.shiftStore(ctx).AddShifts(ctx, c.Config.Name, tst.shifts); err != nil {
					t.Fatalf("%s: AddShifts(ctx, %q, _) failed: %v", tst.name, c.Config.Name, err)
				}
				defer h.shiftStore(ctx).DeleteAllShifts(ctx, c.Config.Name)
			}

			tst.ctx = clock.Set(tst.ctx, testclock.New(tst.time))

			var inPB apb.MigrationInfoRequest
			if err := proto.UnmarshalText(tst.in, &inPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			var wantPB apb.MigrationInfoResponse
			if err := proto.UnmarshalText(tst.want, &wantPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			if tst.user != "" {
				tst.ctx = auth.WithState(tst.ctx, &authtest.FakeState{
					Identity: identity.Identity("user:" + tst.user),
				})
			}

			res, err := h.MigrationInfo(tst.ctx, &inPB)
			if got, want := (err != nil), tst.fail; got != want {
				t.Fatalf("%s: h.MigrationInfo(ctx, _) = %t want: %t, err: %v", tst.name, got, want, err)
			}

			if err != nil {
				return
			}

			// As the members are sourced from a map the order is random.
			sort.Sort(tzByName(wantPB.Members))
			sort.Sort(tzByName(res.Members))

			if diff := pretty.Compare(wantPB, res); diff != "" {
				t.Fatalf("%s: h.MigrateInfo(ctx, _) differ -want +got: %s", tst.name, diff)
			}

		})
	}
}

func TestRPCOncall(t *testing.T) {
	ctx := newTestContext()

	tests := []struct {
		name    string
		fail    bool
		ctx     context.Context
		in      string
		cfgs    []*rotang.Configuration
		members []rotang.Member
		shifts  []rotang.ShiftEntry
		time    time.Time
		want    string
	}{{
		name: "Success",
		in:   `name: "Test Rota"`,
		time: midnight,
		ctx:  ctx,
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Test Rota",
					Shifts: rotang.ShiftConfig{
						Generator: "Fair",
						Shifts: []rotang.Shift{
							{
								Name: "MTV All Day",
							},
						},
					},
				},
				Members: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					}, {
						Email:     "mtv2@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		members: []rotang.Member{
			{
				Email: "mtv1@oncall.com",
				Name:  "Mtv1 Mtvsson",
				TZ:    *time.UTC,
			}, {
				Email: "mtv2@oncall.com",
				Name:  "Mtv2 Mtvsson",
				TZ:    *time.UTC,
			},
		},
		shifts: []rotang.ShiftEntry{
			{
				Name:      "MTV All Day",
				StartTime: midnight.Add(-1 * fullDay),
				EndTime:   midnight.Add(fullDay),
				OnCall: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		want: `shift: {
				name: "MTV All Day",
				oncallers: <
					email: "mtv1@oncall.com",
					name: "Mtv1 Mtvsson",
					tz: "UTC",
				>,
				start: {
					seconds: 1143849600,
				},
				end: {
					seconds: 1144022400,
				},
			}`,
	}, {
		name: "No rota name",
		fail: true,
		ctx:  ctx,
	}, {
		name: "TZConsider generator",
		in:   `name: "Test Rota"`,
		time: midnight,
		ctx:  ctx,
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Test Rota",
					Shifts: rotang.ShiftConfig{
						Generator: "TZRecent",
						Shifts: []rotang.Shift{
							{
								Name: "MTV All Day",
							},
						},
					},
				},
				Members: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					}, {
						Email:     "mtv2@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		members: []rotang.Member{
			{
				Email: "mtv1@oncall.com",
				Name:  "Mtv1 Mtvsson",
				TZ:    *time.UTC,
			}, {
				Email: "mtv2@oncall.com",
				Name:  "Mtv2 Mtvsson",
				TZ:    *time.UTC,
			},
		},
		shifts: []rotang.ShiftEntry{
			{
				Name:      "MTV All Day",
				StartTime: midnight.Add(-1 * fullDay),
				EndTime:   midnight.Add(fullDay),
				OnCall: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		want: `shift: {
				name: "MTV All Day",
				oncallers: <
					email: "mtv1@oncall.com",
					name: "Mtv1 Mtvsson",
					tz: "UTC",
				>,
				start: {
					seconds: 1143849600,
				},
				end: {
					seconds: 1144022400,
				},
			}
			tz_consider: true`,
	}, {
		name: "Nobody OnCall",
		in:   `name: "Test Rota"`,
		time: midnight,
		ctx:  ctx,
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Test Rota",
					Shifts: rotang.ShiftConfig{
						Generator: "TZRecent",
						Shifts: []rotang.Shift{
							{
								Name: "MTV All Day",
							},
						},
					},
				},
				Members: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					}, {
						Email:     "mtv2@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		members: []rotang.Member{
			{
				Email: "mtv1@oncall.com",
				Name:  "Mtv1 Mtvsson",
				TZ:    *time.UTC,
			}, {
				Email: "mtv2@oncall.com",
				Name:  "Mtv2 Mtvsson",
				TZ:    *time.UTC,
			},
		},
		shifts: []rotang.ShiftEntry{
			{
				Name:      "MTV All Day",
				StartTime: midnight.Add(fullDay),
				EndTime:   midnight.Add(2 * fullDay),
				OnCall: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		want: `tz_consider: true`,
	}}

	h := testSetup(t)

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			for _, m := range tst.members {
				if err := h.memberStore(ctx).CreateMember(ctx, &m); err != nil {
					t.Fatalf("%s: CreateMember(ctx, _) failed: %v", tst.name, err)
				}
				defer h.memberStore(ctx).DeleteMember(ctx, m.Email)
			}
			for _, c := range tst.cfgs {
				if err := h.configStore(ctx).CreateRotaConfig(ctx, c); err != nil {
					t.Fatalf("%s: CreateRotaconfig(ctx, _) failed: %v", tst.name, err)
				}
				defer h.configStore(ctx).DeleteRotaConfig(ctx, c.Config.Name)
				if err := h.shiftStore(ctx).AddShifts(ctx, c.Config.Name, tst.shifts); err != nil {
					t.Fatalf("%s: AddShifts(ctx, %q, _) failed: %v", tst.name, c.Config.Name, err)
				}
				defer h.shiftStore(ctx).DeleteAllShifts(ctx, c.Config.Name)
			}

			tst.ctx = clock.Set(tst.ctx, testclock.New(tst.time))

			var inPB apb.OncallRequest
			if err := proto.UnmarshalText(tst.in, &inPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			var wantPB apb.OncallResponse
			if err := proto.UnmarshalText(tst.want, &wantPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			res, err := h.Oncall(tst.ctx, &inPB)
			if got, want := (err != nil), tst.fail; got != want {
				t.Fatalf("%s: h.Oncall(ctx, _) = %t want: %t, err: %v", tst.name, got, want, err)
			}

			if err != nil {
				return
			}

			if diff := pretty.Compare(wantPB, res); diff != "" {
				t.Fatalf("%s: h.Oncall(ctx, _) differ -want +got: %s", tst.name, diff)
			}
		})
	}
}

func TestRPCList(t *testing.T) {
	ctx := newTestContext()

	tests := []struct {
		name    string
		fail    bool
		ctx     context.Context
		in      string
		cfgs    []*rotang.Configuration
		members []rotang.Member
		time    time.Time
		want    string
	}{{
		name: "Success",
		time: midnight,
		ctx:  ctx,
		cfgs: []*rotang.Configuration{
			{
				Config: rotang.Config{
					Name: "Test Rota",
					Shifts: rotang.ShiftConfig{
						Generator: "Fair",
						Shifts: []rotang.Shift{
							{
								Name: "MTV All Day",
							},
						},
					},
				},
				Members: []rotang.ShiftMember{
					{
						Email:     "mtv1@oncall.com",
						ShiftName: "MTV All Day",
					}, {
						Email:     "mtv2@oncall.com",
						ShiftName: "MTV All Day",
					},
				},
			},
		},
		members: []rotang.Member{
			{
				Email: "mtv1@oncall.com",
				Name:  "Mtv1 Mtvsson",
			}, {
				Email: "mtv2@oncall.com",
				Name:  "Mtv2 Mtvsson",
			},
		},
		want: `
		rotations: <
			name: "Test Rota"
    >
		`,
	}, {
		name: "No rotations",
		fail: true,
		ctx:  ctx,
	}}

	h := testSetup(t)

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			for _, m := range tst.members {
				if err := h.memberStore(ctx).CreateMember(ctx, &m); err != nil {
					t.Fatalf("%s: CreateMember(ctx, _) failed: %v", tst.name, err)
				}
				defer h.memberStore(ctx).DeleteMember(ctx, m.Email)
			}
			for _, c := range tst.cfgs {
				if err := h.configStore(ctx).CreateRotaConfig(ctx, c); err != nil {
					t.Fatalf("%s: CreateRotaconfig(ctx, _) failed: %v", tst.name, err)
				}
				defer h.configStore(ctx).DeleteRotaConfig(ctx, c.Config.Name)
			}

			tst.ctx = clock.Set(tst.ctx, testclock.New(tst.time))

			var inPB apb.ListRotationsRequest
			if err := proto.UnmarshalText(tst.in, &inPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			var wantPB apb.ListRotationsResponse
			if err := proto.UnmarshalText(tst.want, &wantPB); err != nil {
				t.Fatalf("%s: proto.UnmarshalText(_, _) failed: %v", tst.name, err)
			}

			res, err := h.ListRotations(tst.ctx, &inPB)
			if got, want := (err != nil), tst.fail; got != want {
				t.Fatalf("%s: h.Oncall(ctx, _) = %t want: %t, err: %v", tst.name, got, want, err)
			}

			if err != nil {
				return
			}

			if diff := pretty.Compare(wantPB, res); diff != "" {
				t.Fatalf("%s: h.Oncall(ctx, _) differ -want +got: %s", tst.name, diff)
			}
		})
	}
}
