// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag/flagenum"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
)

// DefaultCipdPackagePrefix is a package prefix for all the xcode packages.
const DefaultCipdPackagePrefix = "infra_internal/ios/xcode"

// AcceptedLicensesFile keeps record of the accepted Xcode licenses.
const AcceptedLicensesFile = "/Library/Preferences/com.apple.dt.Xcode.plist"

// PackageInstallerOnBots is a special script for securely installing packages
// on bots.
const PackageInstallerOnBots = "/usr/local/bin/xcode_install_wrapper.py"

// Relative path from Xcode.app where simulator runtimes are stored.
const XcodeIOSSimulatorRuntimeRelPath = "Contents/Developer/Platforms/iPhoneOS.platform/Library/Developer/CoreSimulator/Profiles/Runtimes"

// Filename of default simulator runtime in Xcode package.
const XcodeIOSSimulatorRuntimeFilename = "iOS.simruntime"

// Package name of iOS runtime in CIPD.
const IosRuntimePackageName = "ios_runtime"

// Package name of Mac package in CIPD. The package contains Xcode contents that
// are both useful in Mac & iOS.
const MacPackageName = "mac"

// Package name of iOS package in CIPD. The package contains iOS SDK.
const IosPackageName = "ios"

// KindType is the type for enum values for the -kind argument.
type KindType string

var _ flag.Value = (*KindType)(nil)

const (
	macKind        = KindType(MacPackageName)
	iosKind        = KindType(IosPackageName)
	iosRuntimeKind = KindType(IosRuntimePackageName)
	// DefaultKind is the default value for the -kind flag.
	DefaultKind = macKind
)

// KindTypeEnum is the corresponding Enum type for the -kind argument.
var KindTypeEnum = flagenum.Enum{
	MacPackageName: macKind,
	IosPackageName: iosKind,
}

// String implements flag.Value
func (t *KindType) String() string {
	return KindTypeEnum.FlagString(*t)
}

// Set implements flag.Value
func (t *KindType) Set(v string) error {
	return KindTypeEnum.FlagSet(t, v)
}

type commonFlags struct {
	subcommands.CommandRunBase
	verbose           bool
	cipdPackagePrefix string
}

type installRun struct {
	commonFlags
	xcodeVersion       string
	outputDir          string
	kind               KindType
	serviceAccountJSON string
}

type uploadRun struct {
	commonFlags
	xcodePath          string
	serviceAccountJSON string
}

type packageRun struct {
	commonFlags
	xcodePath string
	outputDir string
}

type uploadRuntimeRun struct {
	commonFlags
	runtimePath        string
	serviceAccountJSON string
}

type packageRuntimeRun struct {
	commonFlags
	runtimePath string
	outputDir   string
}

type installRuntimeRun struct {
	commonFlags
	runtimeVersion     string
	xcodeVersion       string
	outputDir          string
	serviceAccountJSON string
}

func stripLastTrailingSlash(prefix string) string {
	// Strip the trailing /.
	for strings.HasSuffix(prefix, "/") {
		prefix = prefix[:len(prefix)-1]
	}
	return prefix
}

func (c *commonFlags) ModifyContext(ctx context.Context) context.Context {
	if c.verbose {
		ctx = logging.SetLevel(ctx, logging.Debug)
	}
	return ctx
}

