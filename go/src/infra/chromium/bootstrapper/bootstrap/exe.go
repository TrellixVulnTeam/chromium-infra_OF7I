// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"path/filepath"

	"infra/chromium/bootstrapper/cas"
	"infra/chromium/bootstrapper/cipd"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

// ExeBootstrapper provides the functionality for deploying the bootstrapped
// executable and providing the information necessary to execute it.
type ExeBootstrapper struct {
	cipd *cipd.Client
	cas  *cas.Client
}

func NewExeBootstrapper(cipd *cipd.Client, cas *cas.Client) *ExeBootstrapper {
	return &ExeBootstrapper{cipd: cipd, cas: cas}
}

func (b *ExeBootstrapper) GetBootstrappedExeInfo(ctx context.Context, input *Input) (*BootstrappedExe, error) {
	var source isBootstrappedExe_Source
	if input.casRecipeBundle != nil {
		source = &BootstrappedExe_Cas{
			Cas: input.casRecipeBundle,
		}
	} else {
		logging.Infof(ctx, "resolving CIPD package %s@%s", input.exeProperties.Exe.CipdPackage, input.exeProperties.Exe.CipdVersion)
		pin, err := b.cipd.ResolveVersion(ctx, input.exeProperties.Exe.CipdPackage, input.exeProperties.Exe.CipdVersion)
		if err != nil {
			return nil, err
		}
		source = &BootstrappedExe_Cipd{
			Cipd: &Cipd{
				Server:           chromeinfra.CIPDServiceURL,
				Package:          input.exeProperties.Exe.CipdPackage,
				RequestedVersion: input.exeProperties.Exe.CipdVersion,
				ActualVersion:    pin.InstanceID,
			},
		}
	}
	return &BootstrappedExe{
		Source: source,
		Cmd:    input.exeProperties.Exe.Cmd,
	}, nil
}

// DeployExe fetches the executable described by exe and returns the command for
// invoking the executable.
func (b *ExeBootstrapper) DeployExe(ctx context.Context, exe *BootstrappedExe) ([]string, error) {
	var packagePath string
	switch source := exe.Source.(type) {
	case *BootstrappedExe_Cipd:
		cipdSource := source.Cipd
		logging.Infof(ctx, "downloading CIPD package %s@%s", cipdSource.Package, cipdSource.ActualVersion)
		var err error
		packagePath, err = b.cipd.DownloadPackage(ctx, cipdSource.Package, cipdSource.ActualVersion, "bootstrapped-exe")
		if err != nil {
			return nil, err
		}

	case *BootstrappedExe_Cas:
		casSource := source.Cas
		logging.Infof(ctx, "downloading CAS isolated %s/%d", casSource.Digest.Hash, casSource.Digest.SizeBytes)
		var err error
		packagePath, err = b.cas.Download(ctx, casSource.CasInstance, casSource.Digest)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.Reason("unknown source %s", source).Err()
	}

	var cmd []string
	cmd = append(cmd, exe.Cmd...)
	cmd[0] = filepath.Join(packagePath, cmd[0])
	return cmd, nil
}
