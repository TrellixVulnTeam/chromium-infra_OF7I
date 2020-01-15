// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	cipd "infra/libs/cros/cipd"

	luciCipd "go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/common/errors"
)

// FindStableVersion2Package locates the CIPD package containing the current executable
func FindStableVersion2Package() (*cipd.Package, error) {
	d, err := ExecutableDir()
	if err != nil {
		return nil, errors.Annotate(err, "find stable_version2 package").Err()
	}
	root, err := FindCIPDRootDir(d)
	if err != nil {
		return nil, errors.Annotate(err, "find stable_version2 package").Err()
	}
	pkgs, err := cipd.InstalledPackages("stable_version2")(root)
	if err != nil {
		return nil, errors.Annotate(err, "find stable_version2 package").Err()
	}
	for _, p := range pkgs {
		if !strings.HasPrefix(p.Package, "chromiumos/infra/stable_version2/") {
			continue
		}
		return &p, nil
	}
	return nil, errors.Reason("find stable_version2 package: not found").Err()
}

// ExecutableDir returns the directory the current executable came
// from.
func ExecutableDir() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", errors.Annotate(err, "get executable directory").Err()
	}
	return filepath.Dir(p), nil
}

// FindCIPDRootDir finds the root directory of the CIPD package.
func FindCIPDRootDir(dir string) (string, error) {
	a, err := filepath.Abs(dir)
	if err != nil {
		return "", errors.Annotate(err, "find CIPD root dir").Err()
	}
	for d := a; d != "/"; d = filepath.Dir(d) {
		if IsCIPDRootDir(d) {
			return d, nil
		}
	}
	return "", errors.Reason("find CIPD root dir: no CIPD root above %s", dir).Err()
}

// IsCIPDRootDir heuristically checks whether a given directory looks like a CIPD root directory.
func IsCIPDRootDir(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".cipd"))
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

const service = "https://chrome-infra-packages.appspot.com"

// Describe returns information about a package instances.
func Describe(ctx context.Context, pkg, version string) (*luciCipd.InstanceDescription, error) {
	opts := luciCipd.ClientOptions{
		ServiceURL:      service,
		AnonymousClient: http.DefaultClient,
	}
	client, err := luciCipd.NewClient(opts)
	if err != nil {
		return nil, errors.Annotate(err, "describe package").Err()
	}
	pin, err := client.ResolveVersion(ctx, pkg, version)
	if err != nil {
		return nil, errors.Annotate(err, "resolve version").Err()
	}
	d, err := client.DescribeInstance(ctx, pin, nil)
	if err != nil {
		return nil, errors.Annotate(err, "describe instance").Err()
	}
	return d, nil
}
