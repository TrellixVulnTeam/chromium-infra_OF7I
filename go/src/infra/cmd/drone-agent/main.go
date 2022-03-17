// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

// Command drone-agent is the client that talks to the drone queen
// service to provide Swarming bots for running tasks against test
// devices.  See the README.
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/grpc/prpc"

	"infra/appengine/drone-queen/api"
	"infra/cmd/drone-agent/internal/agent"
	"infra/cmd/drone-agent/internal/bot"
	"infra/cmd/drone-agent/internal/draining"
	"infra/cmd/drone-agent/internal/tokman"
)

const (
	drainingFile   = "drone-agent.drain"
	oauthTokenPath = "/var/lib/swarming/oauth_bot_token.json"
)

var (
	queenService = os.Getenv("DRONE_AGENT_QUEEN_SERVICE")
	// DRONE_AGENT_SWARMING_URL is the URL of the Swarming
	// instance.  Should be a full URL without the path,
	// e.g. https://host.example.com
	swarmingURL       = os.Getenv("DRONE_AGENT_SWARMING_URL")
	dutCapacity       = getIntEnv("DRONE_AGENT_DUT_CAPACITY", 10)
	reportingInterval = time.Duration(getIntEnv("DRONE_AGENT_REPORTING_INTERVAL_MINS", 1)) * time.Minute

	authOptions = auth.Options{
		Method:                 auth.ServiceAccountMethod,
		ServiceAccountJSONPath: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	}
	workingDirPath = filepath.Join(os.Getenv("HOME"), "skylab_bots")
	// hive value of the drone agent.  This is used for DUT/drone affinity.
	// A drone is assigned DUTs with same hive value.
	hive = initializeHive(os.Getenv("DRONE_AGENT_HIVE"), os.Getenv("DOCKER_DRONE_SERVER_NAME"))
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatal(err)
	}
}

func innerMain() error {
	// TODO(ayatane): Add environment validation.
	ctx, cancel := context.WithCancel(context.Background())
	ctx = notifySIGTERM(ctx)
	ctx = notifyDraining(ctx, filepath.Join(workingDirPath, drainingFile))

	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()

	authn := auth.NewAuthenticator(ctx, auth.SilentLogin, authOptions)

	r, err := tokman.Make(authn, oauthTokenPath, time.Minute)
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		r.KeepNew(ctx)
		wg.Done()
	}()

	h, err := authn.Client()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(workingDirPath, 0777); err != nil {
		return err
	}

	a := agent.Agent{
		Client: api.NewDronePRPCClient(&prpc.Client{
			C:    h,
			Host: queenService,
		}),
		SwarmingURL:       swarmingURL,
		WorkingDir:        workingDirPath,
		ReportingInterval: reportingInterval,
		DUTCapacity:       dutCapacity,
		StartBotFunc:      bot.NewStarter(h).Start,
		Hive:              hive,
	}
	a.Run(ctx)
	return nil
}

const checkDrainingInterval = time.Minute

// notifyDraining returns a context that is marked as draining when a
// file exists at the given path.
func notifyDraining(ctx context.Context, path string) context.Context {
	ctx, drain := draining.WithDraining(ctx)
	_, err := os.Stat(path)
	if err == nil {
		drain()
		return ctx
	}
	go func() {
		for {
			time.Sleep(checkDrainingInterval)
			_, err := os.Stat(path)
			if err == nil {
				drain()
				return
			}
		}
	}()
	return ctx
}

// getIntEnv gets an int value from an environment variable.  If the
// environment variable is not valid or is not set, use the default value.
func getIntEnv(key string, defaultValue int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("Invalid %s, using default value (error: %v)", key, err)
		return defaultValue
	}
	return n
}

// dcLabRegex is the regular expression to identify the Drone server is in a
// data center like lab, e.g. SFO36, in which the server name is like
// 'kube<N>-<SITE>'. If matched, we use the part of '<SITE>' as the hive.
var dcLabRegex = regexp.MustCompile(`^kube[0-9]+-([a-z]+)`)

// initializeHive returns the hive for the agent.
// If hive is not specified, we try to guess it from the hostname.
// The input args are from some envvars, but we don't get them from inside
// the function, so we can keep all code using envvars in a single code block at
// the head of this file for better readability.
func initializeHive(explicitHive, hostname string) string {
	if explicitHive != "" {
		return explicitHive
	}
	log.Printf("Hive not explicitly specified, now guess it by hostname %q", hostname)
	if m := dcLabRegex.FindStringSubmatch(hostname); m != nil {
		return m[1]
	}
	return ""
}
