// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package worker

import (
	"reflect"
	"testing"
	"time"

	"go.chromium.org/luci/common/flag"
)

func TestCommand_Args_with_path(t *testing.T) {
	t.Parallel()
	c := Command{Path: "/tmp/skylab_swarming_worker", TaskName: "admin_repair"}
	got := c.Args()
	want := []string{"/tmp/skylab_swarming_worker", "-task-name", "admin_repair"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("c.Args() = %#v; want %#v", got, want)
	}
}

func TestCommand_Args_default_path(t *testing.T) {
	t.Parallel()
	c := Command{TaskName: "admin_repair"}
	got := c.Args()
	want := []string{DefaultPath, "-task-name", "admin_repair"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("c.Args() = %#v; want %#v", got, want)
	}
}

func TestCommand_Args_isolated_output(t *testing.T) {
	t.Parallel()
	c := Command{TaskName: "name", OutputToIsolate: true}
	got := c.Args()
	want := []string{DefaultPath, "-isolated-outdir", "${ISOLATED_OUTDIR}", "-task-name", "name"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("c.Args() = %#v; want %#v", got, want)
	}
}

type basicEnv struct {
	luciProject string
	logDogHost  string
	logPrefix   string
}

func (e basicEnv) LUCIProject() string {
	return e.luciProject
}

func (e basicEnv) LogDogHost() string {
	return e.logDogHost
}

func (e basicEnv) GenerateLogPrefix() string {
	return e.logPrefix
}

func TestEnv(t *testing.T) {
	t.Parallel()
	const service = "sirius.appspot.com"
	e := basicEnv{
		logDogHost:  "luci-logdog.appspot.com",
		luciProject: "chromeos",
		logPrefix:   "skylab/83e6fa19-2cb0-4cc2-88b6-fb217a6cbb23",
	}
	var c Command
	c.Config(e)
	const wantURL = "logdog://luci-logdog.appspot.com/chromeos/skylab/83e6fa19-2cb0-4cc2-88b6-fb217a6cbb23/+/annotations"
	if c.LogDogAnnotationURL != wantURL {
		t.Errorf("LogdogAnnotationURL = %#v; want %#v", c.LogDogAnnotationURL, wantURL)
	}
}

func TestStipTime(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Tag  string
		Time time.Time
	}{
		{
			Tag:  "riff on default format time",
			Time: time.Date(2006, time.January, 2, 15, 4, 5, 99, time.UTC),
		},
		{
			Tag:  "typical example from stiptime",
			Time: time.Date(2015, time.June, 30, 18, 50, 50, 0, time.UTC),
		},
		{
			Tag:  "non-UTC time",
			Time: time.Date(2015, time.June, 30, 18, 50, 50, 0, time.FixedZone("custom", 3600)),
		},
	}

	for _, c := range cases {
		t.Run(c.Tag, func(t *testing.T) {
			t.Parallel()
			s := stiptime(c.Time)
			var got time.Time
			f := flag.Time(&got)
			if err := f.Set(s); err != nil {
				t.Fatalf("Error parsing back stiptime(%#v) => %s, %s", c.Time, s, err)
			}
			// Roundtrip converts to UTC timezone, but should continue to
			// represent the same terrestrial time.
			if c.Time.Sub(got) != 0 {
				t.Errorf("Incorrectly parsed stiptime(%#v) => %s, got %#v", c.Time, s, got)
			}
		})
	}
}
