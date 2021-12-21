// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"log"
	"os"

	"github.com/golang/protobuf/jsonpb"
	build_api "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/lucictx"
)

// readContainersMetadata reads the jsonproto at path containers metadata file.
func readContainersMetadata(p string) (*build_api.ContainerMetadata, error) {
	in := &build_api.ContainerMetadata{}
	r, err := os.Open(p)
	if err != nil {
		return nil, errors.Annotate(err, "read container metadata %q", p).Err()
	}
	err = jsonpb.Unmarshal(r, in)
	return in, errors.Annotate(err, "read container metadata %q", p).Err()
}

func findContainer(cm *build_api.ContainerMetadata, lookupKey, name string) *build_api.ContainerImageInfo {
	containers := cm.GetContainers()
	if containers == nil {
		return nil
	}
	imageMap, ok := containers[lookupKey]
	if !ok {
		log.Printf("Image %q not found", name)
		return nil
	}
	return imageMap.Images[name]
}

func useSystemAuth(ctx context.Context, authFlags *authcli.Flags) (context.Context, error) {
	authOpts, err := authFlags.Options()
	if err != nil {
		return nil, errors.Annotate(err, "switching to system auth").Err()
	}

	authCtx, err := lucictx.SwitchLocalAccount(ctx, "system")
	if err == nil {
		// If there's a system account use it (the case of running on Swarming).
		// Otherwise default to user credentials (the local development case).
		authOpts.Method = auth.LUCIContextMethod
		return authCtx, nil
	}
	log.Printf("System account not found, err %s.\nFalling back to user credentials for auth.\n", err)
	return ctx, nil
}
