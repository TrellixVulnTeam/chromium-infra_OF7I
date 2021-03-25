// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pinpoint

import (
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"
)

const legacyPrefix = "jobs/legacy-"

// LegacyJobName takes an ID (e.g. just a string of hex digits) and turns it
// into a fully-qualified job name compatible with the legacy pinpoint service.
//
// TODO(chowski): reuse this function throughout the codebase instead of
// continuing to hard-code the logic in multiple places.
func LegacyJobName(jobID string) string {
	if strings.HasPrefix(jobID, legacyPrefix) {
		return jobID
	}
	return legacyPrefix + jobID
}

var jobNameRe = regexp.MustCompile(`^jobs/legacy-(?P<id>[a-f0-9]+)$`)

// LegacyJobID is the inverse of LegacyJobName, turning a fully-qualified job
// name into a LegacyJobID.
func LegacyJobID(jobName string) (string, error) {
	// Ensure that the jobName suffix is a hex number.
	if !jobNameRe.MatchString(jobName) {
		return "", errors.Reason("invalid id format %q: must match %s", jobName, jobNameRe).Err()
	}
	matches := jobNameRe.FindStringSubmatch(jobName)
	legacyID := string(matches[jobNameRe.SubexpIndex("id")])
	if len(legacyID) == 0 {
		return "", errors.Reason("future ids not supported yet").Err()
	}
	return legacyID, nil
}
