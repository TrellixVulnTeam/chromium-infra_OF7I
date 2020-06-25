package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"infra/appengine/rotang"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.chromium.org/gae/service/memcache"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HandleLegacy serves the /legacy endpoint.
func (h *State) HandleLegacy(ctx *router.Context) {
	if err := ctx.Context.Err(); err != nil {
		http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	name := ctx.Params.ByName("name")

	vf, ok := h.legacyMap[name]
	if !ok {
		http.Error(ctx.Writer, "not found", http.StatusNotFound)
		return
	}

	item := memcache.NewItem(ctx.Context, name)
	doCORS(ctx)
	if err := memcache.Get(ctx.Context, item); err != nil {
		logging.Warningf(ctx.Context, "%q not in the cache", name)
		val, err := vf(ctx, name)
		if err != nil {
			http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(ctx.Writer, val)
		return
	}
	fmt.Fprint(ctx.Writer, string(item.Value()))
}

func doCORS(ctx *router.Context) {
	ctx.Writer.Header().Add("Access-Control-Allow-Origin", "*")
}

const (
	trooperCal   = "google.com_3aov6uidfjscpj2hrpsd8i4e7o@group.calendar.google.com"
	matchSummary = "CCI-Trooper:"
	trooperShift = "Legacy Trooper"
	trooperRota  = "troopers"
	cciRota      = "CCI-Trooper"
)

type trooperJSON struct {
	Primary   string   `json:"primary"`
	Secondary []string `json:"secondaries"`
	UnixTS    int64    `json:"updated_unix_timestamp"`
}

func (h *State) legacyTrooper(ctx *router.Context, file string) (string, error) {
	updated := clock.Now(ctx.Context)
	shift, err := h.shiftStore(ctx.Context).Oncall(ctx.Context, clock.Now(ctx.Context), cciRota)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return "", err
		}
		shift = &rotang.ShiftEntry{}
	}

	var oncallers []string
	for _, o := range shift.OnCall {
		oncallers = append(oncallers, strings.Split(o.Email, "@")[0])
	}

	switch file {
	case "trooper.json":
		primary := "None"
		secondary := make([]string, 0)
		if len(oncallers) > 0 {
			primary = oncallers[0]
			if len(oncallers) > 1 {
				secondary = oncallers[1:]
			}
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(&trooperJSON{
			Primary:   primary,
			Secondary: secondary,
			UnixTS:    updated.Unix(),
		}); err != nil {
			return "", err
		}
		return buf.String(), nil
	case "current_trooper.txt":
		if len(oncallers) == 0 {
			return "None", nil
		}
		return strings.Join(oncallers, ","), nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "legacyTrooper only handles `trooper.json` and `current_trooper.txt`")
	}
}

var fileToRota = map[string]string{
	"sheriff_gpu.js": "Chrome GPU Pixel Wrangling",
	"sheriff_ios.js": "Chrome iOS Build Sheriff",

	"sheriff_perf.json":           "Chromium Perf Regression Sheriff Rotation",
	"sheriff_gpu.json":            "Chrome GPU Pixel Wrangling",
	"sheriff_angle.json":          "The ANGLE Wrangle",
	"sheriff_ios.json":            "Chrome iOS Build Sheriff",
	"sheriff_perfbot.json":        "Chromium Perf Bot Sheriff Rotation",
	"sheriff_flutter_engine.json": "Flutter Engine Rotation",
}

const week = 7 * 24 * time.Hour

type sheriffJSON struct {
	UnixTS int64    `json:"updated_unix_timestamp"`
	Emails []string `json:"emails"`
}

// legacySheriff produces the legacy cron created sheriff oncall files.
func (h *State) legacySheriff(ctx *router.Context, file string) (string, error) {
	rota, ok := fileToRota[file]
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "file: %q not handled by legacySheriff", file)
	}
	r, err := h.configStore(ctx.Context).RotaConfig(ctx.Context, rota)
	if err != nil {
		return "", err
	}
	if len(r) != 1 {
		return "", status.Errorf(codes.Internal, "RotaConfig did not return 1 configuration")
	}
	cfg := r[0]

	cal := h.legacyCalendar
	if cfg.Config.Enabled {
		cal = h.calendar
	}

	updated := clock.Now(ctx.Context)
	events, err := cal.Events(ctx, cfg, updated.Add(-week), updated.Add(week))
	if err != nil {
		return "", err
	}

	var entry rotang.ShiftEntry
	for _, e := range events {
		if (updated.After(e.StartTime) || updated.Equal(e.StartTime)) &&
			updated.Before(e.EndTime) {
			entry = e
		}
	}

	sp := strings.Split(file, ".")
	if len(sp) != 2 {
		return "", status.Errorf(codes.InvalidArgument, "filename in wrong format")
	}

	switch sp[1] {
	case "js":
		var oc []string
		for _, o := range entry.OnCall {
			oc = append(oc, strings.Split(o.Email, "@")[0])
		}
		str := "None"
		if len(oc) > 0 {
			str = strings.Join(oc, ", ")
		}
		return "document.write('" + str + "');", nil
	case "json":
		// This makes the JSON encoder produce `[]` instead of `null`
		// for empty lists.
		oc := make([]string, 0)
		for _, o := range entry.OnCall {
			oc = append(oc, o.Email)
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(&sheriffJSON{
			UnixTS: updated.Unix(),
			Emails: oc,
		}); err != nil {
			return "", err
		}
		return buf.String(), nil

	default:
		return "", status.Errorf(codes.InvalidArgument, "filename in wrong format")
	}
}

