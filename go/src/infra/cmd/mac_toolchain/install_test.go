// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.chromium.org/luci/common/errors"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInstallXcode(t *testing.T) {
	t.Parallel()

	Convey("installXcode works", t, func() {
		var s MockSession
		ctx := useMockCmd(context.Background(), &s)
		installArgs := InstallArgs{
			xcodeVersion:           "testVersion",
			xcodeAppPath:           "testdata/Xcode-old.app",
			acceptedLicensesFile:   "testdata/acceptedLicenses.plist",
			cipdPackagePrefix:      "test/prefix",
			kind:                   macKind,
			serviceAccountJSON:     "",
			packageInstallerOnBots: "testdata/dummy_installer",
			withRuntime:            false,
		}

		Convey("for accepted license, mac", func() {
			err := installXcode(ctx, installArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 9)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\n")

			So(s.Calls[1].Executable, ShouldEqual, "cipd")
			So(s.Calls[1].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[1].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\n")

			So(s.Calls[2].Executable, ShouldEqual, "chmod")
			So(s.Calls[2].Args, ShouldResemble, []string{
				"-R", "u+w", "testdata/Xcode-old.app",
			})

			So(s.Calls[3].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[3].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[4].Executable, ShouldEqual, "sudo")
			So(s.Calls[4].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-old.app"})

			So(s.Calls[5].Executable, ShouldEqual, "sudo")
			So(s.Calls[5].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[6].Executable, ShouldEqual, "xcrun")
			So(s.Calls[6].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[7].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[7].Args, ShouldResemble, []string{"-status"})

			So(s.Calls[8].Executable, ShouldEqual, "sudo")
			So(s.Calls[8].Args, ShouldResemble, []string{
				"/usr/sbin/DevToolsSecurity",
				"-enable",
			})
		})

		Convey("for already installed package with Developer mode enabled and -runFirstLaunch needs to run", func() {
			s.ReturnError = []error{
				errors.Reason("CIPD package already installed").Err(),
			}
			s.ReturnOutput = []string{
				"cipd dry run",
				"original/Xcode.app",
				"xcode-select -s prints nothing",
				"xcodebuild -runFirstLaunch installs packages",
				"xcrun simctl list prints a list of all simulators installed",
				"xcode-select -s prints nothing",
				"Developer mode is currently enabled.\n",
			}
			err := installXcode(ctx, installArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 7)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\n")
			So(s.Calls[0].Env, ShouldResemble, []string(nil))

			So(s.Calls[1].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[1].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[2].Executable, ShouldEqual, "sudo")
			So(s.Calls[2].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-old.app"})

			So(s.Calls[3].Executable, ShouldEqual, "sudo")
			So(s.Calls[3].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[4].Executable, ShouldEqual, "xcrun")
			So(s.Calls[4].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[5].Executable, ShouldEqual, "sudo")
			So(s.Calls[5].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "original/Xcode.app"})

			So(s.Calls[6].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[6].Args, ShouldResemble, []string{"-status"})

		})

		Convey("for already installed package with Developer mode disabled", func() {
			s.ReturnError = []error{errors.Reason("already installed").Err()}
			s.ReturnOutput = []string{
				"",
				"original/Xcode.app",
				"xcode-select -s prints nothing",
				"xcodebuild -runFirstLaunch installs packages",
				"xcrun simctl list prints a list of all simulators installed",
				"Developer mode is currently disabled.",
			}
			err := installXcode(ctx, installArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 8)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\n")

			So(s.Calls[1].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[1].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[2].Executable, ShouldEqual, "sudo")
			So(s.Calls[2].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-old.app"})

			So(s.Calls[3].Executable, ShouldEqual, "sudo")
			So(s.Calls[3].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[4].Executable, ShouldEqual, "xcrun")
			So(s.Calls[4].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[5].Executable, ShouldEqual, "sudo")
			So(s.Calls[5].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "original/Xcode.app"})

			So(s.Calls[6].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[6].Args, ShouldResemble, []string{"-status"})

			So(s.Calls[7].Executable, ShouldEqual, "sudo")
			So(s.Calls[7].Args, ShouldResemble, []string{
				"/usr/sbin/DevToolsSecurity",
				"-enable",
			})
		})

		Convey("with a service account", func() {
			installArgs.serviceAccountJSON = "test/service-account.json"
			err := installXcode(ctx, installArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 9)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
				"-service-account-json", "test/service-account.json",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\n")

			So(s.Calls[1].Executable, ShouldEqual, "cipd")
			So(s.Calls[1].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
				"-service-account-json", "test/service-account.json",
			})
			So(s.Calls[1].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\n")

			So(s.Calls[2].Executable, ShouldEqual, "chmod")
			So(s.Calls[2].Args, ShouldResemble, []string{
				"-R", "u+w", "testdata/Xcode-old.app",
			})

			So(s.Calls[3].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[3].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[4].Executable, ShouldEqual, "sudo")
			So(s.Calls[4].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-old.app"})

			So(s.Calls[5].Executable, ShouldEqual, "sudo")
			So(s.Calls[5].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[6].Executable, ShouldEqual, "xcrun")
			So(s.Calls[6].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[7].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[7].Args, ShouldResemble, []string{"-status"})

			So(s.Calls[8].Executable, ShouldEqual, "sudo")
			So(s.Calls[8].Args, ShouldResemble, []string{"/usr/sbin/DevToolsSecurity", "-enable"})
		})

		Convey("for new license, ios", func() {
			s.ReturnError = []error{errors.Reason("already installed").Err()}
			s.ReturnOutput = []string{
				"cipd dry run",
				"old/xcode/path",
				"xcode-select -s prints nothing",
				"license accept",
				"xcode-select -s prints nothing",
				"old/xcode/path",
				"xcode-select -s prints nothing",
				"xcodebuild -runFirstLaunch",
				"xcrun simctl list prints a list of all simulators installed",
				"xcode-select -s prints nothing",
				"Developer mode is currently disabled.",
			}

			installArgs.xcodeAppPath = "testdata/Xcode-new.app"
			installArgs.kind = iosKind
			err := installXcode(ctx, installArgs)
			So(err, ShouldBeNil)
			So(len(s.Calls), ShouldEqual, 12)

			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-new.app",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual,
				"test/prefix/mac testVersion\n"+
					"test/prefix/ios testVersion\n")

			So(s.Calls[1].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[1].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[2].Executable, ShouldEqual, "sudo")
			So(s.Calls[2].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-new.app"})

			So(s.Calls[3].Executable, ShouldEqual, "sudo")
			So(s.Calls[3].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-license", "accept"})

			So(s.Calls[4].Executable, ShouldEqual, "sudo")
			So(s.Calls[4].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "old/xcode/path"})

			So(s.Calls[5].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[5].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[6].Executable, ShouldEqual, "sudo")
			So(s.Calls[6].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-new.app"})

			So(s.Calls[7].Executable, ShouldEqual, "sudo")
			So(s.Calls[7].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[8].Executable, ShouldEqual, "xcrun")
			So(s.Calls[8].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[9].Executable, ShouldEqual, "sudo")
			So(s.Calls[9].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "old/xcode/path"})

			So(s.Calls[10].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[10].Args, ShouldResemble, []string{"-status"})

			So(s.Calls[11].Executable, ShouldEqual, "sudo")
			So(s.Calls[11].Args, ShouldResemble, []string{"/usr/sbin/DevToolsSecurity", "-enable"})
		})

	})

	Convey("install Xcode ios mode with/without ios runtime", t, func() {
		var s MockSession
		ctx := useMockCmd(context.Background(), &s)
		installArgs := InstallArgs{
			xcodeVersion:           "testVersion",
			xcodeAppPath:           "testdata/Xcode-old.app",
			acceptedLicensesFile:   "testdata/acceptedLicenses.plist",
			cipdPackagePrefix:      "test/prefix",
			kind:                   iosKind,
			serviceAccountJSON:     "",
			packageInstallerOnBots: "testdata/dummy_installer",
			withRuntime:            true,
		}

		Convey("install with runtime", func() {
			installArgsForTest := installArgs
			installArgsForTest.withRuntime = true
			// Clean up the added runtime dir.
			defer os.RemoveAll("testdata/Xcode-old.app/Contents/Developer/Platforms")
			err := installXcode(ctx, installArgsForTest)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 13)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\ntest/prefix/ios testVersion\n")

			So(s.Calls[1].Executable, ShouldEqual, "cipd")
			So(s.Calls[1].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[1].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\ntest/prefix/ios testVersion\n")

			So(s.Calls[2].Executable, ShouldEqual, "chmod")
			So(s.Calls[2].Args, ShouldResemble, []string{
				"-R", "u+w", "testdata/Xcode-old.app",
			})

			So(s.Calls[3].Executable, ShouldEqual, "cipd")
			So(s.Calls[3].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testVersion",
			})

			// Normalize for win builder tests.
			runtimeInstallPath := filepath.FromSlash("testdata/Xcode-old.app/Contents/Developer/Platforms/iPhoneOS.platform/Library/Developer/CoreSimulator/Profiles/Runtimes")

			So(s.Calls[4].Executable, ShouldEqual, "cipd")
			So(s.Calls[4].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", runtimeInstallPath,
			})
			So(s.Calls[4].ConsumedStdin, ShouldEqual, "test/prefix/ios_runtime testVersion\n")

			So(s.Calls[5].Executable, ShouldEqual, "cipd")
			So(s.Calls[5].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", runtimeInstallPath,
			})
			So(s.Calls[5].ConsumedStdin, ShouldEqual, "test/prefix/ios_runtime testVersion\n")

			So(s.Calls[6].Executable, ShouldEqual, "chmod")
			So(s.Calls[6].Args, ShouldResemble, []string{
				"-R", "u+w", runtimeInstallPath,
			})

			So(s.Calls[7].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[7].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[8].Executable, ShouldEqual, "sudo")
			So(s.Calls[8].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-old.app"})

			So(s.Calls[9].Executable, ShouldEqual, "sudo")
			So(s.Calls[9].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[10].Executable, ShouldEqual, "xcrun")
			So(s.Calls[10].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[11].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[11].Args, ShouldResemble, []string{"-status"})

			So(s.Calls[12].Executable, ShouldEqual, "sudo")
			So(s.Calls[12].Args, ShouldResemble, []string{
				"/usr/sbin/DevToolsSecurity",
				"-enable",
			})
		})

		Convey("with runtime but runtime already exist", func() {
			installArgsForTest := installArgs
			installArgsForTest.withRuntime = true
			installArgsForTest.xcodeAppPath = "testdata/Xcode-with-runtime.app"
			err := installXcode(ctx, installArgsForTest)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 9)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-with-runtime.app",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\ntest/prefix/ios testVersion\n")

			So(s.Calls[1].Executable, ShouldEqual, "cipd")
			So(s.Calls[1].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", "testdata/Xcode-with-runtime.app",
			})
			So(s.Calls[1].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\ntest/prefix/ios testVersion\n")

			So(s.Calls[2].Executable, ShouldEqual, "chmod")
			So(s.Calls[2].Args, ShouldResemble, []string{
				"-R", "u+w", "testdata/Xcode-with-runtime.app",
			})

			So(s.Calls[3].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[3].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[4].Executable, ShouldEqual, "sudo")
			So(s.Calls[4].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-with-runtime.app"})

			So(s.Calls[5].Executable, ShouldEqual, "sudo")
			So(s.Calls[5].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[6].Executable, ShouldEqual, "xcrun")
			So(s.Calls[6].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[7].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[7].Args, ShouldResemble, []string{"-status"})

			So(s.Calls[8].Executable, ShouldEqual, "sudo")
			So(s.Calls[8].Args, ShouldResemble, []string{
				"/usr/sbin/DevToolsSecurity",
				"-enable",
			})
		})

		Convey("without runtime", func() {
			installArgsForTest := installArgs
			installArgsForTest.withRuntime = false
			installArgsForTest.xcodeAppPath = "testdata/Xcode-old.app"
			err := installXcode(ctx, installArgsForTest)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 9)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[0].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\ntest/prefix/ios testVersion\n")

			So(s.Calls[1].Executable, ShouldEqual, "cipd")
			So(s.Calls[1].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", "testdata/Xcode-old.app",
			})
			So(s.Calls[1].ConsumedStdin, ShouldEqual, "test/prefix/mac testVersion\ntest/prefix/ios testVersion\n")

			So(s.Calls[2].Executable, ShouldEqual, "chmod")
			So(s.Calls[2].Args, ShouldResemble, []string{
				"-R", "u+w", "testdata/Xcode-old.app",
			})

			So(s.Calls[3].Executable, ShouldEqual, "/usr/bin/xcode-select")
			So(s.Calls[3].Args, ShouldResemble, []string{"-p"})

			So(s.Calls[4].Executable, ShouldEqual, "sudo")
			So(s.Calls[4].Args, ShouldResemble, []string{"/usr/bin/xcode-select", "-s", "testdata/Xcode-old.app"})

			So(s.Calls[5].Executable, ShouldEqual, "sudo")
			So(s.Calls[5].Args, ShouldResemble, []string{"/usr/bin/xcodebuild", "-runFirstLaunch"})

			So(s.Calls[6].Executable, ShouldEqual, "xcrun")
			So(s.Calls[6].Args, ShouldResemble, []string{"simctl", "list"})

			So(s.Calls[7].Executable, ShouldEqual, "/usr/sbin/DevToolsSecurity")
			So(s.Calls[7].Args, ShouldResemble, []string{"-status"})

			So(s.Calls[8].Executable, ShouldEqual, "sudo")
			So(s.Calls[8].Args, ShouldResemble, []string{
				"/usr/sbin/DevToolsSecurity",
				"-enable",
			})
		})
	})

	Convey("resolveRef works", t, func() {
		var s MockSession
		ctx := useMockCmd(context.Background(), &s)
		Convey("ref exists", func() {
			err := resolveRef(ctx, "test/prefix/ios_runtime", "testXcodeVersion", "")
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 1)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testXcodeVersion",
			})
		})
		Convey("ref doesn't exist", func() {
			s.ReturnError = []error{errors.Reason("input ref doesn't exist").Err()}
			err := resolveRef(ctx, "test/prefix/ios_runtime", "testNonExistRef", "")
			So(s.Calls, ShouldHaveLength, 1)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testNonExistRef",
			})
			So(err.Error(), ShouldContainSubstring, "Error when resolving package path test/prefix/ios_runtime with ref testNonExistRef.")
		})
	})

	Convey("resolveRuntimeRef works", t, func() {
		var s MockSession
		ctx := useMockCmd(context.Background(), &s)
		Convey("only input xcode version", func() {
			resolveRuntimeRefArgs := ResolveRuntimeRefArgs{
				runtimeVersion:     "",
				xcodeVersion:       "testXcodeVersion",
				packagePath:        "test/prefix/ios_runtime",
				serviceAccountJSON: "",
			}
			ver, err := resolveRuntimeRef(ctx, resolveRuntimeRefArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 1)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testXcodeVersion",
			})
			So(ver, ShouldEqual, "testXcodeVersion")
		})
		Convey("only input sim runtime version", func() {
			resolveRuntimeRefArgs := ResolveRuntimeRefArgs{
				runtimeVersion:     "testSimVersion",
				xcodeVersion:       "",
				packagePath:        "test/prefix/ios_runtime",
				serviceAccountJSON: "",
			}
			ver, err := resolveRuntimeRef(ctx, resolveRuntimeRefArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 1)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion",
			})
			So(ver, ShouldEqual, "testSimVersion")
		})
		Convey("input both Xcode and sim version: default runtime exists", func() {
			resolveRuntimeRefArgs := ResolveRuntimeRefArgs{
				runtimeVersion:     "testSimVersion",
				xcodeVersion:       "testXcodeVersion",
				packagePath:        "test/prefix/ios_runtime",
				serviceAccountJSON: "",
			}
			ver, err := resolveRuntimeRef(ctx, resolveRuntimeRefArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 1)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion_testXcodeVersion",
			})
			So(ver, ShouldEqual, "testSimVersion_testXcodeVersion")
		})
		Convey("input both Xcode and sim version: fallback to uploaded runtime", func() {
			s.ReturnError = []error{errors.Reason("default runtime doesn't exist").Err()}
			resolveRuntimeRefArgs := ResolveRuntimeRefArgs{
				runtimeVersion:     "testSimVersion",
				xcodeVersion:       "testXcodeVersion",
				packagePath:        "test/prefix/ios_runtime",
				serviceAccountJSON: "",
			}
			ver, err := resolveRuntimeRef(ctx, resolveRuntimeRefArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 2)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion_testXcodeVersion",
			})
			So(s.Calls[1].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion",
			})
			So(ver, ShouldEqual, "testSimVersion")
		})
		Convey("input both Xcode and sim version: fallback to any latest runtime", func() {
			s.ReturnError = []error{
				errors.Reason("default runtime doesn't exist").Err(),
				errors.Reason("uploaded runtime doesn't exist").Err(),
			}
			resolveRuntimeRefArgs := ResolveRuntimeRefArgs{
				runtimeVersion:     "testSimVersion",
				xcodeVersion:       "testXcodeVersion",
				packagePath:        "test/prefix/ios_runtime",
				serviceAccountJSON: "",
			}
			ver, err := resolveRuntimeRef(ctx, resolveRuntimeRefArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 3)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion_testXcodeVersion",
			})
			So(s.Calls[1].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion",
			})
			So(s.Calls[2].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion_latest",
			})
			So(ver, ShouldEqual, "testSimVersion_latest")
		})
		Convey("input both Xcode and sim version: raise when all fallbacks fail", func() {
			s.ReturnError = []error{
				errors.Reason("default runtime doesn't exist").Err(),
				errors.Reason("uploaded runtime doesn't exist").Err(),
				errors.Reason("any latest runtime doesn't exist").Err(),
			}
			resolveRuntimeRefArgs := ResolveRuntimeRefArgs{
				runtimeVersion:     "testSimVersion",
				xcodeVersion:       "testXcodeVersion",
				packagePath:        "test/prefix/ios_runtime",
				serviceAccountJSON: "",
			}
			ver, err := resolveRuntimeRef(ctx, resolveRuntimeRefArgs)
			So(s.Calls, ShouldHaveLength, 3)
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion_testXcodeVersion",
			})
			So(s.Calls[1].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion",
			})
			So(s.Calls[2].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion_latest",
			})
			So(err.Error(), ShouldContainSubstring, "Failed to resolve runtime ref given runtime version: testSimVersion, xcode version: testXcodeVersion.")
			So(ver, ShouldEqual, "")
		})

	})

	Convey("installRuntime works", t, func() {
		var s MockSession
		ctx := useMockCmd(context.Background(), &s)

		Convey("install an Xcode default runtime", func() {
			runtimeInstallArgs := RuntimeInstallArgs{
				runtimeVersion:     "",
				xcodeVersion:       "testVersion",
				installPath:        "test/path/to/install/runtimes",
				cipdPackagePrefix:  "test/prefix",
				serviceAccountJSON: "",
			}
			err := installRuntime(ctx, runtimeInstallArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 4)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testVersion",
			})

			So(s.Calls[1].Executable, ShouldEqual, "cipd")
			So(s.Calls[1].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "test/path/to/install/runtimes",
			})
			So(s.Calls[1].ConsumedStdin, ShouldEqual, "test/prefix/ios_runtime testVersion\n")

			So(s.Calls[2].Executable, ShouldEqual, "cipd")
			So(s.Calls[2].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", "test/path/to/install/runtimes",
			})
			So(s.Calls[2].ConsumedStdin, ShouldEqual, "test/prefix/ios_runtime testVersion\n")

			So(s.Calls[3].Executable, ShouldEqual, "chmod")
			So(s.Calls[3].Args, ShouldResemble, []string{
				"-R", "u+w", "test/path/to/install/runtimes",
			})
		})

		Convey("install an uploaded runtime", func() {
			runtimeInstallArgs := RuntimeInstallArgs{
				runtimeVersion:     "testSimVersion",
				xcodeVersion:       "",
				installPath:        "test/path/to/install/runtimes",
				cipdPackagePrefix:  "test/prefix",
				serviceAccountJSON: "",
			}
			err := installRuntime(ctx, runtimeInstallArgs)
			So(err, ShouldBeNil)
			So(s.Calls, ShouldHaveLength, 4)
			So(s.Calls[0].Executable, ShouldEqual, "cipd")
			So(s.Calls[0].Args, ShouldResemble, []string{
				"resolve", "test/prefix/ios_runtime", "-version", "testSimVersion",
			})

			So(s.Calls[1].Executable, ShouldEqual, "cipd")
			So(s.Calls[1].Args, ShouldResemble, []string{
				"puppet-check-updates", "-ensure-file", "-", "-root", "test/path/to/install/runtimes",
			})
			So(s.Calls[1].ConsumedStdin, ShouldEqual, "test/prefix/ios_runtime testSimVersion\n")

			So(s.Calls[2].Executable, ShouldEqual, "cipd")
			So(s.Calls[2].Args, ShouldResemble, []string{
				"ensure", "-ensure-file", "-", "-root", "test/path/to/install/runtimes",
			})
			So(s.Calls[2].ConsumedStdin, ShouldEqual, "test/prefix/ios_runtime testSimVersion\n")

			So(s.Calls[3].Executable, ShouldEqual, "chmod")
			So(s.Calls[3].Args, ShouldResemble, []string{
				"-R", "u+w", "test/path/to/install/runtimes",
			})
		})
	})

}
