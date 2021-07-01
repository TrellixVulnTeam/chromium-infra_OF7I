// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cipd

import (
	"context"
	"fmt"
	bscipd "infra/chromium/bootstrapper/cipd"
	"path/filepath"

	"go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/cipd/common"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/testing/testfs"
)

// PackageInstance is the fake data for an instance of a package.
type PackageInstance struct {
	// Contents maps file paths to the contents.
	Contents map[string]string
}

// Package is the fake data for a package including its refs and all its
// instances.
type Package struct {
	// Refs maps refs to their instance IDs.
	//
	// Missing keys will have a default instance ID computed. An empty
	// string value indicates that the ref does not exist.
	Refs map[string]string

	// Instances maps instance IDs to the instances.
	//
	// Missing keys will have a default instance. A nil value indicates that
	// the instance does not exist.
	Instances map[string]*PackageInstance
}

// Client is the client that will serve fake data for a given host.
type Client struct {
	cipdRoot string
	packages map[string]*Package
}

// Factory creates a factory that returns CIPD clients that use fake data to
// respond to requests.
//
// The fake data is taken from the packages argument, which is a map from
// package names to the Package instances containing the fake data for the
// package. Missing keys will have a default Package. A nil value indicates that
// the given package is not the name of a package.
func Factory(packages map[string]*Package) bscipd.CipdClientFactory {
	return func(ctx context.Context, cipdRoot string) (bscipd.CipdClient, error) {
		return &Client{cipdRoot: cipdRoot, packages: packages}, nil
	}
}

func (c *Client) packageForName(packageName string) (*Package, error) {
	if pkg, ok := c.packages[packageName]; ok {
		if pkg == nil {
			return nil, errors.Reason("unknown package %#v", packageName).Err()
		}
		return pkg, nil
	}
	return &Package{}, nil
}

func (c *Client) ResolveVersion(ctx context.Context, packageName, version string) (common.Pin, error) {
	pkg, err := c.packageForName(packageName)
	if err != nil {
		return common.Pin{}, err
	}
	instanceId, ok := pkg.Refs[version]
	if !ok {
		instanceId = fmt.Sprintf("fake-instance-id|%s|%s", packageName, version)
	} else if instanceId == "" {
		return common.Pin{}, errors.Reason("unknown version %#v of package %#v", version, packageName).Err()
	}
	return common.Pin{PackageName: packageName, InstanceID: instanceId}, nil
}

func (c *Client) EnsurePackages(ctx context.Context, packages common.PinSliceBySubdir, paranoia cipd.ParanoidMode, dryRun bool) (cipd.ActionMap, error) {
	if err := paranoia.Validate(); err != nil {
		return nil, err
	}

	if paranoia != cipd.CheckIntegrity {
		panic(fmt.Sprintf("unexpected paranoia: %s", paranoia))
	}
	if dryRun {
		panic("dryRun is not supported")
	}
	for subdir, pins := range packages {
		if len(pins) != 1 {
			panic(fmt.Sprintf("multiple pins for a subdirectory are not supported: subdirectory: %#v, pins: %#v", subdir, pins))
		}
		pin := pins[0]
		pkg, err := c.packageForName(pin.PackageName)
		if err != nil {
			return nil, err
		}
		instance, ok := pkg.Instances[pin.InstanceID]
		if !ok {
			instance = &PackageInstance{}
		} else if instance == nil {
			return nil, errors.Reason("unknown instance ID %#v of package %#v", pin.InstanceID, pin.PackageName).Err()
		}
		recipeRoot := filepath.Join(c.cipdRoot, filepath.FromSlash(subdir))
		contents := instance.Contents
		if contents == nil {
			contents = map[string]string{}
		}
		if err := testfs.Build(recipeRoot, contents); err != nil {
			panic(err)
		}
	}
	return nil, nil
}
