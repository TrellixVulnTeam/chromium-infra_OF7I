// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/system/filesystem"
)

// InstallPackagesArgs are the parameters for installPackages() to keep them manageable.
type InstallPackagesArgs struct {
	ref                string
	rootPath           string
	cipdPackagePrefix  string
	kind               KindType
	serviceAccountJSON string
}

// Installs the cpid package to |rootPath| of specified |kind|, find package
// as input |cipdPackagePrefix| & |ref|. These args are passed within
// |InstallPackagesArgs| struct.
func installPackages(ctx context.Context, args InstallPackagesArgs) error {
	cipdArgs := []string{
		"-ensure-file", "-",
		"-root", args.rootPath,
	}
	if args.serviceAccountJSON != "" {
		cipdArgs = append(cipdArgs, "-service-account-json", args.serviceAccountJSON)
	}
	cipdCheckArgs := append([]string{"puppet-check-updates"}, cipdArgs...)
	cipdEnsureArgs := append([]string{"ensure"}, cipdArgs...)

	ensureSpec := ""
	switch args.kind {
	case macKind:
		ensureSpec += fmt.Sprintf("%s/%s %s\n", args.cipdPackagePrefix, MacPackageName, args.ref)
	case iosKind:
		ensureSpec += fmt.Sprintf("%s/%s %s\n%s/%s %s\n", args.cipdPackagePrefix, MacPackageName, args.ref, args.cipdPackagePrefix, IosPackageName, args.ref)
	case iosRuntimeKind:
		ensureSpec += fmt.Sprintf("%s/%s %s\n", args.cipdPackagePrefix, IosRuntimePackageName, args.ref)
	default:
		return errors.Reason("unknown package kind: %s", args.kind).Err()
	}

	// Check if `cipd ensure` will do something. Note: `cipd puppet-check-updates`
	// returns code 0 when `cipd ensure` has work to do, and "fails" otherwise.
	// TODO(sergeyberezin): replace this with a better option when
	// https://crbug.com/788032 is fixed.
	if err := RunWithStdin(ctx, ensureSpec, "cipd", cipdCheckArgs...); err != nil {
		// The rest logic ensures the Xcode is intact so it only applies to
		// iosKind or macKind.
		if args.kind != macKind && args.kind != iosKind {
			return nil
		}
		xcodeAppPath := args.rootPath
		// Sometimes Xcode cache in bots loses Contents/Developer/usr and CIPD
		// doesn't check if the package is intact. Add an additional check and
		// only return when the directory exists.
		binDirPath := filepath.Join(xcodeAppPath, "Contents", "Developer", "usr", "bin")
		if _, statErr := os.Stat(binDirPath); !os.IsNotExist(statErr) {
			return nil
		}
		logging.Warningf(ctx, "Contents/Developer/usr/bin doesn't exist in cached Xcode. Reinstalling Xcode.")
		// Remove and create an empty Xcode dir so `cipd ensure` will work to
		// download a new one.
		if removeErr := filesystem.RemoveAll(xcodeAppPath); removeErr != nil {
			return errors.Annotate(removeErr, "failed to remove corrupted Xcode package.").Err()
		}
		if err := os.MkdirAll(xcodeAppPath, 0700); err != nil {
			return errors.Annotate(err, "failed to create a folder %s", xcodeAppPath).Err()
		}
	}

	if err := RunWithStdin(ctx, ensureSpec, "cipd", cipdEnsureArgs...); err != nil {
		return errors.Annotate(err, "failed to install CIPD packages: %s", ensureSpec).Err()
	}
	// Xcode really wants its files to be user-writable (hangs mysteriously
	// otherwise). CIPD by default installs everything read-only. Update
	// permissions post-install.
	//
	// TODO(sergeyberezin): remove this once crbug.com/803158 is resolved and all
	// currently used Xcode versions are re-uploaded.
	if err := RunCommand(ctx, "chmod", "-R", "u+w", args.rootPath); err != nil {
		return errors.Annotate(err, "failed to update package permissions in %s for %s", args.rootPath, args.kind).Err()
	}
	return nil
}

