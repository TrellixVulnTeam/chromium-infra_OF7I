// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

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

// findStableVersion2Package locates the CIPD package containing the current executable
func findStableVersion2Package() (*cipd.Package, error) {
	d, err := executableDir()
	if err != nil {
		return nil, errors.Annotate(err, "find stable_version2 package").Err()
	}
	root, err := findCIPDRootDir(d)
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

// executableDir returns the directory the current executable came
// from.
func executableDir() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", errors.Annotate(err, "get executable directory").Err()
	}
	return filepath.Dir(p), nil
}

// findCIPDRootDir finds the root directory of the CIPD package.
func findCIPDRootDir(dir string) (string, error) {
	a, err := filepath.Abs(dir)
	if err != nil {
		return "", errors.Annotate(err, "find CIPD root dir").Err()
	}
	for d := a; d != "/"; d = filepath.Dir(d) {
		if isCIPDRootDir(d) {
			return d, nil
		}
	}
	return "", errors.Reason("find CIPD root dir: no CIPD root above %s", dir).Err()
}

// isCIPDRootDir heuristically checks whether a given directory looks like a CIPD root directory.
func isCIPDRootDir(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".cipd"))
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

const service = "https://chrome-infra-packages.appspot.com"

// describe returns information about a package instances.
func describe(ctx context.Context, pkg, version string) (*luciCipd.InstanceDescription, error) {
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
