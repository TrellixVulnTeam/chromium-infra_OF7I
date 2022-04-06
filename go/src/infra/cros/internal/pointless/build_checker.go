// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package pointless contains code for the pointless build checker.
package pointless

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"infra/cros/internal/gerrit"
	"infra/cros/internal/match"

	"github.com/golang/protobuf/ptypes/wrappers"
	testplans_pb "go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
)

// CheckBuilder assesses whether a child builder is pointless for a given CQ run. This may be the
// case if the commits in the CQ run don't affect any files that could possibly affect this
// builder's relevant paths.
func CheckBuilder(
	affectedFiles []string,
	relevantPaths []*testplans_pb.PointlessBuildCheckRequest_Path,
	ignoreKnownNonPortageDirectories bool,
	cfg *testplans_pb.BuildIrrelevanceCfg) (*testplans_pb.PointlessBuildCheckResponse, error) {

	if len(affectedFiles) == 0 {
		log.Printf("No affected files, so this run is irrelevant to the relevant paths")
		return &testplans_pb.PointlessBuildCheckResponse{
			PointlessBuildReason: testplans_pb.PointlessBuildCheckResponse_IRRELEVANT_TO_DEPS_GRAPH,
			BuildIsPointless:     &wrappers.BoolValue{Value: true},
		}, nil
	}

	// If the build affects forced relevantPaths we must consider it relevant (unless flagged not to).
	if ignoreKnownNonPortageDirectories {
		log.Printf("Ignoring RELEVANT_TO_KNOWN_NON_PORTAGE_DIRECTORIES check as requested by ignore_known_non_portage_directories option.")
	} else {
		hasAlwaysRelevantPaths := checkAlwaysRelevantPaths(affectedFiles, cfg)
		log.Printf("After considering the always relevant file paths, is the build pointless yet?: %t", hasAlwaysRelevantPaths)
		if hasAlwaysRelevantPaths {
			log.Printf("Since we know the build isn't pointless, we can return early")
			return &testplans_pb.PointlessBuildCheckResponse{
				PointlessBuildReason: testplans_pb.PointlessBuildCheckResponse_RELEVANT_TO_KNOWN_NON_PORTAGE_DIRECTORIES,
				BuildIsPointless:     &wrappers.BoolValue{Value: false},
			}, nil
		}
	}
	// Filter out files that are irrelevant to Portage because of the config.
	affectedFiles = filterByBuildIrrelevantPaths(affectedFiles, cfg)
	if len(affectedFiles) == 0 {
		log.Printf("All files ruled out by build-irrelevant paths. This means that " +
			"none of the Gerrit changes in the build input could affect the outcome of the build")
		return &testplans_pb.PointlessBuildCheckResponse{
			BuildIsPointless:     &wrappers.BoolValue{Value: true},
			PointlessBuildReason: testplans_pb.PointlessBuildCheckResponse_IRRELEVANT_TO_KNOWN_NON_PORTAGE_DIRECTORIES,
		}, nil
	}
	log.Printf("After considering build-irrelevant paths, we still must consider files:\n%v",
		strings.Join(affectedFiles, "\n"))

	// Filter out files that aren't in the relevant paths.
	affectedFiles = filterByPortageDeps(affectedFiles, relevantPaths)
	if len(affectedFiles) == 0 {
		log.Printf("All files ruled out after checking relevant paths")
		return &testplans_pb.PointlessBuildCheckResponse{
			BuildIsPointless:     &wrappers.BoolValue{Value: true},
			PointlessBuildReason: testplans_pb.PointlessBuildCheckResponse_IRRELEVANT_TO_DEPS_GRAPH,
		}, nil
	}

	log.Printf("This build is not pointless, due to files:\n%v",
		strings.Join(affectedFiles, "\n"))
	return &testplans_pb.PointlessBuildCheckResponse{
		BuildIsPointless: &wrappers.BoolValue{Value: false},
	}, nil
}

// Get all of the files referenced by each Gerrit Change in the build.
// File paths are prefixed by the source path for the Gerrit project as
// specified in the manifest.
func ExtractAffectedFiles(changes []*bbproto.GerritChange,
	changeRevs *gerrit.ChangeRevData, repoToSrcRoot map[string]map[string]string) ([]string, error) {
	allAffectedFiles := make([]string, 0)
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
			return nil, fmt.Errorf("Found no source mapping for project %s and branch %s", rev.Project, rev.Branch)
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

func filterByPortageDeps(files []string, relevantPaths []*testplans_pb.PointlessBuildCheckRequest_Path) []string {
	portageDeps := make([]string, 0)
	for _, path := range relevantPaths {
		portageDeps = append(portageDeps, path.Path)
	}
	log.Printf("Found %d affected files to consider:\n"+
		"<portage dep paths>\n%v\n</portage dep paths>",
		len(portageDeps), strings.Join(portageDeps, "\n"))

	portageFilteredFiles := make([]string, 0)
affectedFile:
	for _, f := range files {
		for _, pd := range portageDeps {
			if f == pd {
				log.Printf("Cannot ignore file %s due to Portage dependency %s", f, pd)
				portageFilteredFiles = append(portageFilteredFiles, f)
				continue affectedFile
			}
			pdAsDir := strings.TrimSuffix(pd, "/") + "/"
			if strings.HasPrefix(f, pdAsDir) {
				log.Printf("Cannot ignore file %s since it's in Portage dependency %s", f, pd)
				portageFilteredFiles = append(portageFilteredFiles, f)
				continue affectedFile
			}
		}
		log.Printf("Ignoring file %s because no prefix of it is referenced in the relevant paths", f)
	}
	return portageFilteredFiles
}

func checkAlwaysRelevantPaths(files []string, cfg *testplans_pb.BuildIrrelevanceCfg) bool {
	for _, f := range files {
		for _, pattern := range cfg.RelevantFilePatterns {
			match, err := match.FilePatternMatches(pattern, f)
			if err != nil {
				log.Fatalf("Failed to match pattern %s against file %s: %v", pattern, f, err)
			}
			if match {
				log.Printf("File %s matches %s, therefore we are not pointless", f, pattern.Pattern)
				return true
			}
		}
	}
	return false
}