// getCurrentOncall returns the email address of the oncaller for the given rotation name
// at the given time, or "" if no-one is oncall.
func (h *State) getCurrentOncall(ctx *router.Context, name string, at time.Time) (string, error) {
	ss := h.shiftStore(ctx.Context)
	s, err := ss.Oncall(ctx.Context, at, name)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", err
	}
	if len(s.OnCall) == 0 {
		return "", nil
	}
	return s.OnCall[0].Email, nil
}

var chromeBuildSheriffRotations = []string{
	"Chrome Build Sheriff AMER-EAST",
	"Chrome Build Sheriff AMER-WEST",
	"Chrome Build Sheriff APAC",
	"Chrome Build Sheriff EMEA",
}

var clankBuildSheriffRotations = []string{
	"Clank Build Sheriff AMER",
	"Clank Build Sheriff EMEA",
}

func (h *State) getExternalSheriffs(ctx *router.Context, sheriffRotations []string) (*sheriffJSON, error) {
	now := clock.Now(ctx.Context)
	emails := make([]string, 0, len(sheriffRotations))
	for _, name := range sheriffRotations {
		email, err := h.getCurrentOncall(ctx, name, now)
		if err != nil {
			return nil, err
		}
		if email != "" {
			emails = append(emails, email)
		}
	}
	return &sheriffJSON{
		UnixTS: now.Unix(),
		Emails: emails,
	}, nil
}

// sheriffJSONFromExternal produces a sheriff.json file containing sheriffs sourced from external calendar events.
func (h *State) sheriffJSONFromExternal(ctx *router.Context, file string, sheriffRotations []string) (string, error) {
	sp := strings.Split(file, ".")
	if len(sp) != 2 {
		return "", status.Errorf(codes.InvalidArgument, "filename in wrong format")
	}

	sheriffs, err := h.getExternalSheriffs(ctx, sheriffRotations)
	if err != nil {
		return "", status.Errorf(codes.Internal, "Unable to fetch sheriffs list")
	}

	switch sp[1] {
	case "js":
		var usernames []string
		for _, email := range sheriffs.Emails {
			usernames = append(usernames, strings.Split(email, "@")[0])
		}
		str := "None"
		if len(usernames) > 0 {
			str = strings.Join(usernames, ", ")
		}
		return "document.write('" + str + "');", nil
	case "json":
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(sheriffs); err != nil {
			return "", err
		}
		return buf.String(), nil

	default:
		return "", status.Errorf(codes.InvalidArgument, "filename in wrong format")
	}
}

func (h *State) buildSheriff(ctx *router.Context, file string) (string, error) {
	return h.sheriffJSONFromExternal(ctx, file, chromeBuildSheriffRotations)
}

func (h *State) androidSheriff(ctx *router.Context, file string) (string, error) {
	return h.sheriffJSONFromExternal(ctx, file, clankBuildSheriffRotations)
}

var rotaToName = map[string]string{
	"angle":             "The ANGLE Wrangle",
	"ios_internal_roll": "Bling Piper Roll",
	"blink_bug_triage":  "Blink Bug Triage",
	"android_stability": "Clank Stability Sheriff (go/clankss)",
	"ecosystem_infra":   "Ecosystem Infra rotation",
	"gitadmin":          "Chrome Infra Git Admin Rotation",
	"gpu":               "Chrome GPU Pixel Wrangling",
	"headless_roll":     "Headless Chrome roll sheriff",
	"infra_platform":    "Chops Foundation Triage",
	"infra_triage":      "Chrome Infra Bug Triage Rotation",
	"monorail":          "Chrome Infra Monorail Triage Rotation",
	"network":           "Chrome Network Bug Triage",
	"perfbot":           "Chromium Perf Bot Sheriff Rotation",
	"ios":               "Chrome iOS Build Sheriff",
	"perf":              "Chromium Perf Regression Sheriff Rotation",
	"stability":         "Chromium Stability Sheriff",
	"v8_infra_triage":   "V8 Infra Bug Triage Rotation",
	"webview_bugcop":    "WebView Bug Cop",
	"flutter_engine":    "Flutter Engine Rotation",
	"webgl_bug_triage":  "WebGL Bug Triage",
}

