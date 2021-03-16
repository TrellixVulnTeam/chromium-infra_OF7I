package pinpoint

import (
	"regexp"

	"go.chromium.org/luci/common/errors"
)

// LegacyJobName takes the "short" ID (e.g. just a string of hex digits) and
// turns it into a fully-qualified job name compatible with the legacy pinpoint
// service.
//
// TODO(chowski): reuse this function throughout the codebase instead of
// continuing to hard-code the logic in multiple places.
func LegacyJobName(shortID string) string {
	return "jobs/legacy-" + shortID
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
