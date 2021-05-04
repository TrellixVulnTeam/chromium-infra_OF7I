// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	cipd "go.chromium.org/luci/cipd/client/cipd/builder"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// defaultExcludePrefixes excludes parts of Xcode.app that are not necessary for
// any of our purposes. Specifically, it excludes unused platforms like
// AppleTVOS and WatchOS, documentation.
var defaultExcludePrefixes = []string{
	"Contents/Applications",
	"Contents/Developer/Platforms/AppleTVOS.platform",
	"Contents/Developer/Platforms/AppleTVSimulator.platform",
	"Contents/Developer/Platforms/WatchOS.platform",
	"Contents/Developer/Platforms/WatchSimulator.platform",
}

// iosPrefixes excludes parts of Xcode.app not required for building
// Chrome on Mac OS, but is useful for iOS.
var iosPrefixes = []string{
	"Contents/Developer/Platforms/iPhoneOS.platform/Library/Developer/CoreSimulator",
	"Contents/Developer/Platforms/iPhoneSimulator.platform/Developer/SDKs",
}

// Packages is the set of CIPD package definitions. The key is a convenience
// package name for direct reference.
type Packages map[string]cipd.PackageDef

// PackageSpec bundles the package name with a path to its YAML definition file.
type PackageSpec struct {
	Name     string
	YamlPath string
}

func isUnderPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		p := filepath.Join(strings.Split(prefix, "/")...)
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// MakePackageArgs are the parameters for makePackage() to keep them manageable.
type MakePackageArgs struct {
	cipdPackageName   string
	cipdPackagePrefix string
	rootPath          string
	includePrefixes   []string
	excludePrefixes   []string
}

// Makes a CIPD PackageDef using |MakePackageArgs|. Only files in |rootPath|,
// and meanwhile under any of |includePrefixes| relative path prefixes (if
// provided), and not under any of |excludePrefixes| will be included. All paths
// in |rootPath| are first filtered by |includePrefixes| (if provided), then
// tested to ensure it's not in |excludePrefixes|, to be included in the
// package.
func makePackage(args MakePackageArgs) (packageDef cipd.PackageDef, err error) {
	absRootPath, err := filepath.Abs(args.rootPath)
	if err != nil {
		err = errors.Annotate(err, "failed to create an absolute root path from %s", args.rootPath).Err()
		return
	}
	packageDef = cipd.PackageDef{
		Root:             absRootPath,
		InstallMode:      "copy",
		PreserveModTime:  true,
		PreserveWritable: true,
		Package:          args.cipdPackagePrefix + "/" + args.cipdPackageName,
		Data: []cipd.PackageChunkDef{
			{VersionFile: ".xcode_versions/" + args.cipdPackageName + ".cipd_version"},
		},
	}

	err = filepath.Walk(absRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsDir() {
			if !strings.HasPrefix(path, absRootPath+string(os.PathSeparator)) {
				return errors.Reason("file is not in the source folder: %s", path).Err()
			}
			relPath := path[len(absRootPath)+1:]

			if len(args.includePrefixes) > 0 && !isUnderPrefix(relPath, args.includePrefixes) {
				return nil
			}
			if len(args.excludePrefixes) > 0 && isUnderPrefix(relPath, args.excludePrefixes) {
				return nil
			}

			packageDef.Data = append(packageDef.Data, cipd.PackageChunkDef{File: relPath})
		}
		return nil
	})
	return packageDef, err
}

// Makes Xcode's CIPD package definitions, including "mac" and "ios" package
// types.
func makeXcodePackages(xcodeAppPath string, cipdPackagePrefix string) (p Packages, err error) {
	absXcodeAppPath, err := filepath.Abs(xcodeAppPath)
	if err != nil {
		err = errors.Annotate(err, "failed to create an absolute path from %s", xcodeAppPath).Err()
		return
	}

	// Mac package exclude prefixes include prefixes in |defaultExcludePrefixes|
	// and |iosPrefixes|. Use |make|, |copy| and |append| functions to ensure
	// slices won't be accidentally changed.
	excludePrefixesForMacPackage := make([]string, len(defaultExcludePrefixes))
	copy(excludePrefixesForMacPackage, defaultExcludePrefixes)
	excludePrefixesForMacPackage = append(excludePrefixesForMacPackage, iosPrefixes...)

	macMakePackageArgs := MakePackageArgs{
		cipdPackageName:   "mac",
		cipdPackagePrefix: cipdPackagePrefix,
		rootPath:          absXcodeAppPath,
		includePrefixes:   []string{},
		excludePrefixes:   excludePrefixesForMacPackage,
	}
	mac, err := makePackage(macMakePackageArgs)
	if err != nil {
		err = errors.Annotate(err, "failed to create mac cipd pakcage").Err()
	}

	iosMakePackageArgs := MakePackageArgs{
		cipdPackageName:   "ios",
		cipdPackagePrefix: cipdPackagePrefix,
		rootPath:          absXcodeAppPath,
		includePrefixes:   iosPrefixes,
		excludePrefixes:   defaultExcludePrefixes,
	}
	ios, err := makePackage(iosMakePackageArgs)
	if err != nil {
		err = errors.Annotate(err, "failed to create ios cipd pakcage").Err()
	}

	p = Packages{"mac": mac, "ios": ios}
	return
}

