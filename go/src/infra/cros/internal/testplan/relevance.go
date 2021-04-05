package testplan

import (
	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
	"regexp"
	"strings"

	"go.chromium.org/chromiumos/config/go/test/plan"
)

// matchesAnyPattern returns true if s matches any pattern in patterns.
func matchesAnyPattern(patterns []string, s string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, s)
		if err != nil {
			return false, err
		}

		if matched {
			return true, nil
		}
	}

	return false, nil
}

// metadataForFile finds the Metadata mapped closest to file.
//
// The directory in mapping that has the longest string match with file is
// chosen. Usually, this would be called with a mapping with form COMPUTED,
// so the returned Metadata has inherited metadata.
func metadataForFile(mapping *dirmd.Mapping, file string) *dirmdpb.Metadata {
	longestMatchingDir := ""
	for dir := range mapping.Dirs {
		if strings.Contains(file, dir) && len(dir) > len(longestMatchingDir) {
			longestMatchingDir = dir
		}
	}

	// If no directory matches file, use the root metadata in the mapping.
	if len(longestMatchingDir) == 0 {
		longestMatchingDir = "."
	}

	return mapping.Dirs[longestMatchingDir]
}

// relevantSourceTestPlans finds SourceTestPlans relevant to affectedFiles.
//
// SourceTestPlan has descriptions of how relevant file paths are determined.
// The paths in mapping and affectedFiles must have the same root.
func relevantSourceTestPlans(mapping *dirmd.Mapping, affectedFiles []string) ([]*plan.SourceTestPlan, error) {
	// Use a map to keep track of what plans have been added, so the same plan
	// isn't added twice. Accumulate plans in a slice, so the return order is
	// stable, based on the order of files and SourceTestPlans in mapping.
	plans := make([]*plan.SourceTestPlan, 0)
	addedPlans := make(map[*plan.SourceTestPlan]bool)

	for _, file := range affectedFiles {
		sourceTestPlans := metadataForFile(mapping, file).GetChromeos().GetCq().GetSourceTestPlans()

		for _, plan := range sourceTestPlans {
			// If a file matches any exclude regexp, it cannot make the plan
			// relevant.
			fileExcluded, err := matchesAnyPattern(plan.GetPathRegexpExcludes(), file)
			if err != nil {
				return nil, err
			}

			if fileExcluded {
				continue
			}

			// No regexps is treated as matching all files.
			fileIncluded := true
			if len(plan.GetPathRegexps()) > 0 {
				fileIncluded, err = matchesAnyPattern(plan.GetPathRegexps(), file)
				if err != nil {
					return nil, err
				}
			}

			if fileIncluded {
				if _, added := addedPlans[plan]; !added {
					addedPlans[plan] = true
					plans = append(plans, plan)
				}
			}
		}
	}

	return plans, nil
}
