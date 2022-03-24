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

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.chromium.org/luci/auth"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	projectpb "go.chromium.org/luci/common/proto/config"
	realmspb "go.chromium.org/luci/common/proto/realms"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	"go.chromium.org/luci/config/impl/remote"
	triciumpb "go.chromium.org/luci/cv/api/config/legacy"
	cvpb "go.chromium.org/luci/cv/api/config/v2"
	"go.chromium.org/luci/hardcoded/chromeinfra"
	logdogpb "go.chromium.org/luci/logdog/api/config/svcconfig"
	notifypb "go.chromium.org/luci/luci_notify/api/config"
	milopb "go.chromium.org/luci/milo/api/config"
	schedulerpb "go.chromium.org/luci/scheduler/appengine/messages"
)

var outputDir = flag.String("output-dir", "-", "Where to store fetched files or - to print them to stdout for grepping")
var configService = flag.String("config-service-host", chromeinfra.ConfigServiceHost, "Hostname of LUCI Config service to query")
var convertJSON = flag.Bool("convert-json", false, ("Convert fetched files to JSONPB (best-effort)." +
	" If output to -, wraps messages in {\"proj\": ..., \"data\": ...}."))

// quick and dirty map for known config file names to the proto message they
// contain.
var messageMap map[string]protoreflect.Message = map[string]protoreflect.Message{
	"commit-queue.cfg":       (*cvpb.Config)(nil).ProtoReflect(),
	"cr-buildbucket-dev.cfg": (*bbpb.BuildbucketCfg)(nil).ProtoReflect(),
	"cr-buildbucket.cfg":     (*bbpb.BuildbucketCfg)(nil).ProtoReflect(),
	"luci-logdog-dev.cfg":    (*logdogpb.ProjectConfig)(nil).ProtoReflect(),
	"luci-logdog.cfg":        (*logdogpb.ProjectConfig)(nil).ProtoReflect(),
	"luci-milo-dev.cfg":      (*milopb.Project)(nil).ProtoReflect(),
	"luci-milo.cfg":          (*milopb.Project)(nil).ProtoReflect(),
	"luci-notify-dev.cfg":    (*notifypb.ProjectConfig)(nil).ProtoReflect(),
	"luci-notify.cfg":        (*notifypb.ProjectConfig)(nil).ProtoReflect(),
	"luci-scheduler-dev.cfg": (*schedulerpb.ProjectConfig)(nil).ProtoReflect(),
	"luci-scheduler.cfg":     (*schedulerpb.ProjectConfig)(nil).ProtoReflect(),
	"project.cfg":            (*projectpb.ProjectCfg)(nil).ProtoReflect(),
	"realms-dev.cfg":         (*realmspb.RealmsCfg)(nil).ProtoReflect(),
	"realms.cfg":             (*realmspb.RealmsCfg)(nil).ProtoReflect(),
	"tricium-dev.cfg":        (*triciumpb.ProjectConfig)(nil).ProtoReflect(),
	"tricium-prod.cfg":       (*triciumpb.ProjectConfig)(nil).ProtoReflect(),
}

var stdoutLock sync.Mutex

func main() {
	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)

	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Expecting one positional argument with the name of the file to fetch.\n")
		os.Exit(2)
	}

	path := flag.Arg(0)

	var msg protoreflect.Message
	if *convertJSON {
		var ok bool
		if msg, ok = messageMap[flag.Arg(0)]; !ok {
			fmt.Fprintf(os.Stderr, "Cannot convert %q to JSON: unknown file", path)
			os.Exit(2)
		}
	}

	if err := run(ctx, path, *outputDir, msg); err != nil {
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

func run(ctx context.Context, path, output string, msg protoreflect.Message) error {
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
				if err := processProject(ctx, proj, path, output, msg); err != nil {
					logging.Errorf(ctx, "Failed when processing %s: %s", proj, err)
					return err
				}
				return nil
			}
		}
	})
}

func processProject(ctx context.Context, proj, path, output string, msg protoreflect.Message) error {
	var blob []byte
	err := cfgclient.Get(ctx, config.Set("projects/"+proj), path, cfgclient.Bytes(&blob), nil)
	if err != nil {
		return err
	}

	if msg != nil {
		decoded := msg.New().Interface()
		if err = prototext.Unmarshal(blob, decoded); err != nil {
			return errors.Annotate(err, "decoding textpb").Err()
		}
		blob, err = protojson.MarshalOptions{
			Multiline:     true,
			Indent:        "  ",
			UseProtoNames: true,
		}.Marshal(decoded)
		if err != nil {
			return errors.Annotate(err, "encoding jsonpb").Err()
		}
	}

	if output != "-" {
		stdoutLock.Lock()
		defer stdoutLock.Unlock()
		fmt.Printf("writing %s\n", proj)
		return ioutil.WriteFile(filepath.Join(output, proj+".cfg"), blob, 0666)
	}

	stdoutLock.Lock()
	defer stdoutLock.Unlock()
	if msg == nil {
		// raw
		for _, line := range bytes.Split(blob, []byte{'\n'}) {
			fmt.Printf("%s: %s\n", proj, line)
		}
	} else {
		// json
		fmt.Printf("{\"project\": \"%s\", \"data\": %s}\n", proj, blob)
	}
	return nil
}
