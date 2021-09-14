// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/hardcoded/chromeinfra"
	"log"
	"os"
)

var (
	// StdoutLog contains the stdout logger for this package.
	StdoutLog *log.Logger
	// StderrLog contains the stderr logger for this package.
	StderrLog *log.Logger
)

func getApplication(authOpts auth.Options) *subcommands.DefaultApplication {
	return &subcommands.DefaultApplication{
		Name:  "debug-symbols-uploader",
		Title: "upload debug symbols builder",
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			getCmdUploadDebugSymbols(authOpts),
			authcli.SubcommandInfo(authOpts, "auth-info", false),
			authcli.SubcommandLogin(authOpts, "auth-login", false),
			authcli.SubcommandLogout(authOpts, "auth-logout", false),
		},
	}
}

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

func SetUp(c uploadDebugSymbolsCommand, a subcommands.Application, args []string, env subcommands.Env) int {
	StdoutLog = a.(*uploadDebugSymbolsApplication).stdoutLog
	StderrLog = a.(*uploadDebugSymbolsApplication).stderrLog

	// Validate flags/arguments.
	if err := c.validate(); err != nil {
		LogErr(err.Error())
		return 1
	}
	return 0
}

type uploadDebugSymbolsApplication struct {
	*subcommands.DefaultApplication
	stdoutLog *log.Logger
	stderrLog *log.Logger
}

type uploadDebugSymbolsCommand interface {
	validate() error
}

func main() {
	opts := chromeinfra.DefaultAuthOptions()
	opts.Scopes = []string{
		gerrit.OAuthScope,
		auth.OAuthScopeEmail,
		"https://www.googleapis.com/auth/devstorage.full_control"}
	s := &uploadDebugSymbolsApplication{
		getApplication(opts),
		log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile),
		log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)}
	os.Exit(subcommands.Run(s, nil))
}