func (c *installRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if c.xcodeVersion == "" {
		errors.Log(ctx, errors.Reason("no Xcode version specified (-xcode-version)").Err())
		return 1
	}
	if c.outputDir == "" {
		errors.Log(ctx, errors.Reason("no output folder specified (-output-dir)").Err())
		return 1
	}
	logging.Infof(ctx, "About to install Xcode %s in %s for %s", c.xcodeVersion, c.outputDir, c.kind.String())

	c.cipdPackagePrefix = stripLastTrailingSlash(c.cipdPackagePrefix)
	installArgs := InstallArgs{
		xcodeVersion:           c.xcodeVersion,
		xcodeAppPath:           c.outputDir,
		acceptedLicensesFile:   AcceptedLicensesFile,
		cipdPackagePrefix:      c.cipdPackagePrefix,
		kind:                   c.kind,
		serviceAccountJSON:     c.serviceAccountJSON,
		packageInstallerOnBots: PackageInstallerOnBots,
	}
	if err := installXcode(ctx, installArgs); err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

// Entrance function to upload an Xcode for "upload" cmd line switch. Also uploads
// the iOS runtime package within the Xcode.
func (c *uploadRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if c.xcodePath == "" {
		errors.Log(ctx, errors.Reason("path to Xcode.app is not specified (-xcode-path)").Err())
		return 1
	}
	c.cipdPackagePrefix = stripLastTrailingSlash(c.cipdPackagePrefix)
	packageRuntimeAndXcodeArgs := PackageRuntimeAndXcodeArgs{
		xcodeAppPath:       c.xcodePath,
		cipdPackagePrefix:  c.cipdPackagePrefix,
		serviceAccountJSON: c.serviceAccountJSON,
		outputDir:          "",
	}
	if err := packageRuntimeAndXcode(ctx, packageRuntimeAndXcodeArgs); err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

// Entrance function to locally package an Xcode for "package" cmd line switch.
// Also packages the iOS runtime package within the Xcode.
func (c *packageRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if c.xcodePath == "" {
		errors.Log(ctx, errors.Reason("path to Xcode.app is not specified (-xcode-path)").Err())
		return 1
	}
	if c.outputDir == "" {
		errors.Log(ctx, errors.Reason("output directory is not specified (-output-dir)").Err())
		return 1
	}
	c.cipdPackagePrefix = stripLastTrailingSlash(c.cipdPackagePrefix)
	packageRuntimeAndXcodeArgs := PackageRuntimeAndXcodeArgs{
		xcodeAppPath:       c.xcodePath,
		cipdPackagePrefix:  c.cipdPackagePrefix,
		serviceAccountJSON: "",
		outputDir:          c.outputDir,
	}
	if err := packageRuntimeAndXcode(ctx, packageRuntimeAndXcodeArgs); err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

// Entrance function to upload a runtime for upload-runtime cmd line switch.
func (c *uploadRuntimeRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if c.runtimePath == "" {
		errors.Log(ctx, errors.Reason("path to iOS runtime is not specified (-runtime-path)").Err())
		return 1
	}

	packageRuntimeArgs := PackageRuntimeArgs{
		xcodeAppPath:       "",
		runtimePath:        stripLastTrailingSlash(c.runtimePath),
		cipdPackagePrefix:  stripLastTrailingSlash(c.cipdPackagePrefix),
		serviceAccountJSON: c.serviceAccountJSON,
		outputDir:          "",
	}
	if err := packageRuntime(ctx, packageRuntimeArgs); err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

// Entrance function to package a runtime locally for package-runtime cmd line
// switch.
func (c *packageRuntimeRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if c.runtimePath == "" {
		errors.Log(ctx, errors.Reason("path to iOS runtime is not specified (-runtime-path)").Err())
		return 1
	}
	if c.outputDir == "" {
		errors.Log(ctx, errors.Reason("output directory is not specified (-output-dir)").Err())
		return 1
	}

	packageRuntimeArgs := PackageRuntimeArgs{
		xcodeAppPath:       "",
		runtimePath:        stripLastTrailingSlash(c.runtimePath),
		cipdPackagePrefix:  stripLastTrailingSlash(c.cipdPackagePrefix),
		serviceAccountJSON: "",
		outputDir:          c.outputDir,
	}
	if err := packageRuntime(ctx, packageRuntimeArgs); err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

// Entrance function to install a runtime for install-runtime cmd line switch.
func (c *installRuntimeRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if c.runtimeVersion == "" && c.xcodeVersion == "" {
		errors.Log(ctx, errors.Reason("no runtime or xcode version specified").Err())
		return 1
	}
	if c.outputDir == "" {
		errors.Log(ctx, errors.Reason("no output folder specified (-output-dir)").Err())
		return 1
	}
	logging.Infof(ctx, "About to install runtime %s %s to %s", c.runtimeVersion, c.xcodeVersion, c.outputDir)

	c.cipdPackagePrefix = stripLastTrailingSlash(c.cipdPackagePrefix)

	runtimeInstallArgs := RuntimeInstallArgs{
		runtimeVersion:     c.runtimeVersion,
		xcodeVersion:       c.xcodeVersion,
		installPath:        c.outputDir,
		cipdPackagePrefix:  c.cipdPackagePrefix,
		serviceAccountJSON: c.serviceAccountJSON,
	}
	if err := installRuntime(ctx, runtimeInstallArgs); err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

func commonFlagVars(c *commonFlags) {
	c.Flags.BoolVar(&c.verbose, "verbose", false, "Log more.")
	c.Flags.StringVar(&c.cipdPackagePrefix, "cipd-package-prefix", DefaultCipdPackagePrefix, "CIPD package prefix.")
}

func installFlagVars(c *installRun) {
	commonFlagVars(&c.commonFlags)
	c.Flags.StringVar(&c.xcodeVersion, "xcode-version", "", "Xcode version code. (required)")
	c.Flags.StringVar(&c.outputDir, "output-dir", "", "Path where to install contents of Xcode.app (required).")
	c.Flags.StringVar(&c.serviceAccountJSON, "service-account-json", "", "Service account to use for authentication.")
	c.Flags.Var(&c.kind, "kind", "Installation kind: "+KindTypeEnum.Choices()+". (default: \""+string(DefaultKind)+"\")")
	c.kind = DefaultKind
}

func uploadFlagVars(c *uploadRun) {
	commonFlagVars(&c.commonFlags)
	c.Flags.StringVar(&c.serviceAccountJSON, "service-account-json", "", "Service account to use for authentication.")
	c.Flags.StringVar(&c.xcodePath, "xcode-path", "", "Path to Xcode.app to be uploaded. (required)")
}

func packageFlagVars(c *packageRun) {
	commonFlagVars(&c.commonFlags)
	c.Flags.StringVar(&c.xcodePath, "xcode-path", "", "Path to Xcode.app to be uploaded. (required)")
	c.Flags.StringVar(&c.outputDir, "output-dir", "", "Path to drop created CIPD packages. (required)")
}

func uploadRuntimeFlagVars(c *uploadRuntimeRun) {
	commonFlagVars(&c.commonFlags)
	c.Flags.StringVar(&c.serviceAccountJSON, "service-account-json", "", "Service account to use for authentication.")
	c.Flags.StringVar(&c.runtimePath, "runtime-path", "", "Path to iOS.simruntime to be uploaded. (required)")
}

func packageRuntimeFlagVars(c *packageRuntimeRun) {
	commonFlagVars(&c.commonFlags)
	c.Flags.StringVar(&c.runtimePath, "runtime-path", "", "Path to iOS.simruntime to be uploaded. (required)")
	c.Flags.StringVar(&c.outputDir, "output-dir", "", "Path to drop created CIPD packages. (required)")
}

func installRuntimeFlagVars(c *installRuntimeRun) {
	commonFlagVars(&c.commonFlags)
	c.Flags.StringVar(&c.runtimeVersion, "runtime-version", "", "iOS runtime version. Format e.g. \"ios-14-4\"")
	c.Flags.StringVar(&c.xcodeVersion, "xcode-version", "", "Xcode version code.")
	c.Flags.StringVar(&c.outputDir, "output-dir", "", "Path where to install the runtime (required).")
	c.Flags.StringVar(&c.serviceAccountJSON, "service-account-json", "", "Service account to use for authentication.")
}

var (
	cmdInstall = &subcommands.Command{
		UsageLine: "install <options>",
		ShortDesc: "Installs Xcode.",
		LongDesc: `Installs the requested parts of Xcode toolchain.

Note: the "Xcode.app" part of the path is not created.
Instead, "Contents" folder is placed directly in the folder specified
by the -output-dir. If you want an actual app that Finder can launch, specify
-output-dir "<path>/Xcode.app".`,
		CommandRun: func() subcommands.CommandRun {
			c := &installRun{}
			installFlagVars(c)
			return c
		},
	}

	cmdUpload = &subcommands.Command{
		UsageLine: "upload <options>",
		ShortDesc: "Uploads Xcode CIPD packages.",
		LongDesc:  "Creates and uploads Xcode toolchain CIPD packages.",
		CommandRun: func() subcommands.CommandRun {
			c := &uploadRun{}
			uploadFlagVars(c)
			return c
		},
	}

	cmdPackage = &subcommands.Command{
		UsageLine: "package <options>",
		ShortDesc: "Create CIPD packages locally.",
		LongDesc:  "Package Xcode into CIPD packages locally (will not upload).",
		CommandRun: func() subcommands.CommandRun {
			c := &packageRun{}
			packageFlagVars(c)
			return c
		},
	}

	cmdUploadRuntime = &subcommands.Command{
		UsageLine: "upload-runtime <options>",
		ShortDesc: "Uploads iOS runtime package.",
		LongDesc:  "Creates and uploads iOS runtime CIPD package.",
		CommandRun: func() subcommands.CommandRun {
			c := &uploadRuntimeRun{}
			uploadRuntimeFlagVars(c)
			return c
		},
	}

	cmdPackageRuntime = &subcommands.Command{
		UsageLine: "package-runtime <options>",
		ShortDesc: "Creates iOS runtime CIPD package locally.",
		LongDesc:  "Packages iOS runtime CIPD package locally (won't upload).",
		CommandRun: func() subcommands.CommandRun {
			c := &packageRuntimeRun{}
			packageRuntimeFlagVars(c)
			return c
		},
	}

	cmdInstallRuntime = &subcommands.Command{
		UsageLine: "install-runtime <options>",
		ShortDesc: "Installs Runtime.",
		LongDesc: `Installs the requested iOS runtime package to -output-dir.

If only "runtime-version" is specified, installs the runtime mannually uploaded.
If only "xcode-version" is specified, installs the default runtime came with the
Xcode version.
If both "runtime-version" and "xcode-version" are specified, the command finds
and installs the package by the following priority:
  1) The default runtime of input Xcode, if the runtime version matches.
  2) Manually uploaded runtime of the version specified.
  3) Any latest runtime of the version specified in CIPD.`,
		CommandRun: func() subcommands.CommandRun {
			c := &installRuntimeRun{}
			installRuntimeFlagVars(c)
			return c
		},
	}
)

func main() {
	application := &cli.Application{
		Name:  "mac_toolchain",
		Title: "Mac OS / iOS toolchain management",
		Context: func(ctx context.Context) context.Context {
			goLoggerCfg := gologger.LoggerConfig{Out: os.Stderr}
			goLoggerCfg.Format = "[%{level:.1s} %{time:2006-01-02 15:04:05}] %{message}"
			ctx = goLoggerCfg.Use(ctx)

			ctx = logging.SetLevel(ctx, logging.Warning)
			ctx = useRealExec(ctx)
			return ctx
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			cmdInstall,
			cmdUpload,
			cmdPackage,
			cmdUploadRuntime,
			cmdPackageRuntime,
			cmdInstallRuntime,
		},
	}
	os.Exit(subcommands.Run(application, nil))
}
