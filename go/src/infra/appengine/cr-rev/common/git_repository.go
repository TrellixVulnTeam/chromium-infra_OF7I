package common

import (
	"infra/appengine/cr-rev/config"
	"strings"
)

// DefaultIncludeRefs is reference path that is indexed by default.
const DefaultIncludeRefs = "refs/heads"

// GitRepository uniquely identifies a repository and has reference to config, if
// present.
type GitRepository struct {
	// Host identifies GoB host, usually only subdomain portion of
	// googlesource.com. Example: chromium, pdfium
	Host string
	// Name is name of Git repository. Example: chromium/src
	Name string
	// Config is a snapshot of config.Repository. It may not be up-to-date.
	Config *config.Repository
}

// ShouldIndex returns true if given reference should be indexed.
func (r *GitRepository) ShouldIndex(ref string) bool {
	if r.Config == nil {
		return strings.HasPrefix(ref, DefaultIncludeRefs)
	}
	for _, excludedRef := range r.Config.GetExcludeRefs() {
		if strings.HasPrefix(ref, excludedRef) {
			return false
		}
	}

	if len(r.Config.GetRefs()) == 0 {
		return strings.HasPrefix(ref, DefaultIncludeRefs)
	}

	for _, includedRef := range r.Config.GetRefs() {
		if strings.HasPrefix(ref, includedRef) {
			return true
		}
	}
	return false
}
