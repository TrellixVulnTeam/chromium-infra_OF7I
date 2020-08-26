package common

import "infra/appengine/cr-rev/config"

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