func needToAcceptLicense(ctx context.Context, xcodeAppPath, acceptedLicensesFile string) bool {
	licenseInfoFile := filepath.Join(xcodeAppPath, "Contents", "Resources", "LicenseInfo.plist")

	licenseID, licenseType, err := getXcodeLicenseInfo(licenseInfoFile)
	if err != nil {
		errors.Log(ctx, err)
		return true
	}

	acceptedLicenseID, err := getXcodeAcceptedLicense(acceptedLicensesFile, licenseType)
	if err != nil {
		errors.Log(ctx, err)
		return true
	}

	// Historically all Xcode build numbers have been in the format of AANNNN, so
	// a simple string compare works.  If Xcode's build numbers change this may
	// need a more complex compare.
	if licenseID <= acceptedLicenseID {
		// Don't accept the license of older toolchain builds, this will break the
		// license of newer builds.
		return false
	}
	return true
}

func getXcodePath(ctx context.Context) string {
	path, err := RunOutput(ctx, "/usr/bin/xcode-select", "-p")
	if err != nil {
		return ""
	}
	return strings.Trim(path, " \n")
}

func setXcodePath(ctx context.Context, xcodeAppPath string) error {
	err := RunCommand(ctx, "sudo", "/usr/bin/xcode-select", "-s", xcodeAppPath)
	if err != nil {
		return errors.Annotate(err, "failed xcode-select -s %s", xcodeAppPath).Err()
	}
	return nil
}

// RunWithXcodeSelect temporarily sets the Xcode path with `sudo xcode-select
// -s` and runs a callback.
func RunWithXcodeSelect(ctx context.Context, xcodeAppPath string, f func() error) error {
	oldPath := getXcodePath(ctx)
	if oldPath != "" {
		defer setXcodePath(ctx, oldPath)
	}
	if err := setXcodePath(ctx, xcodeAppPath); err != nil {
		return err
	}
	if err := f(); err != nil {
		return err
	}
	return nil
}

func acceptLicense(ctx context.Context, xcodeAppPath string) error {
	err := RunWithXcodeSelect(ctx, xcodeAppPath, func() error {
		return RunCommand(ctx, "sudo", "/usr/bin/xcodebuild", "-license", "accept")
	})
	if err != nil {
		return errors.Annotate(err, "failed to accept new license").Err()
	}
	return nil
}

func finalizeInstall(ctx context.Context, xcodeAppPath, xcodeVersion, packageInstallerOnBots string) error {
	return RunWithXcodeSelect(ctx, xcodeAppPath, func() error {
		err := RunCommand(ctx, "sudo", "/usr/bin/xcodebuild", "-runFirstLaunch")
		if err != nil {
			return errors.Annotate(err, "failed to install Xcode packages").Err()
		}
		return nil
	})
}

func enableDeveloperMode(ctx context.Context) error {
	out, err := RunOutput(ctx, "/usr/sbin/DevToolsSecurity", "-status")
	if err != nil {
		return errors.Annotate(err, "failed to run /usr/sbin/DevToolsSecurity -status").Err()
	}
	if !strings.Contains(out, "Developer mode is currently enabled.") {
		err = RunCommand(ctx, "sudo", "/usr/sbin/DevToolsSecurity", "-enable")
		if err != nil {
			return errors.Annotate(err, "failed to run sudo /usr/sbin/DevToolsSecurity -enable").Err()
		}
	}
	return nil
}

// InstallArgs are the parameters for installXcode() to keep them manageable.
type InstallArgs struct {
	xcodeVersion           string
	xcodeAppPath           string
	acceptedLicensesFile   string
	cipdPackagePrefix      string
	kind                   KindType
	serviceAccountJSON     string
	packageInstallerOnBots string
}

func installXcode(ctx context.Context, args InstallArgs) error {
	if err := os.MkdirAll(args.xcodeAppPath, 0700); err != nil {
		return errors.Annotate(err, "failed to create a folder %s", args.xcodeAppPath).Err()
	}
	installPackagesArgs := InstallPackagesArgs{
		ref:                args.xcodeVersion,
		rootPath:           args.xcodeAppPath,
		cipdPackagePrefix:  args.cipdPackagePrefix,
		kind:               args.kind,
		serviceAccountJSON: args.serviceAccountJSON,
	}
	if err := installPackages(ctx, installPackagesArgs); err != nil {
		return err
	}
	if needToAcceptLicense(ctx, args.xcodeAppPath, args.acceptedLicensesFile) {
		if err := acceptLicense(ctx, args.xcodeAppPath); err != nil {
			return err
		}
	}
	if err := finalizeInstall(ctx, args.xcodeAppPath, args.xcodeVersion, args.packageInstallerOnBots); err != nil {
		return err
	}
	return enableDeveloperMode(ctx)
}
