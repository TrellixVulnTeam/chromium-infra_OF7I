// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package docker

import (
	"context"
	"log"
	"os/exec"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-tool-runner/internal/common"
)

// CreateNetwork created network for docker.
func CreateNetwork(ctx context.Context, name string) error {
	cmd := exec.Command("docker", "network", "create", name)
	out, e, err := common.RunWithTimeout(ctx, cmd, time.Minute)
	if err != nil {
		log.Printf("Create network %q: %s", name, e)
		return errors.Annotate(err, "create network %q", name).Err()
	}
	log.Printf("Create network %q: done. Result: %s", name, out)
	return nil
}

// RemoveNetwork removes network from docker.
func RemoveNetwork(ctx context.Context, name string) error {
	cmd := exec.Command("docker", "network", "rm", name)
	out, e, err := common.RunWithTimeout(ctx, cmd, time.Minute)
	if err != nil {
		log.Printf("Remove network %q: %s", name, e)
		return errors.Annotate(err, "remove network %q", name).Err()
	}
	log.Printf("Remove network %q: done. Result: %s", name, out)
	return nil
}
