// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"log"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

var (
	// StdoutLog contains the stdout logger for this package.
	StdoutLog *log.Logger
	// StderrLog contains the stderr logger for this package.
	StderrLog *log.Logger
)

// LogOut logs to stdout.
func LogOut(format string, a ...interface{}) {
	if StdoutLog != nil {
		StdoutLog.Printf(format, a...)
	}
}

// LogErr logs to stderr.
func LogErr(format string, a ...interface{}) {
	if StderrLog != nil {
		StderrLog.Printf(format, a...)
	}
}

type manifestDoctorCommand interface {
	validate() error
}

func SetUp(c manifestDoctorCommand, a subcommands.Application, args []string, env subcommands.Env) int {
	StdoutLog = a.(*manifestDoctorApplication).stdoutLog
	StderrLog = a.(*manifestDoctorApplication).stderrLog

	// Validate flags/arguments.
	if err := c.validate(); err != nil {
		LogErr(err.Error())
		return 1
	}

	return 0
}

// GetApplication returns an instance of the application.
func GetApplication(authOpts auth.Options) *subcommands.DefaultApplication {
	return &subcommands.DefaultApplication{
		Name: "manifest_doctor",
		Commands: []*subcommands.Command{
			authcli.SubcommandInfo(authOpts, "auth-info", false),
			authcli.SubcommandLogin(authOpts, "auth-login", false),
			authcli.SubcommandLogout(authOpts, "auth-logout", false),
			cmdLocalManifestBrancher(authOpts),
			cmdProjectBuildspec(authOpts),
		},
	}
}

type manifestDoctorApplication struct {
	*subcommands.DefaultApplication
	stdoutLog *log.Logger
	stderrLog *log.Logger
}

func main() {
	opts := chromeinfra.DefaultAuthOptions()
	scopes := []string{
		gerrit.OAuthScope,
		auth.OAuthScopeEmail,
		"https://www.googleapis.com/auth/datastore",
	}
	scopes = append(scopes, gs.ReadWriteScopes...)
	opts.Scopes = scopes
	s := &manifestDoctorApplication{
		GetApplication(opts),
		log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
		log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)}
	os.Exit(subcommands.Run(s, nil))
}
