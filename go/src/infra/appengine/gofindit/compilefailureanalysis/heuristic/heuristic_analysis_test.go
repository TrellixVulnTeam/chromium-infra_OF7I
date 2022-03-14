// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/gae/impl/memory"

	"infra/appengine/gofindit/internal/buildbucket"
	"infra/appengine/gofindit/internal/logdog"
	"infra/appengine/gofindit/model"
)

func TestAnalyzeFailure(t *testing.T) {
	t.Parallel()
	c := memory.Use(context.Background())

	// Setup mock
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mc := buildbucket.NewMockedClient(c, ctl)
	c = mc.Ctx
	c = logdog.MockClientContext(c, map[string]string{
		"https://logs.chromium.org/logs/ninja_log":  "ninja_log",
		"https://logs.chromium.org/logs/stdout_log": "stdout_log",
	})
	res := &bbpb.Build{
		Steps: []*bbpb.Step{
			{
				Name: "compile",
				Logs: []*bbpb.Log{
					{
						Name:    "json.output[ninja_info]",
						ViewUrl: "https://logs.chromium.org/logs/ninja_log",
					},
					{
						Name:    "stdout",
						ViewUrl: "https://logs.chromium.org/logs/stdout_log",
					},
				},
			},
		},
	}
	mc.Client.EXPECT().GetBuild(gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil).AnyTimes()

	Convey("GetCompileLog", t, func() {
		ninjaLogJson := map[string]interface{}{
			"failures": []interface{}{
				map[string]interface{}{
					"dependencies": []string{"d1", "d2"},
					"output":       "/opt/s/w/ir/cache/goma/client/gomacc blah blah...",
					"output_nodes": []string{"n1", "n2"},
					"rule":         "CXX",
				},
			},
		}
		ninjaLogStr, err := json.Marshal(ninjaLogJson)
		So(err, ShouldBeNil)

		c = logdog.MockClientContext(c, map[string]string{
			"https://logs.chromium.org/logs/ninja_log":  string(ninjaLogStr),
			"https://logs.chromium.org/logs/stdout_log": "stdout_log",
		})
		logs, err := GetCompileLogs(c, 12345)
		So(err, ShouldBeNil)
		So(*logs, ShouldResemble, model.CompileLogs{
			NinjaLog: &model.NinjaLog{
				Failures: []*model.NinjaLogFailure{
					{
						Dependencies: []string{"d1", "d2"},
						Output:       "/opt/s/w/ir/cache/goma/client/gomacc blah blah...",
						OutputNodes:  []string{"n1", "n2"},
						Rule:         "CXX",
					},
				},
			},
			StdOutLog: "stdout_log",
		})
	})
	Convey("GetCompileLog failed", t, func() {
		c = logdog.MockClientContext(c, map[string]string{})
		_, err := GetCompileLogs(c, 12345)
		So(err, ShouldNotBeNil)
	})

}