// buildCipdPackages builds and optionally uploads CIPD packages to the
// server. `buildFn` callback takes a PackageSpec for each package in `packages`
// and is expected to call `cipd pkg-build` or `cipd create` on it.
func buildCipdPackages(packages Packages, buildFn func(PackageSpec) error) error {
	tmpDir, err := ioutil.TempDir("", "mac_toolchain_")
	if err != nil {
		return errors.Annotate(err, "cannot create a temporary folder for CIPD package configuration files in %s", os.TempDir()).Err()
	}
	defer os.RemoveAll(tmpDir)

	// Iterate deterministically (for testability).
	names := make([]string, 0, len(packages))
	for name := range packages {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		p := packages[name]
		yamlBytes, err := yaml.Marshal(p)
		if err != nil {
			return errors.Annotate(err, "failed to serialize %s.yaml", name).Err()
		}
		yamlPath := filepath.Join(tmpDir, name+".yaml")
		if err = ioutil.WriteFile(yamlPath, yamlBytes, 0600); err != nil {
			return errors.Annotate(err, "failed to write package definition file %s", yamlPath).Err()
		}
		if err = buildFn(PackageSpec{Name: p.Package, YamlPath: yamlPath}); err != nil {
			return err
		}
	}
	return nil
}

func createBuilder(ctx context.Context, tags []string, refs []string, serviceAccountJSON, outputDir string) func(PackageSpec) error {
	builder := func(p PackageSpec) error {
		args := []string{}
		if outputDir != "" {
			pkgParts := strings.Split(p.Name, "/")
			fileName := pkgParts[len(pkgParts)-1] + ".cipd"
			args = append(args, "pkg-build",
				"-out", filepath.Join(outputDir, fileName),
			)
			// Ensure outputDir exists. MkdirAll returns nil if path already exists.
			if err := os.MkdirAll(outputDir, 0777); err != nil {
				return errors.Annotate(err, "failed to create output directory %s", outputDir).Err()
			}
		} else {
			args = append(args,
				"create", "-verification-timeout", "60m",
			)
			for _, tag := range tags {
				args = append(args, "-tag", tag)
			}
			for _, ref := range refs {
				args = append(args, "-ref", strings.ToLower(ref))
			}
		}
		args = append(args, "-pkg-def", p.YamlPath)
		if serviceAccountJSON != "" {
			args = append(args, "-service-account-json", serviceAccountJSON)
		}

		logging.Infof(ctx, "Creating a CIPD package %s", p.Name)
		logging.Debugf(ctx, "Running cipd %s", strings.Join(args, " "))
		if err := RunCommand(ctx, "cipd", args...); err != nil {
			return errors.Annotate(err, "creating a CIPD package failed.").Err()
		}
		return nil
	}
	return builder
}

func packageXcode(ctx context.Context, xcodeAppPath string, cipdPackagePrefix, serviceAccountJSON, outputDir string) error {
	xcodeVersion, buildVersion, err := getXcodeVersion(filepath.Join(xcodeAppPath, "Contents", "version.plist"))
	if err != nil {
		return errors.Annotate(err, "this doesn't look like a valid Xcode.app folder: %s", xcodeAppPath).Err()
	}

	packages, err := makeXcodePackages(xcodeAppPath, cipdPackagePrefix)
	if err != nil {
		return err
	}
	tags := []string{
		"xcode_version:" + xcodeVersion,
		"build_version:" + buildVersion,
	}
	refs := []string{
		strings.ToLower(buildVersion), // Refs must match [a-z0-9_-]*
		"latest",
	}
	buildFn := createBuilder(ctx, tags, refs, serviceAccountJSON, outputDir)

	if err = buildCipdPackages(packages, buildFn); err != nil {
		return err
	}

	fmt.Printf("\nCIPD packages:\n")
	for _, p := range packages {
		fmt.Printf("  %s  %s\n", p.Package, strings.ToLower(buildVersion))
	}

	return nil
}
