// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"go.chromium.org/luci/common/cli"

	"github.com/maruel/subcommands"
)

// TODO(gregorynisbet): is a status.log file is the
// timestamp the start time or the end time? I think it's the start time
// but I need proof.

// TODO(gregorynisbet): handle errors in a smarter way.

const timestampPat = `timestamp=(\d+)`
const endPat = `^END.*`

// match leading prefix of line
const linePat = `^\s*(START|GOOD|END GOOD|FAIL|END FAIL)+\t----`

const sampleSwarmingTaskID = "4ff01b2173d08410"

// extractTimestamp takes an expression of the form
// timestamp=54 and extracts the integer part (in this case 54)
func extractTimestamp(line string) (int64, error) {
	r := regexp.MustCompile(timestampPat)
	m := r.FindStringSubmatch(line)
	if len(m) == 0 {
		return 0, fmt.Errorf("line does not contain timestamp")
	}
	t, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not an integer: %s", m[1], err)
	}
	return t, nil
}

// event is a raw event without an amount of time to the preceding event.
type event struct {
	timestamp int64
	level     int
	status    string
	name      string
	isEnd     bool
}

func (e event) Equal(o event) bool {
	return (e.timestamp == o.timestamp &&
		e.level == o.level &&
		e.status == o.status &&
		e.name == o.name &&
		e.isEnd == o.isEnd)
}

func (e *event) toString() string {
	return fmt.Sprintf("%d %d %q %q %t", e.timestamp, e.level, e.status, e.name, e.isEnd)
}

// normalizeIndent takes a list of fields and strips off leading empty fields
func normalizeIndent(fields []string) (int, []string) {
	for idx, item := range fields {
		if len(item) != 0 {
			return idx, fields[idx:]
		}
	}
	return len(fields), nil
}

// looksLikeEventLine takes an event line and discards it if
// the first few fields do not resemble an event line.
func looksLikeEventLine(s string) bool {
	r := regexp.MustCompile(linePat)
	return r.MatchString(s)
}

//  parseEvent takes a line of output from status.log (tab-delimited) and
//  converts it into an event.
//
//  sample parseable line (replace multiple consecutive spaces with tabs).
//  GOOD    ----    verify.PASS     timestamp=1605634967    localtime=Nov 17 17:42:47
//
func parseEvent(line string) (*event, error) {
	if !looksLikeEventLine(line) {
		return nil, fmt.Errorf("line %q does not appear to be an event line", line)
	}
	t, err := extractTimestamp(line)
	if err != nil {
		return nil, err
	}
	rawFields := strings.Split(line, "\t")
	level, fields := normalizeIndent(rawFields)
	if len(fields) <= 0 {
		return nil, fmt.Errorf("no status field")
	}
	status := fields[0]
	if len(status) == 0 {
		return nil, fmt.Errorf("status cannot be empty")
	}
	if len(fields) <= 2 {
		return nil, fmt.Errorf("no name field")
	}
	name := fields[2]
	if len(name) == 0 {
		return nil, fmt.Errorf("name cannot be empty")
	}
	r := regexp.MustCompile(endPat)
	isEnd := r.MatchString(status)
	return &event{
		t,
		level,
		status,
		name,
		isEnd,
	}, nil
}

// getStatusLogPaths takes a swarming ID and returns a list of paths
func getStatusLogPaths(taskID string) ([]string, error) {
	gspath := fmt.Sprintf("gs://chromeos-autotest-results/swarming-%s", taskID)
	gspath = fmt.Sprintf("%s/**/status.log", gspath)
	var b bytes.Buffer
	cmd := exec.Command("gsutil", "ls", gspath)
	cmd.Stdout = &b
	cmd.Run()
	s := b.String()
	lines := strings.Split(s, "\n")
	var nonEmptyLines []string
	for _, line := range lines {
		if len(line) > 0 {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}
	return nonEmptyLines, nil
}

// getEvents takes a swarming task id and returns a list of events
func getEvents(taskID string) ([]event, error) {
	paths, err := getStatusLogPaths(taskID)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths for swarming task %q", taskID)
	}
	if len(paths) >= 2 {
		return nil, fmt.Errorf("too many paths %d %v", len(paths), paths)
	}
	path := paths[0]
	var b bytes.Buffer
	cmd := exec.Command("gsutil", "cat", path)
	cmd.Stdout = &b
	cmd.Run()
	lines := strings.Split(b.String(), "\n")
	var events []event
	for i, line := range lines {
		if len(line) > 0 {
			event, err := parseEvent(line)
			if err != nil {
				// TODO(gregorynisbet): add logging
				fmt.Printf("line %d failed %q\n", i, line)
				continue
			}
			// TODO(gregorynisbet): this can fail
			events = append(events, *event)
		}
	}
	return events, nil
}

var dump = &subcommands.Command{
	UsageLine: "statuslog",
	ShortDesc: "statuslog events for task",
	LongDesc:  `statuslog events for admin tasks only`,
	CommandRun: func() subcommands.CommandRun {
		c := &dumpRun{}
		return c
	},
}

type dumpRun struct {
	subcommands.CommandRunBase
}

func (c *dumpRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *dumpRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	os, es := getEvents(sampleSwarmingTaskID)
	for i, o := range os {
		fmt.Fprintf(a.GetOut(), "#%d %s\n", i, o.toString())
	}
	fmt.Fprintf(a.GetErr(), "errors: %v\n", es)
	return nil
}

// getApplication produces the main application entity for eventdumper.
func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "eventdumper",
		Title: `Parse Logs for Admin Swarming Tasks`,
		Context: func(ctx context.Context) context.Context {
			return ctx
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			dump,
		},
	}
}

func main() {
	os.Exit(subcommands.Run(getApplication(), nil))
}
