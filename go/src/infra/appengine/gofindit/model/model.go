// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// package model contains the datastore model for GoFindit.
package model

import (
	"time"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/gae/service/datastore"
)

type BuildFailureType string

const (
	BuildFailureType_Compile BuildFailureType = "Compile"
	BuildFailureType_Test    BuildFailureType = "Test"
	BuildFailureType_Infra   BuildFailureType = "Infra"
	BuildFailureType_Other   BuildFailureType = "Other"
)

type AnalysisStatus string

const (
	AnalysisStatus_Pending   AnalysisStatus = "Pending"
	AnalysisStatus_Running   AnalysisStatus = "Running"
	AnalysisStatus_Completed AnalysisStatus = "Completed"
	AnalysisStatus_Error     AnalysisStatus = "Error"
	AnalysisStatus_Skipped   AnalysisStatus = "Skipped"
)

type GitilesCommit struct {
	GitilesProject        string `gae:"gitiles_project"`
	GitilesHost           string `gae:"gitiles_host"`
	GitilesRef            string `gae:"gitiles_ref"`
	GitilesCommitID       string `gae:"gitiles_commit_id"`
	GitilesCommitPosition int    `gae:"gitiles_commit_position"`
}

// LuciBuild represents one LUCI build
type LuciBuild struct {
	BuildId     int64  `gae:"build_id"`
	Project     string `gae:"project"`
	Bucket      string `gae:"bucket"`
	Builder     string `gae:"builder"`
	BuildNumber int    `gae:"build_number"`
	GitilesCommit
	CreateTime time.Time            `gae:"create_time"`
	EndTime    time.Time            `gae:"end_time"`
	StartTime  time.Time            `gae:"start_time"`
	Status     buildbucketpb.Status `gae:"status"`
}

type LuciFailedBuild struct {
	// Id is the build Id
	Id int64 `gae:"$id"`
	LuciBuild
	// e.g. compile, test, infra...
	FailureType BuildFailureType `gae:"failure_type"`
}

// CompileFailure represents a compile failure in one or more targets.
type CompileFailure struct {
	Id int64 `gae:"$id"`
	// The key to LuciFailedBuild that the failure belongs to.
	Build *datastore.Key `gae:"$parent"`

	// The list of output targets that failed to compile
	OutputTargets []string `gae:"output_targets"`

	// Compile rule, e.g. ACTION, CXX, etc.
	// For chromium builds, it can be found in json.output[ninja_info] log of
	// compile step.
	// For chromeos builds, it can be found in an output property 'compile_failure'
	// of the build.
	Rule string `gae:"rule"`

	// Only for CC and CXX rules
	// These are the source files that this compile failure uses as input
	Dependencies []string `gae:"dependencies"`

	// Key to the CompileFailure that this failure merges into.
	// If this exists, no analysis on current failure, instead use the results
	// of merged_failure.
	MergedFailureKey *datastore.Key `gae:"merged_failure_key"`
}

// CompileFailureAnalysis is the analysis for CompileFailure.
// This stores information that is needed during the analysis, and also
// some metadata for the analysis.
type CompileFailureAnalysis struct {
	Id int64 `gae:"$id"`
	// Keys to the CompileFailure
	CompileFailures []*datastore.Key `gae:"compile_failures"`
	// Time when the analysis is created.
	CreateTime time.Time `gae:"create_time"`
	// Time when the analysis starts to run.
	StartTime time.Time `gae:"start_time"`
	// Time when the analysis runs to the end.
	EndTime time.Time `gae:"end_time"`
	// Status of the analysis
	Status AnalysisStatus `gae:"status"`
	// Id of the build in which the compile failures occurred the first time in
	// a sequence of consecutive failed builds.
	FirstFailedBuildId int64 `gae:"first_failed_build_id"`
	// Id of the latest build in which the failures did not happen.
	LastPassedBuildId int64 `gae:"last_passed_build_id"`
}

// CompileFailureInRerunBuild is a compile failure in a rerun build.
// Since we only need to keep a simple record on what's failed in rerun build,
// there is no need to reuse CompileFailure.
type CompileFailureInRerunBuild struct {
	// Json string of the failed output target
	// We store as json string instead of []string to avoid the "slice within
	// slice" error when saving to datastore
	OutputTargets string `gae:"output_targets"`
}

// CompileRerunBuild is one rerun build for CompileFailureAnalysis.
type CompileRerunBuild struct {
	// Id is the build Id.
	Id int64 `gae:"$id"`
	// Key to the parent CompileFailureAnalysis
	ParentAnalysis *datastore.Key `gae:"$parent"`
	LuciBuild
	// Failures occurring in the rerun build.
	Failures []CompileFailureInRerunBuild `gae:"failures"`
}

// Culprit is the culprit of rerun analysis.
type Culprit struct {
	// Key to the CompileFailureAnalysis that results in this culprit.
	ParentAnalysis *datastore.Key `gae:"parent"`
	GitilesCommit
}

// SuspectHint describes the reason why a CL is a suspect.
type SuspectHint struct {
	// A short, human-readable string that concisely describes a fact about the
	// suspect. e.g. 'add a/b/x.cc'
	Content string `gae:"content,noindex"`
	// Score is an integer from 0-100 representing the confidence in the suspect.
	// A higher score means a stronger signal that the suspect is responsible for
	// a failure.
	Score int `gae:"score,noindex"`
}

// Suspect is the suspect of heuristic analysis.
type Suspect struct {
	// Key to the CompileFailureAnalysis that results in this suspect.
	ParentAnalysis *datastore.Key `gae:"parent"`
	// SuspectHint describes the reason why a CL is a suspect.
	Hint SuspectHint `gae:"hint"`
	GitilesCommit
}
