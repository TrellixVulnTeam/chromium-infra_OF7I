// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package buildplan contains support code for the build planner.
package buildplan

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"infra/cros/internal/gerrit"
	"infra/cros/internal/match"

	"github.com/bmatcuk/doublestar"
	cros_pb "go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	testplans_pb "go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
)

var manifestFilePattern = testplans.FilePattern{Pattern: "{manifest,manifest-internal}/*.xml"}

// CheckBuildersInput is the input for a CheckBuilders call.
type CheckBuildersInput struct {
	Builders              []*cros_pb.BuilderConfig
	Changes               []*bbproto.GerritChange
	ChangeRevs            *gerrit.ChangeRevData
	RepoToBranchToSrcRoot map[string]map[string]string
	BuildIrrelevanceCfg   *testplans_pb.BuildIrrelevanceCfg
	SlimBuildCfg          *testplans_pb.SlimBuildCfg
	TestReqsCfg           *testplans_pb.TargetTestRequirementsCfg
	BuilderConfigs        *cros_pb.BuilderConfigs
}

// CheckBuilders determines which builders can be skipped and which must be run.
func (c *CheckBuildersInput) CheckBuilders() (*cros_pb.GenerateBuildPlanResponse, error) {

	response := &cros_pb.GenerateBuildPlanResponse{}

	// Get all of the files referenced by each GerritCommit in the Build.
	affectedFiles, err := extractAffectedFiles(c.Changes, c.ChangeRevs, c.RepoToBranchToSrcRoot)
	if err != nil {
		return nil, fmt.Errorf("error in extractAffectedFiles: %+v", err)
	}
	hasAffectedFiles := len(affectedFiles) > 0

	// TODO(crbug/1169870): Be more selective once we have a way to know what source paths are
	// relevant for each builder.
	hasXMLChange, err := hasManifestXMLChange(affectedFiles)
	if err != nil {
		return nil, fmt.Errorf("error in hasManifestXMLChange: %+v", err)
	}
	if hasXMLChange {
		log.Printf("Manifest change modifies XML file, running all children builds.")
		for _, b := range c.Builders {
			log.Printf("Must run builder %v", b.GetId().GetName())
			response.BuildsToRun = append(response.BuildsToRun, b.GetId())
		}
		return response, nil
	}

	ignoreImageBuilders := ignoreImageBuilders(affectedFiles, c.BuildIrrelevanceCfg, c.Changes)
	allowSlimBuilds := allowSlimBuilds(affectedFiles, c.SlimBuildCfg)

builderLoop:
	for _, b := range c.Builders {
		if (eligibleForGlobalIrrelevance(b) || !hasAffectedFiles) && ignoreImageBuilders {
			log.Printf("Ignoring %v because it's an image builder and the changes don't affect Portage", b.GetId().GetName())
			response.SkipForGlobalBuildIrrelevance = append(response.SkipForGlobalBuildIrrelevance, b.GetId())
			continue builderLoop
		}
		switch b.GetGeneral().GetRunWhen().GetMode() {
		case cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH:
			if hasAffectedFiles && ignoreByOnlyRunOnFileMatch(affectedFiles, b) {
				log.Printf("For %v, no files match OnlyRunOnFileMatch rules", b.GetId().GetName())
				response.SkipForRunWhenRules = append(response.SkipForRunWhenRules, b.GetId())
				continue builderLoop
			}
		case cros_pb.BuilderConfig_General_RunWhen_NO_RUN_ON_FILE_MATCH:
			if hasAffectedFiles && ignoreByNoRunOnFileMatch(affectedFiles, b) {
				log.Printf("For %v, all files match NoRunOnFileMatch rules", b.GetId().GetName())
				response.SkipForRunWhenRules = append(response.SkipForRunWhenRules, b.GetId())
				continue builderLoop
			}
		case cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, cros_pb.BuilderConfig_General_RunWhen_MODE_UNSPECIFIED:
			log.Printf("Builder %v has %v RunWhen mode", b.GetId().GetName(), b.GetGeneral().GetRunWhen().GetMode())
		}
		if allowSlimBuilds && eligibleForSlimBuild(b, c.TestReqsCfg) {
			slimB := getSlimBuilder(b.GetId().GetName(), c.BuilderConfigs)
			if slimB != nil {
				log.Printf("Must run builder %v", slimB.GetId().GetName())
				response.BuildsToRun = append(response.BuildsToRun, slimB.GetId())
				continue builderLoop
			}
		}
		log.Printf("Must run builder %v", b.GetId().GetName())
		response.BuildsToRun = append(response.BuildsToRun, b.GetId())
	}
	return response, nil
}

