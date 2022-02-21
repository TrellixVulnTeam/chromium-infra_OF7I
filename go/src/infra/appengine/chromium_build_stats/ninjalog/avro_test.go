// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ninjalog

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestAVROCodec(t *testing.T) {
	if _, err := avroCodec(); err != nil {
		t.Fatalf("failed to parse AVRO schema: %v", err)
	}
}

func TestToAVRO(t *testing.T) {
	outputTestCase := append([]Step{
		{
			Start:   76 * time.Millisecond,
			End:     187 * time.Millisecond,
			Out:     "resources/inspector/devtools_api.js",
			CmdHash: "75430546595be7c2",
		},
		{
			Start:   78 * time.Millisecond,
			End:     286 * time.Millisecond,
			Out:     "gen/angle/commit_id_2.py",
			CmdHash: "4ede38e2c1617d8c",
		},
		{
			Start:   78 * time.Millisecond,
			End:     286 * time.Millisecond,
			Out:     "gen/angle/commit_id_3.py",
			CmdHash: "4ede38e2c1617d8c",
		}}, stepsTestCase...)

	info := NinjaLog{
		Filename: ".ninja_log",
		Start:    1,
		Steps:    outputTestCase,
		Metadata: metadataTestCase,
	}

	createdTime := time.Unix(1514768400, 12345678)
	defer func() {
		timeNow = time.Now
	}()
	timeNow = func() time.Time { return createdTime }

	got := toAVRO(&info)
	if diff := cmp.Diff(map[string]interface{}{
		"build_configs": []map[string]interface{}{},
		"build_id":      int64(12345),
		"cpu_core":      int(0),
		"created_at":    createdTime,
		"jobs":          int(50),
		"os":            string("LINUX"),
		"step_name":     string("compile"),
		"targets":       []string{"all"},
		"log_entries": []map[string]interface{}{
			{
				"end_duration_sec":      float64(0.187),
				"outputs":               []string{"resources/inspector/devtools_api.js", "resources/inspector/devtools_extension_api.js"},
				"start_duration_sec":    float64(0.076),
				"weighted_duration_sec": float64(0.025783333),
			},
			{
				"end_duration_sec":      float64(0.286),
				"outputs":               []string{"gen/angle/commit_id.py", "gen/angle/commit_id_2.py", "gen/angle/commit_id_3.py"},
				"start_duration_sec":    float64(0.078),
				"weighted_duration_sec": float64(0.043683333),
			},
			{
				"end_duration_sec":      float64(0.287),
				"outputs":               []string{"gen/angle/copy_compiler_dll.bat"},
				"start_duration_sec":    float64(0.079),
				"weighted_duration_sec": float64(0.043516666),
			},
			{
				"end_duration_sec":      float64(0.284),
				"outputs":               []string{"gen/autofill_regex_constants.cc"},
				"start_duration_sec":    float64(0.08),
				"weighted_duration_sec": float64(0.04235),
			},
			{
				"end_duration_sec":      float64(0.287),
				"outputs":               []string{"PepperFlash/manifest.json"},
				"start_duration_sec":    float64(0.141),
				"weighted_duration_sec": float64(0.027933333),
			},
			{
				"end_duration_sec":      float64(0.288),
				"outputs":               []string{"PepperFlash/libpepflashplayer.so"},
				"start_duration_sec":    float64(0.142),
				"weighted_duration_sec": float64(0.028233333),
			},
			{
				"end_duration_sec":      float64(0.29),
				"outputs":               []string{"obj/third_party/angle/src/copy_scripts.actions_rules_copies.stamp"},
				"start_duration_sec":    float64(0.287),
				"weighted_duration_sec": float64(0.0025),
			},
		},
	}, got); diff != "" {
		t.Errorf("ToAVRO(%v) mismatch (-want +got):\n%s", info, diff)
	}
}
