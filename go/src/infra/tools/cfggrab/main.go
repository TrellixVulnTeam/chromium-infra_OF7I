// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Command cfggrab fetches some <name>.cfg from all LUCI project configs.
//
// By default prints all fetched configs to stdout, prefixing each printed
// line with the project name for easier grepping.
//
// If `-output-dir` is given, stores the fetched files into per-project
// <output-dir>/<project>.cfg.
//
// Usage:
//   luci-auth login
//   cfggrab cr-buildbucket.cfg | grep "service_account"
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	"go.chromium.org/luci/config/impl/remote"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

var outputDir = flag.String("output-dir", "-", "Where to store fetched files or - to print them to stdout for grepping")
var configService = flag.String("config-service-host", chromeinfra.ConfigServiceHost, "Hostname of LUCI Config service to query")

var stdoutLock sync.Mutex

func main() {
	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)

	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Expecting one positional argument with the name of the file to fetch.\n")
		os.Exit(2)
	}

	if err := run(ctx, flag.Arg(0), *outputDir); err != nil {
		errors.Log(ctx, err)
		os.Exit(1)
	}
}

func withConfigClient(ctx context.Context) (context.Context, error) {
	auth := auth.NewAuthenticator(ctx, auth.SilentLogin, chromeinfra.DefaultAuthOptions())
	client, err := auth.Client()
	if err != nil {
		return nil, err
	}
	return cfgclient.Use(ctx, remote.New(*configService, false, func(context.Context) (*http.Client, error) {
		return client, nil
	})), nil
}

func run(ctx context.Context, path, output string) error {
	if output != "-" {
		if err := os.MkdirAll(output, 0777); err != nil {
			return err
		}
	}

	ctx, err := withConfigClient(ctx)
	if err != nil {
		return err
	}

	projects, err := cfgclient.ProjectsWithConfig(ctx, path)
	if err != nil {
		return err
	}

	return parallel.FanOutIn(func(work chan<- func() error) {
		for _, proj := range projects {
			proj := proj
			work <- func() error {
				if err := processProject(ctx, proj, path, output); err != nil {
					logging.Errorf(ctx, "Failed when processing %s: %s", proj, err)
					return err
				}
				return nil
			}
		}
	})
}

func processProject(ctx context.Context, proj, path, output string) error {
	var blob []byte
	err := cfgclient.Get(ctx, config.Set("projects/"+proj), path, cfgclient.Bytes(&blob), nil)
	if err != nil {
		return err
	}

	if output != "-" {
		return ioutil.WriteFile(filepath.Join(output, proj+".cfg"), blob, 0666)
	}

	stdoutLock.Lock()
	defer stdoutLock.Unlock()
	for _, line := range bytes.Split(blob, []byte{'\n'}) {
		fmt.Printf("%s: %s\n", proj, line)
	}
	return nil
}