// Slim builds are only allows in select repos.
func allowSlimBuilds(affectedFiles []string, cfg *testplans_pb.SlimBuildCfg) bool {
	if len(affectedFiles) == 0 {
		log.Print("Cannot schedule slim builds since no affected files were provided")
		return false
	}
affectedFile:
	for _, f := range affectedFiles {
		for _, pattern := range cfg.SlimEligibleFilePatterns {
			match, err := match.FilePatternMatches(pattern, f)
			if err != nil {
			}
			if match {
				continue affectedFile
			}
		}
		log.Printf("Not all affected files match slim-eligible patterns")
		return false
	}
	log.Printf("All files matched slim-eligible patterns.")
	return true
}

// Given a builder name, returns the builder config for the slim variant if it exists.
func getSlimBuilder(b string, builderConfigs *cros_pb.BuilderConfigs) *cros_pb.BuilderConfig {
	suffixIndex := strings.LastIndex(b, "-")
	slimName := b[:suffixIndex] + "-slim" + b[suffixIndex:]
	for _, builderConfig := range builderConfigs.BuilderConfigs {
		if slimName == builderConfig.GetId().GetName() {
			return builderConfig
		}
	}
	return nil
}

// A CQ build target can be run as slim build if no HW or VM tests are configured for it.
func eligibleForSlimBuild(b *cros_pb.BuilderConfig, testReqsCfg *testplans_pb.TargetTestRequirementsCfg) bool {
	if b.GetId().GetType() != cros_pb.BuilderConfig_Id_CQ {
		return false
	}
	for _, targetTestReq := range testReqsCfg.PerTargetTestRequirements {
		if b.GetId().GetName() == targetTestReq.GetTargetCriteria().GetBuilderName() {
			return false
		}
	}
	return true
}

func eligibleForGlobalIrrelevance(b *cros_pb.BuilderConfig) bool {
	// As of 2020-07-08, the chromite builders are the only ones that should still trigger on matches
	// to global build irrelevance rules. The chromite builders just run unit tests; they don't build
	// the OS. If there are ever more such builders introduced, it would be much more useful to include
	// in the builderconfig something that indicates that property about the builder.
	if strings.HasPrefix(b.GetId().GetName(), "chromite-") {
		return false
	}
	return true
}

func ignoreImageBuilders(affectedFiles []string, cfg *testplans_pb.BuildIrrelevanceCfg, changes []*bbproto.GerritChange) bool {
	if len(changes) == 0 {
		// This happens during postsubmit runs, for example.
		log.Print("Cannot ignore image builders, since no changes were provided")
		return false
	}
	// Filter out files that are irrelevant to Portage because of the config.
	affectedFiles = filterByBuildIrrelevantPaths(affectedFiles, cfg)
	if len(affectedFiles) == 0 {
		log.Printf("All files ruled out by build-irrelevant paths for builder. " +
			"This means that none of the Gerrit changes in the build input could affect " +
			"the outcome of image builders")
		return true
	}
	log.Printf("After considering build-irrelevant paths, we still must consider "+
		"the following files for image builders:\n%v",
		strings.Join(affectedFiles, "\n"))
	return false
}

func ignoreByOnlyRunOnFileMatch(affectedFiles []string, b *cros_pb.BuilderConfig) bool {
	rw := b.GetGeneral().GetRunWhen()
	if rw.GetMode() != cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH {
		log.Printf("Can't apply OnlyRunOnFileMatch rule to %v, since it has mode %v", b.GetId().GetName(), rw.GetMode())
		return false
	}
	if len(rw.GetFilePatterns()) == 0 {
		log.Printf("Can't apply OnlyRunOnFileMatch rule to %v, since it has empty FilePatterns", b.GetId().GetName())
		return false
	}
	affectedFiles = findFilesMatchingPatterns(affectedFiles, b.GetGeneral().GetRunWhen().GetFilePatterns())
	if len(affectedFiles) == 0 {
		return true
	}
	log.Printf("After considering OnlyRunOnFileMatch rules, the following files require builder %v:\n%v",
		b.GetId().GetName(), strings.Join(affectedFiles, "\n"))
	return false
}