const (
	fullDay   = 24 * time.Hour
	timeDelta = 90 * fullDay
)

type allRotations struct {
	Rotations []string   `json:"rotations"`
	Calendar  []dayEntry `json:"calendar"`
}

type dayEntry struct {
	Date         string     `json:"date"`
	Participants [][]string `json:"participants"`
}

func (h *State) legacyAllRotations(ctx *router.Context, _ string) (string, error) {
	start := clock.Now(ctx.Context).In(mtvTime)
	end := start.Add(timeDelta)
	cs := h.configStore(ctx.Context)

	var res allRotations
	dateMap := make(map[string]map[string][]string)

	// The Sheriff rotations.
	for k, v := range rotaToName {
		rs, err := cs.RotaConfig(ctx.Context, v)
		if err != nil {
			logging.Errorf(ctx.Context, "Getting configuration for: %q failed: %v", v, err)
			continue
		}
		if len(rs) != 1 {
			return "", status.Errorf(codes.Internal, "RotaConfig did not return 1 configuration")
		}
		cfg := rs[0]
		cal := h.legacyCalendar
		if cfg.Config.Enabled {
			cal = h.calendar
		}
		shifts, err := cal.Events(ctx, cfg, start, end)
		if err != nil {
			logging.Errorf(ctx.Context, "Fetching calendar events for: %q failed: %v", v[0], err)
			continue
		}
		res.Rotations = append(res.Rotations, k)
		buildLegacyRotation(dateMap, k, shifts)
	}
	// Troopers rotation.
	ts, err := h.calendar.TrooperShifts(ctx, trooperCal, matchSummary, trooperShift, start, end)
	if err != nil {
		logging.Errorf(ctx.Context, "Fetching calendar events for: Troopers failed: %v", err)
	}
	buildLegacyRotation(dateMap, trooperRota, ts)
	res.Rotations = append(res.Rotations, trooperRota)

	for k, v := range dateMap {
		entry := dayEntry{
			Date: k,
		}
		for _, r := range res.Rotations {
			p, ok := v[r]
			// When JSON encoding slices creating a slice with.
			// var bla []slice -> Produces `null` in the json output.
			// If instead creating the slice with.
			// make([]string,0) -> Will produce `[]`.
			if !ok || len(p) == 0 {
				p = make([]string, 0)
			}
			entry.Participants = append(entry.Participants, p)
		}
		res.Calendar = append(res.Calendar, entry)
	}

	sort.Slice(res.Calendar, func(i, j int) bool {
		return res.Calendar[i].Date < res.Calendar[j].Date
	})

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(res); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func buildLegacyRotation(dateMap map[string]map[string][]string, rota string, shifts []rotang.ShiftEntry) {
	if len(shifts) < 1 {
		return
	}
	for _, s := range shifts {
		// Truncade to fulldays for go/chromecals
		date := time.Date(s.StartTime.Year(), s.StartTime.Month(), s.StartTime.Day(), 0, 0, 0, 0, time.UTC)
		oc := make([]string, 0)
		for _, o := range s.OnCall {
			oc = append(oc, strings.Split(o.Email, "@")[0])
		}
		for ; ; date = date.Add(fullDay) {
			s.StartTime = time.Date(s.StartTime.Year(), s.StartTime.Month(), s.StartTime.Day(), 0, 0, 0, 0, time.UTC)
			s.EndTime = time.Date(s.EndTime.Year(), s.EndTime.Month(), s.EndTime.Day(), 0, 0, 0, 0, time.UTC)
			if !((date.After(s.StartTime) || date.Equal(s.StartTime)) && date.Before(s.EndTime)) {
				break
			}
			if _, ok := dateMap[date.Format(elementTimeFormat)]; !ok {
				dateMap[date.Format(elementTimeFormat)] = make(map[string][]string)
			}
			dateMap[date.Format(elementTimeFormat)][rota] = append(dateMap[date.Format(elementTimeFormat)][rota], oc...)
		}
	}
}
