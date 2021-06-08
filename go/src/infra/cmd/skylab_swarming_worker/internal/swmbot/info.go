// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package swmbot provides interaction with the Swarming bot running
// the Skylab worker process.  This includes information about the
// Swarming bot as well as any Swarming bot local state.
package swmbot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"infra/cmd/skylab_swarming_worker/internal/lucifer"
)

// Info contains information about the current Swarming bot.
type Info struct {
	AdminService     string
	AutotestPath     string
	BotDUTID         string
	InventoryService string
	UFSService       string
	LuciferBinDir    string
	ParserPath       string
	SwarmingService  string
	LabpackDir       string
	IsSchedulingUnit bool
	Task             Task
}

// GetInfo returns the Info for the current Swarming bot, built from
// environment variables.
//
// Per-bot variables:
//
//   ADMIN_SERVICE: Admin service host, e.g. foo.appspot.com.
//   INVENTORY_SERVICE: Inventory V2 service host, e.g. foo.appspot.com.
//   AUTOTEST_DIR: Path to the autotest checkout on server.
//   LUCIFER_TOOLS_DIR: Path to the lucifer installation.
//   PARSER_PATH: Path to the autotest_status_parser installation.
//   FLEET_RESOURCE_NAME: The name to locate a fleet resource, for now we
//                        use swarming dut_id dimension for this purpose.
//   SWARMING_SERVICE: Swarming service host, e.g. https://foo.appspot.com.
//   FLEET_MULTIDUTS_FLAG: Indicates if the bot is hosting a Scheduling Unit.
//
// Per-task variables:
//
//   SWARMING_TASK_ID: task id of the swarming task being serviced.
func GetInfo() *Info {
	info := &Info{
		AdminService:     os.Getenv("ADMIN_SERVICE"),
		AutotestPath:     os.Getenv("AUTOTEST_DIR"),
		BotDUTID:         os.Getenv("FLEET_RESOURCE_NAME"),
		InventoryService: os.Getenv("INVENTORY_SERVICE"),
		UFSService:       os.Getenv("UFS_SERVICE"),
		LuciferBinDir:    os.Getenv("LUCIFER_TOOLS_DIR"),
		ParserPath:       os.Getenv("PARSER_PATH"),
		SwarmingService:  os.Getenv("SWARMING_SERVICE"),
		LabpackDir:       os.Getenv("LABPACK_DIR"),
		Task: Task{
			RunID: os.Getenv("SWARMING_TASK_ID"),
		},
	}
	suFlag := os.Getenv("FLEET_MULTIDUTS_FLAG")
	if strings.ToLower(suFlag) == "true" || suFlag == "1" {
		info.IsSchedulingUnit = true
	}
	return info
}

// Task describes the bot's current task.
type Task struct {
	RunID string
}

// LuciferConfig returns the lucifer.Config for the Swarming bot.
func (b *Info) LuciferConfig() lucifer.Config {
	return lucifer.Config{
		AutotestPath: b.AutotestPath,
		LabpackDir:   b.LabpackDir,
		BinDir:       b.LuciferBinDir,
	}
}

// ResultsDir returns the path to the results directory used by the bot task.
func (b *Info) ResultsDir() string {
	// TODO(pprabhu): Reflect the requesting swarming server URL in the resultdir.
	// This will truly disambiguate results between different swarming servers.
	return filepath.Join(b.AutotestPath, "results", resultsSubdir(b.Task.RunID))
}

// TaskRunURL returns the URL for the current Swarming task execution.
func (b *Info) TaskRunURL() string {
	// TODO(ayatane): Remove this fallback once SWARMING_SERVICE is passed down here.
	if b.SwarmingService == "" {
		return fmt.Sprintf("https://chromeos-swarming.appspot.com/task?id=%s", b.Task.RunID)
	}
	return fmt.Sprintf("%s/task?id=%s", b.SwarmingService, b.Task.RunID)
}

// StainlessURL returns the URL to the stainless logs browser for logs offloaded
// from this task.
func (t *Task) StainlessURL() string {
	return fmt.Sprintf(
		"https://stainless.corp.google.com/browse/chromeos-autotest-results/%s/",
		resultsSubdir(t.RunID))
}

// GsURL returns the URL for the Google Storage location of the logs offloaded
// from this task.
func (t *Task) GsURL(gsBucket string) string {
	return fmt.Sprintf("gs://%s/%s/", gsBucket, resultsSubdir(t.RunID))
}

func resultsSubdir(runID string) string {
	return filepath.Join(fmt.Sprintf("swarming-%s0", runID[:len(runID)-1]), runID[len(runID)-1:])
}
