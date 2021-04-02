// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"os"
	"strings"

	"howett.net/plist"

	"go.chromium.org/luci/common/errors"
)

type versionPlist struct {
	BuildVersion string `plist:"ProductBuildVersion"`
	XcodeVersion string `plist:"CFBundleShortVersionString"`
}

// getXcodeVersion takes the path to the `version.plist` file within Xcode.app
// and extracts the Xcode and build versions.
func getXcodeVersion(versionFile string) (xcodeVersion string, buildVersion string, err error) {
	var vp versionPlist
	r, err := os.Open(versionFile)
	if err != nil {
		err = errors.Annotate(err, "failed to open %s", versionFile).Err()
		return
	}
	defer r.Close()
	decoder := plist.NewDecoder(r)
	if err = decoder.Decode(&vp); err != nil {
		err = errors.Annotate(err, "failed to decode %s", versionFile).Err()
		return
	}

	xcodeVersion = vp.XcodeVersion
	buildVersion = vp.BuildVersion
	if xcodeVersion == "" || buildVersion == "" {
		err = errors.Reason("Contents/version.plist is missing ProductBuildVersion or CFBundleShortVersionString").Err()
		return
	}
	return
}

type licenseInfoPlist struct {
	LicenseID   string `plist:"licenseID"`
	LicenseType string `plist:"licenseType"`
}

func getXcodeLicenseInfo(licenseInfoFile string) (licenseID string, licenseType string, err error) {
	var licenseInfo licenseInfoPlist
	r, err := os.Open(licenseInfoFile)
	if err != nil {
		err = errors.Annotate(err, "failed to open %s", licenseInfoFile).Err()
		return
	}
	defer r.Close()
	decoder := plist.NewDecoder(r)
	if err = decoder.Decode(&licenseInfo); err != nil {
		err = errors.Annotate(err, "failed to decode %s", licenseInfoFile).Err()
		return
	}
	if licenseInfo.LicenseID == "" || licenseInfo.LicenseType == "" {
		err = errors.Reason("Contents/Resources/LicenseInfo.plist is missing licenseID or licenseType").Err()
		return
	}
	return licenseInfo.LicenseID, licenseInfo.LicenseType, nil
}

type xcodeAcceptedLicenses struct {
	IDELastGMLicenseAgreedTo   string `plist:"IDELastGMLicenseAgreedTo"`
	IDELastBetaLicenseAgreedTo string `plist:"IDELastBetaLicenseAgreedTo"`
}

func getXcodeAcceptedLicense(plistFile, licenseType string) (licenseID string, err error) {
	var acceptedLicenses xcodeAcceptedLicenses
	r, err := os.Open(plistFile)
	if err != nil {
		err = errors.Annotate(err, "failed to open %s", plistFile).Err()
		return
	}
	defer r.Close()
	decoder := plist.NewDecoder(r)
	if err = decoder.Decode(&acceptedLicenses); err != nil {
		err = errors.Annotate(err, "failed to decode %s", plistFile).Err()
		return
	}
	if licenseType == "GM" {
		licenseID = acceptedLicenses.IDELastGMLicenseAgreedTo
	} else {
		licenseID = acceptedLicenses.IDELastBetaLicenseAgreedTo
	}
	return
}

type simulatorVersionPlist struct {
	BundleName       string `plist:"CFBundleName"`
	BundleIdentifier string `plist:"CFBundleIdentifier"`
}

// getSimulatorVersion takes the path to the `Info.plist` in simulator. e.g.
// Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Library/Developer
// /CoreSimulator/Profiles/Runtimes/iOS.simruntime/Contents/Info.plist
// and extracts the simulator runtime bundleName and identifier.
func getSimulatorVersion(plistFile string) (bundleName string, simRuntimeIdentifier string, err error) {
	var simulatorVersion simulatorVersionPlist
	r, err := os.Open(plistFile)
	if err != nil {
		err = errors.Annotate(err, "failed to open %s", plistFile).Err()
		return
	}
	defer r.Close()
	decoder := plist.NewDecoder(r)
	if err = decoder.Decode(&simulatorVersion); err != nil {
		err = errors.Annotate(err, "failed to decode %s", plistFile).Err()
		return
	}

	if simulatorVersion.BundleName == "" || simulatorVersion.BundleIdentifier == "" {
		err = errors.Reason("Simulator Info.plist is missing Bundle name or Bundle identifier").Err()
		return
	}

	// BundleName is a string of format like "iOS 14.4".
	bundleName = simulatorVersion.BundleName
	// BundleIdentifier is a string like
	// "com.apple.CoreSimulator.SimRuntime.iOS-14-4". "ios-14-4" is returned as
	// |simRuntimeIdentifier|.
	simRuntimeIdentifier = strings.ToLower(simulatorVersion.BundleIdentifier[strings.LastIndex(simulatorVersion.BundleIdentifier, ".")+1:])
	return
}
