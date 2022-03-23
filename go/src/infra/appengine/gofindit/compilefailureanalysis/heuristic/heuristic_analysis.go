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
	heuristicAnalysis := &gfim.CompileHeuristicAnalysis{
		ParentAnalysis: datastore.KeyForObj(c, cfa),
		StartTime:      clock.Now(c),
		Status:         gfipb.AnalysisStatus_CREATED,
	}

	if err := datastore.Put(c, heuristicAnalysis); err != nil {
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

	analysisResult, err := AnalyzeChangeLogs(c, signal, changelogs)
	if err != nil {
		return nil, fmt.Errorf("Error in justifying changelogs %w", err)
	}

	for _, item := range analysisResult.Items {
		logging.Infof(c, "Commit %s, with review URL %s, has score of %d", item.Commit, item.ReviewUrl, item.Justification.GetScore())
	}

	// Updates heuristic analysis
	if len(analysisResult.Items) > 0 {
		heuristicAnalysis.Status = gfipb.AnalysisStatus_FOUND
		err = saveResultsToDatastore(c, heuristicAnalysis, analysisResult, rr.LastPassed.Host, rr.LastPassed.Project, rr.LastPassed.Ref)
		if err != nil {
			return nil, fmt.Errorf("Failed to store result in datastore: %w", err)
		}
	} else {
		heuristicAnalysis.Status = gfipb.AnalysisStatus_NOTFOUND
	}

	heuristicAnalysis.EndTime = clock.Now(c)
	if err := datastore.Put(c, heuristicAnalysis); err != nil {
		return nil, fmt.Errorf("Failed to update heuristic analysis: %w", err)
	}

	return heuristicAnalysis, nil
}

func saveResultsToDatastore(c context.Context, analysis *gfim.CompileHeuristicAnalysis, result *gfim.HeuristicAnalysisResult, gitilesHost string, gitilesProject string, gitilesRef string) error {
	suspects := make([]*gfim.Suspect, len(result.Items))
	for i, item := range result.Items {
		suspect := &gfim.Suspect{
			ParentAnalysis: datastore.KeyForObj(c, analysis),
			ReviewUrl:      item.ReviewUrl,
			Score:          item.Justification.GetScore(),
			Justification:  item.Justification.GetReasons(),
			GitilesCommit: buildbucketpb.GitilesCommit{
				Host:    gitilesHost,
				Project: gitilesProject,
				Ref:     gitilesRef,
				Id:      item.Commit,
			},
		}
		suspects[i] = suspect
	}
	return datastore.Put(c, suspects)
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
