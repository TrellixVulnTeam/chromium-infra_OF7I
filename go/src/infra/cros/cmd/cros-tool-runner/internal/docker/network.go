// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package docker

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-tool-runner/internal/common"
)

// CreateNetwork created network for docker.
func CreateNetwork(ctx context.Context, name string) error {
	// Remove network first if already exists.
	log.Printf("remove network %q if already exists.", name)
	RemoveNetwork(ctx, name)

	cmd := exec.Command("docker", "network", "create", name)
	stdout, stderr, err := common.RunWithTimeout(ctx, cmd, time.Minute, true)
	common.PrintToLog(fmt.Sprintf("Create network %q", name), stdout, stderr)
	if err != nil {
		log.Printf("create network %q failed with error: %s", name, err)
		return errors.Annotate(err, "Create network %q", name).Err()
	}
	log.Printf("create network %q: done.", name)
	return nil
}

// RemoveNetwork removes network from docker.
func RemoveNetwork(ctx context.Context, name string) error {
	cmd := exec.Command("docker", "network", "rm", name)
	stdout, stderr, err := common.RunWithTimeout(ctx, cmd, time.Minute, true)
	common.PrintToLog(fmt.Sprintf("remove network %q", name), stdout, stderr)
	if err != nil {
		return errors.Annotate(err, "remove network %q", name).Err()
	}
	log.Printf("remove network %q: done.", name)
	return nil
}
