// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	"encoding/json"
	"fmt"
	"infra/appengine/gofindit/internal/buildbucket"
	"infra/appengine/gofindit/internal/gitiles"
	"infra/appengine/gofindit/internal/logdog"
	"infra/appengine/gofindit/model"
	gfim "infra/appengine/gofindit/model"
	gfipb "infra/appengine/gofindit/proto"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func Analyze(
	c context.Context,
	cfa *gfim.CompileFailureAnalysis,
	rr *gfipb.RegressionRange) (*gfim.CompileHeuristicAnalysis, error) {
	// Create a new HeuristicAnalysis Entity
	heuristic_analysis := &gfim.CompileHeuristicAnalysis{
		ParentAnalysis: datastore.KeyForObj(c, cfa),
		StartTime:      clock.Now(c),
		Status:         gfipb.AnalysisStatus_CREATED,
	}

	if err := datastore.Put(c, heuristic_analysis); err != nil {
		return nil, err
	}

	// Get changelogs for heuristic analysis
	changelogs, err := getChangeLogs(c, rr)
	if err != nil {
		return nil, fmt.Errorf("Failed getting changelogs %w", err)
	}
	logging.Infof(c, "Changelogs has %d logs", len(changelogs))

	// Gets compile logs from logdog
	// We need this to get the failure signals
	compileLogs, err := GetCompileLogs(c, cfa.FirstFailedBuildId)
	if err != nil {
		return nil, fmt.Errorf("Failed getting compile log: %w", err)
	}
	logging.Infof(c, "Compile log: %v", compileLogs)
	signal, err := ExtractSignals(c, compileLogs)
	if err != nil {
		return nil, fmt.Errorf("Error extracting signals %w", err)
	}

	justificationMap, err := AnalyzeChangeLogs(c, signal, changelogs)
	if err != nil {
		return nil, fmt.Errorf("Error in justifying changelogs %w", err)
	}
	for commit, justification := range justificationMap {
		logging.Infof(c, "Justification for commit %s", commit)
		logging.Infof(c, "Score: %d", justification.GetScore())
		logging.Infof(c, "Reasons: %s", justification.GetReasons())
	}
	return heuristic_analysis, nil
}

// getChangeLogs queries Gitiles for changelogs in the regression range
func getChangeLogs(c context.Context, rr *gfipb.RegressionRange) ([]*model.ChangeLog, error) {
	if rr.LastPassed.Host != rr.FirstFailed.Host || rr.LastPassed.Project != rr.FirstFailed.Project {
		return nil, fmt.Errorf("RepoURL for last pass and first failed commits must be same, but aren't: %v and %v", rr.LastPassed, rr.FirstFailed)
	}
	repoUrl := gitiles.GetRepoUrl(c, rr.LastPassed)
	return gitiles.GetChangeLogs(c, repoUrl, rr.LastPassed.Id, rr.FirstFailed.Id)
}

// GetCompileLogs gets the compile log for a build bucket build
// Returns the ninja log and stdout log
func GetCompileLogs(c context.Context, bbid int64) (*model.CompileLogs, error) {
	build, err := buildbucket.GetBuild(c, bbid, &buildbucketpb.BuildMask{
		Fields: &fieldmaskpb.FieldMask{
			Paths: []string{"steps"},
		},
	})
	if err != nil {
		return nil, err
	}
	ninjaUrl := ""
	stdoutUrl := ""
	for _, step := range build.Steps {
		if step.Name == "compile" {
			for _, log := range step.Logs {
				if log.Name == "json.output[ninja_info]" {
					ninjaUrl = log.ViewUrl
				}
				if log.Name == "stdout" {
					stdoutUrl = log.ViewUrl
				}
			}
			break
		}
	}

	ninjaLog := &model.NinjaLog{}
	stdoutLog := ""

	// TODO(crbug.com/1295566): Parallelize downloading ninja & stdout logs
	if ninjaUrl != "" {
		log, err := logdog.GetLogFromViewUrl(c, ninjaUrl)
		if err != nil {
			logging.Errorf(c, "Failed to get ninja log: %v", err)
		}
		if err = json.Unmarshal([]byte(log), ninjaLog); err != nil {
			return nil, fmt.Errorf("Failed to unmarshal ninja log %w. Log: %s", err, log)
		}
	}

	if stdoutUrl != "" {
		stdoutLog, err = logdog.GetLogFromViewUrl(c, stdoutUrl)
		if err != nil {
			logging.Errorf(c, "Failed to get stdout log: %v", err)
		}
	}

	if len(ninjaLog.Failures) > 0 || stdoutLog != "" {
		return &gfim.CompileLogs{
			NinjaLog:  ninjaLog,
			StdOutLog: stdoutLog,
		}, nil
	}

	return nil, fmt.Errorf("Could not get compile log from build %d", bbid)
}
