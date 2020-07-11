// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.package utils

package swarming

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"
)

func TestChangeDUTStateCommand(t *testing.T) {
	t.Parallel()
	testcases := []string{
		"test1",
		"something",
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			got := changeDUTStateCommand(tc)
			out := []string{
				"/bin/sh",
				"-c",
				"/opt/infra-tools/skylab_swarming_worker -task-name " + tc + "; echo Zzz...; do sleep 180",
			}
			if diff := cmp.Diff(out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

func TestReserveDUTRequest(t *testing.T) {
	t.Parallel()
	Convey("Verify deploy task request is correct formated", t, func() {
		tc := &TaskCreator{
			client:          nil,
			swarmingService: "https://chromium-swarm-dev.appspot.com/",
			session:         "session0",
		}
		r := tc.reserveDUTRequest("fake_dut_host")
		So(r.Name, ShouldEqual, "Reserve")
		So(r.TaskSlices, ShouldHaveLength, 1)
		command := strings.Join(r.TaskSlices[0].Properties.Command, " ")
		So(command, ShouldContainSubstring, "/bin/sh -c /opt/infra-tools/skylab_swarming_worker")
		So(command, ShouldContainSubstring, "-task-name set_reserved")
		for _, d := range r.TaskSlices[0].Properties.Dimensions {
			switch d.Key {
			case "pool":
				So(d.Value, ShouldEqual, "ChromeOSSkylab")
			case "dut_id":
				So(d.Value, ShouldEqual, "fake_dut_id")
			}
		}
		So("skylab-tool:Reserve", ShouldBeIn, r.Tags)
		So("admin_session:session0", ShouldBeIn, r.Tags)
		So("dut-name:fake_dut_host", ShouldBeIn, r.Tags)
		So("pool:ChromeOSSkylab", ShouldBeIn, r.Tags)
	})
}
func TestDUTNameToBotID(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in  string
		out string
	}{
		{
			"dut",
			"crossk-dut",
		},
		{
			"dut2.cros",
			"crossk-dut2",
		},
		{
			"crossk-dut3.cros",
			"crossk-dut3",
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := dutNameToBotID(tc.in)
			if diff := cmp.Diff(tc.out, got); diff != "" {
				t.Errorf("output mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

func TestSessionTag(t *testing.T) {
	t.Parallel()
	tc := &TaskCreator{
		session: "my_super_session",
	}
	got := tc.sessionTag()
	out := "admin_session:my_super_session"
	if diff := cmp.Diff(out, got); diff != "" {
		t.Errorf("output mismatch (-want +got): %s\n", diff)
	}
}

func TestCombineTags(t *testing.T) {
	t.Parallel()
	var combineTagsCases = []struct {
		name       string
		toolName   string
		logDogURL  string
		customTags []string
		out        []string
	}{
		{
			"test1",
			"tool1",
			"",
			nil,
			[]string{
				"skylab-tool:tool1",
				"luci_project:Env1",
				"pool:ChromeOSSkylab",
				"admin_session:session1",
			},
		},
		{
			"test2",
			"tool2",
			"log2",
			[]string{},
			[]string{
				"skylab-tool:tool2",
				"luci_project:Env1",
				"pool:ChromeOSSkylab",
				"admin_session:session1",
				"log_location:log2",
			},
		},
		{
			"test3",
			"tool3",
			"",
			[]string{
				"mytag:val3",
			},
			[]string{
				"skylab-tool:tool3",
				"luci_project:Env1",
				"pool:ChromeOSSkylab",
				"admin_session:session1",
				"mytag:val3",
			},
		},
		{
			"test4",
			"tool4",
			"log4",
			[]string{
				"mytag:val4",
			},
			[]string{
				"skylab-tool:tool4",
				"luci_project:Env1",
				"pool:ChromeOSSkylab",
				"admin_session:session1",
				"log_location:log4",
				"mytag:val4",
			},
		},
	}
	for _, tt := range combineTagsCases {
		t.Run(tt.name, func(t *testing.T) {
			tc := &TaskCreator{
				LUCIProject: "Env1",
				session:     "session1",
			}
			got := tc.combineTags(tt.toolName, tt.logDogURL, tt.customTags)
			if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("%s output mismatch (-want +got): %s\n", tt.name, diff)
			}
		})
	}
}
