package pinpoint

// LegacyJobName takes the "short" ID (e.g. just a string of hex digits) and
// turns it into a fully-qualified job name compatible with the legacy pinpoint
// service.
//
// TODO(chowski): reuse this function throughout the codebase instead of
// continuing to hard-code the logic in multiple places.
func LegacyJobName(shortID string) string {
	return "jobs/legacy-" + shortID
}
