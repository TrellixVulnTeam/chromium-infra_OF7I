// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"path/filepath"

	"infra/chromium/bootstrapper/cipd"

	"go.chromium.org/luci/common/logging"
)

// ExeBootstrapper provides the functionality for deploying the bootstrapped
// executable and providing the information necessary to execute it.
type ExeBootstrapper struct {
	cipd *cipd.Client
}

func NewExeBootstrapper(cipd *cipd.Client) *ExeBootstrapper {
	return &ExeBootstrapper{cipd: cipd}
}

// DeployExe fetches the CIPD bundle identified by the exe field of the build's
// $bootstrap property and returns the command for invoking the executable.
func (b *ExeBootstrapper) DeployExe(ctx context.Context, input *Input) ([]string, error) {
	logging.Infof(ctx, "downloading CIPD package %s@%s", input.properties.Exe.CipdPackage, input.properties.Exe.CipdVersion)
	packagePath, err := b.cipd.DownloadPackage(ctx, input.properties.Exe.CipdPackage, input.properties.Exe.CipdVersion)
	if err != nil {
		return nil, err
	}
	var cmd []string
	cmd = append(cmd, input.properties.Exe.Cmd...)
	cmd[0] = filepath.Join(packagePath, cmd[0])
	return cmd, nil
}