func ignoreByNoRunOnFileMatch(affectedFiles []string, b *cros_pb.BuilderConfig) bool {
	rw := b.GetGeneral().GetRunWhen()
	if rw.GetMode() != cros_pb.BuilderConfig_General_RunWhen_NO_RUN_ON_FILE_MATCH {
		log.Printf("Can't apply NoRunOnFileMatch rule to %v, since it has mode %v", b.GetId().GetName(), rw.GetMode())
		return false
	}
	if len(rw.GetFilePatterns()) == 0 {
		log.Printf("Can't apply OnlyRunOnFileMatch rule to %v, since it has empty FilePatterns", b.GetId().GetName())
		return false
	}
	matchedFiles := findFilesMatchingPatterns(affectedFiles, b.GetGeneral().GetRunWhen().GetFilePatterns())
	// If every file matched at least one pattern, we can ignore this builder.
	if len(affectedFiles) == len(matchedFiles) {
		return true
	}
	log.Printf("After considering NoRunOnFileMatch rules, the following files require builder %v:\n%v",
		b.GetId().GetName(), strings.Join(sliceDiff(affectedFiles, matchedFiles), "\n"))
	return false
}

func sliceDiff(a, b []string) []string {
	bm := make(map[string]bool)
	for _, be := range b {
		bm[be] = true
	}
	var diff []string
	for _, ae := range a {
		if !bm[ae] {
			diff = append(diff, ae)
		}
	}
	return diff
}

// stringInSlice returns a bool if a string exists in a slice.
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func hasManifestXMLChange(files []string) (bool, error) {
	for _, f := range files {
		match, err := match.FilePatternMatches(&manifestFilePattern, f)
		if err != nil {
			log.Fatalf("Failed to match pattern %s against file %s: %v", &manifestFilePattern, f, err)
		}
		if match {
			log.Printf("File %s matches pattern %s", f, &manifestFilePattern)
			return true, nil
		}
	}
	return false, nil
}

func extractAffectedFiles(changes []*bbproto.GerritChange, changeRevs *gerrit.ChangeRevData, repoToSrcRoot map[string]map[string]string) ([]string, error) {
	allAffectedFiles := make([]string, 0)
changeLoop:
	for _, gc := range changes {
		rev, err := changeRevs.GetChangeRev(gc.Host, gc.Change, int32(gc.Patchset))
		if err != nil {
			return nil, err
		}
		branchMapping, found := repoToSrcRoot[rev.Project]
		if !found {
			return nil, fmt.Errorf("Found no branch mapping for project %s", rev.Project)
		}
		srcRootMapping, found := branchMapping[rev.Branch]
		if !found {
			log.Printf("Found no source mapping for project %s and branch %s", rev.Project, rev.Branch)
			continue changeLoop
		}
		affectedFiles := make([]string, 0, len(rev.Files))
		for _, file := range rev.Files {
			fileSrcPath := fmt.Sprintf("%s/%s", srcRootMapping, file)
			affectedFiles = append(affectedFiles, fileSrcPath)
		}
		sort.Strings(affectedFiles)
		log.Printf("For https://%s/%d, affected files:\n%v\n\n",
			gc.Host, gc.Change, strings.Join(affectedFiles, "\n"))
		allAffectedFiles = append(allAffectedFiles, affectedFiles...)
	}
	sort.Strings(allAffectedFiles)
	log.Printf("All affected files:\n%v\n\n", strings.Join(allAffectedFiles, "\n"))
	return allAffectedFiles, nil
}

func filterByBuildIrrelevantPaths(files []string, cfg *testplans_pb.BuildIrrelevanceCfg) []string {
	pipFilteredFiles := make([]string, 0)
affectedFile:
	for _, f := range files {
		for _, pattern := range cfg.IrrelevantFilePatterns {
			match, err := match.FilePatternMatches(pattern, f)
			if err != nil {
				log.Fatalf("Failed to match pattern %s against file %s: %v", pattern, f, err)
			}
			if match {
				log.Printf("Ignoring file %s, since it matches Portage irrelevant pattern %s", f, pattern.Pattern)
				continue affectedFile
			}
		}
		log.Printf("Cannot ignore file %s by Portage irrelevant path rules", f)
		pipFilteredFiles = append(pipFilteredFiles, f)
	}
	return pipFilteredFiles
}

func findFilesMatchingPatterns(files []string, patterns []string) []string {
	matchedFiles := make([]string, 0)
affectedFile:
	for _, f := range files {
		for _, pattern := range patterns {
			match, err := doublestar.Match(pattern, f)
			if err != nil {
				log.Fatalf("Failed to match pattern %s against file %s: %v", pattern, f, err)
			}
			if match {
				log.Printf("File %s matches pattern %s", f, pattern)
				matchedFiles = append(matchedFiles, f)
				continue affectedFile
			}
		}
		log.Printf("File %s matches none of the patterns %v", f, patterns)
	}
	return matchedFiles
}
