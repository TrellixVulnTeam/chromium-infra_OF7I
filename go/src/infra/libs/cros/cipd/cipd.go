// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cipd is an internal CIPD tool wrapper.
package cipd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/common/errors"
)

const service = "https://chrome-infra-packages.appspot.com"

// Package contains information about an installed package.
type Package struct {
	Package  string `json:"package"`
	Pin      Pin    `json:"pin"`
	Tracking string `json:"tracking"`
}

// Pin contains information about an installed package instance.
type Pin struct {
	Package    string `json:"package"`
	InstanceID string `json:"instance_id"`
}

type installedPackagesFunc = func(root string) ([]Package, error)
type installedPackagesInnerFunc = func(root string) ([]byte, error)
type jsonCmdOutputFunc = func(cmd *exec.Cmd) ([]byte, error)

// InstalledPackages returns information about installed CIPD packages.
func InstalledPackages(applicationName string) installedPackagesFunc {
	return func(root string) ([]Package, error) {
		out, err := installedPackages(applicationName)(root)
		if err != nil {
			return nil, errors.Annotate(err, "get CIPD packages for %s", root).Err()
		}
		pkgs, err := unmarshalPackages(out)
		if err != nil {
			return nil, errors.Annotate(err, "get CIPD packages for %s", root).Err()
		}
		return pkgs, nil
	}
}

// installedPackages returns the raw JSON from running a cipd installed command.
func installedPackages(applicationName string) installedPackagesInnerFunc {
	return func(root string) ([]byte, error) {
		cmd := exec.Command("cipd", "installed", "-root", root)
		return jsonCmdOutput(applicationName)(cmd)
	}
}

func unmarshalPackages(jsonData []byte) ([]Package, error) {
	var obj struct {
		Result map[string][]Package `json:"result"`
	}
	if err := json.Unmarshal(jsonData, &obj); err != nil {
		return nil, errors.Annotate(err, "unmarshal packages").Err()
	}
	if obj.Result == nil {
		return nil, errors.Reason("unmarshal packages: bad JSON").Err()
	}
	pkgs, ok := obj.Result[""]
	if !ok {
		return nil, errors.Reason("unmarshal packages: bad JSON").Err()
	}
	return pkgs, nil
}

// Run a cipd command that supports -json-output and return the raw JSON.
func jsonCmdOutput(applicationName string) jsonCmdOutputFunc {
	return func(cmd *exec.Cmd) ([]byte, error) {
		f, err := ioutil.TempFile("", fmt.Sprintf("%s-cipd-output", applicationName))
		if err != nil {
			return nil, errors.Annotate(err, "JSON command output").Err()
		}
		defer os.Remove(f.Name())
		cmd.Args = append(cmd.Args, "-json-output", f.Name())
		if _, err = cmd.Output(); err != nil {
			return nil, errors.Annotate(err, "JSON command output").Err()
		}
		out, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, errors.Annotate(err, "JSON command output").Err()
		}
		return out, nil
	}
}

// FindPackage find the package by a given package name.
func FindPackage(packageName, cipdPackagePath string) (*Package, error) {
	errAnnotation := fmt.Sprintf("find package %s", packageName)
	d, err := executableDir()
	if err != nil {
		return nil, errors.Annotate(err, errAnnotation).Err()
	}
	root, err := findCIPDRootDir(d)
	if err != nil {
		return nil, errors.Annotate(err, errAnnotation).Err()
	}
	pkgs, err := InstalledPackages(packageName)(root)
	if err != nil {
		return nil, errors.Annotate(err, errAnnotation).Err()
	}
	for _, p := range pkgs {
		if !strings.HasPrefix(p.Package, cipdPackagePath) {
			continue
		}
		return &p, nil
	}
	return nil, errors.Reason(fmt.Sprintf("%s package: not found in %s", packageName, root)).Err()
}

// DescribePackage returns information about a package instances.
func DescribePackage(ctx context.Context, pkg, version string) (*cipd.InstanceDescription, error) {
	opts := cipd.ClientOptions{
		ServiceURL:      service,
		AnonymousClient: http.DefaultClient,
	}
	client, err := cipd.NewClient(opts)
	if err != nil {
		return nil, errors.Annotate(err, "describe package").Err()
	}
	pin, err := client.ResolveVersion(ctx, pkg, version)
	if err != nil {
		return nil, errors.Annotate(err, "describe package").Err()
	}
	d, err := client.DescribeInstance(ctx, pin, nil)
	if err != nil {
		return nil, errors.Annotate(err, "describe package").Err()
	}
	return d, nil
}

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

func isCIPDRootDir(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".cipd"))
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
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
