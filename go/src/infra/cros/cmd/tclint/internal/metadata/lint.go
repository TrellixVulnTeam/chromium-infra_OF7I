// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package metadata provides functions to lint Chrome OS integration test
// metadata.
package metadata

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"go.chromium.org/chromiumos/config/go/api/test/metadata/v1"
	"go.chromium.org/luci/common/data/stringset"
)

// Lint checks a given metadata specification for violations of requirements
// stated in the API definition.
func Lint(spec *metadata.Specification) Result {
	if len(spec.RemoteTestDrivers) == 0 {
		return errorResult("Specification must contain at least one RemoteTestDriver")
	}

	result := Result{}
	for _, rtd := range spec.RemoteTestDrivers {
		result.Merge(lintRTD(rtd))
	}
	result.Merge(verifyUniqueRemoteTestDriverNames(spec.RemoteTestDrivers))
	return result
}

func verifyUniqueRemoteTestDriverNames(rtds []*metadata.RemoteTestDriver) Result {
	result := Result{}
	ns := make([]string, len(rtds))
	for i, rtd := range rtds {
		ns[i] = rtd.GetName()
	}
	if repeated := formatRepeated(ns); repeated != "" {
		result.AppendError("RemoteTestDriver names must be unique, found repeated name(s): %s", repeated)
	}
	return result
}

func formatRepeated(ss []string) string {
	seen := stringset.New(len(ss))
	repeated := stringset.New(len(ss))
	for _, s := range ss {
		if seen.Has(s) {
			repeated.Add(fmt.Sprintf("'%s'", s))
		}
		seen.Add(s)
	}
	return strings.Join(repeated.ToSortedSlice(), ",")
}

func lintRTD(rtd *metadata.RemoteTestDriver) Result {
	result := lintRTDName(rtd.GetName())
	for _, t := range rtd.Tests {
		result.Merge(lintTest(t, rtd.GetName()))
	}
	result.MergeWithContext(verifyUniqueTestNames(rtd.Tests), "RemoteTestDriver '%s'", rtd.GetName())
	return result
}

func verifyUniqueTestNames(tests []*metadata.Test) Result {
	result := Result{}
	ns := make([]string, len(tests))
	for i, test := range tests {
		ns[i] = test.GetName()
	}
	if repeated := formatRepeated(ns); repeated != "" {
		result.AppendError("Test names must be unique, found repeated name(s): %s", repeated)
	}
	return result
}

const (
	rtdCollection  = "remoteTestDrivers"
	testCollection = "tests"
)

func lintRTDName(name string) Result {
	result := Result{}
	tag := fmt.Sprintf("RemoteTestDriver '%s'", name)
	if result.MergeWithContext(lintResourceName(name), tag); result.Errors != nil {
		return result
	}
	parts := strings.SplitN(name, "/", 3)
	switch len(parts) {
	case 0:
		result.AppendError("%s: name must be of the form '%s/{remoteTestDriver}', found empty string", tag, rtdCollection)
	case 1:
		if parts[0] == rtdCollection {
			result.AppendError("%s: name must be of the form '%s/{remoteTestDriver}', missing name after '%s'", tag, rtdCollection, rtdCollection)
		} else {
			result.AppendError("%s: name must be of the form '%s/{remoteTestDriver}', missing prefix '%s'", tag, rtdCollection, rtdCollection)
		}
	case 2:
		if parts[0] != rtdCollection {
			result.AppendError("%s: name must be of the form '%s/{remoteTestDriver}', missing prefix '%s'", tag, rtdCollection, rtdCollection)
		}
	default:
		result.AppendError("%s: name must be of the form '%s/{remoteTestDriver}', found trailing suffix '%s'", tag, rtdCollection, parts[2])
	}

	return result
}

func lintTest(test *metadata.Test, rtdName string) Result {
	result := lintTestName(test.GetName(), rtdName)
	tag := fmt.Sprintf("Test '%s'", test.GetName())
	if test.DutConstraint != nil {
		result.MergeWithContext(lintDutConstraint(test.DutConstraint), tag)
	}
	return result
}

func lintTestName(name string, rtdName string) Result {
	result := Result{}
	tag := fmt.Sprintf("Test '%s'", name)
	if result.MergeWithContext(lintResourceName(name), tag); result.Errors != nil {
		return result
	}
	prefix := fmt.Sprintf("%s/", rtdName)
	if !strings.HasPrefix(name, prefix) {
		result.AppendError("%s: name must be prefixed with RemoteTestDriver name '%s'", tag, rtdName)
		return result
	}
	name = strings.TrimPrefix(name, prefix)
	parts := strings.Split(name, "/")
	switch len(parts) {
	case 0:
		result.AppendError("%s: name must be of the form '%s/{test}', found empty string", tag, testCollection)
	case 1:
		if parts[0] == testCollection {
			result.AppendError("%s: name must be of the form '%s/{test}', missing name after '%s'", tag, testCollection, testCollection)
		} else {
			result.AppendError("%s: name must be of the form '%s/{test}', missing prefix '%s'", tag, testCollection, testCollection)
		}
	case 2:
		if parts[0] != testCollection {
			result.AppendError("%s: name must be of the form '%s/{test}', missing prefix '%s'", tag, testCollection, testCollection)
		}
	default:
		result.AppendError("%s: name must be of the form '%s/{test}', found trailing suffix '%s'", tag, testCollection, parts[2])
	}
	return result
}

// lintResourceName lints resource names.
//
// This lint enforces some rules in addition to the recommendations in
// https://aip.dev/122.
//
// The returned results _do not_ add the argument as a context in diagnostic
// messages, because the caller can provide better context about the object
// being named (e.g. "RemoteTest Driver <name>" instead of "<name>").
func lintResourceName(name string) Result {
	if name == "" {
		return errorResult("name must be non-empty (https://aip.dev/122)")
	}

	result := Result{}
	u, err := url.Parse(name)
	if err != nil {
		result.AppendErrorWithContext(err, "parse name")
		return result
	}

	if u.Scheme != "" {
		result.AppendError("name must be a URL path component (https://aip.dev/122), found non-empty scheme '%s'", u.Scheme)
	}
	if u.Opaque != "" {
		result.AppendError("name must be a URL path component (https://aip.dev/122), found non-empty opaque data '%s'", u.Opaque)
	}
	if u.User != nil {
		result.AppendError("name must be a URL path component (https://aip.dev/122), found non-empty user information '%s'", u.User.String())
	}
	if u.Host != "" {
		result.AppendError("name must be a URL path component (https://aip.dev/122), found non-empty host '%s'", u.Host)
	}
	if u.Fragment != "" {
		result.AppendError("resource versions are not yet supported, found version '%s'", u.Fragment)
	}

	if u.Path == "" {
		result.AppendError("name must be a non-empty URL path component (https://aip.dev/122), found empty path")
		return result
	}

	if strings.HasPrefix(u.Path, "/") {
		result.AppendError("name must be a URL relative path component (https://aip.dev/122), found absolute path '%s'", u.Path)
	}
	if strings.HasSuffix(u.Path, "/") {
		result.AppendError("name must not contain a trailing '/' (https://aip.dev/122), found trailing '/' in '%s'", u.Path)
	}
	if !isASCII(u.Path) {
		result.AppendError("name must only use ASCII characters, found non-ASCII chracters in '%s'", strconv.QuoteToASCII(u.Path))
	}
	return result
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func lintDutConstraint(cond *metadata.DUTConstraint) Result {
	result := Result{}
	if cond.Setup == nil && cond.Config == nil {
		result.AppendError("dut_constraint: some constraint must be set")
	}
	if cond.Setup != nil {
		result.MergeWithContext(lintDUTSetupConstraint(cond.Setup), "dut_constraint")
	}
	if cond.Config != nil {
		result.MergeWithContext(lintDUTConfigConstraint(cond.Config), "dut_constraint")
	}
	return result
}

func lintDUTSetupConstraint(cond *metadata.DUTSetupConstraint) Result {
	dutType := decls.NewObjectType("chromiumos.config.api.test.metadata.v1.DUTSetupConstraint.DUT")
	dutValue := &metadata.DUTSetupConstraint_DUT{}
	env, err := cel.NewEnv(
		cel.Container("chromiumos.config.api.test.dut.v1"),
		// Adding the type to environment allows the expression to use fully
		// qualified type names like `chromiumos.config.api.test.dut.v1.Wifi`.
		cel.Types(dutValue),
		cel.Declarations(
			decls.NewIdent("dut", dutType, nil),
		),
	)
	if err != nil {
		// Failure to setup the environment is a programming error that we never
		// expect to recover from.
		panic(err)
	}
	result := Result{}
	result.MergeWithContext(parseAndCheckExpression(env, cond.Expression), "setup")
	return result
}

func parseAndCheckExpression(env *cel.Env, expr string) Result {
	result := Result{}
	if expr == "" {
		result.AppendError("expression must be non-empty")
		return result
	}
	if _, iss := env.Compile(expr); iss.Err() != nil {
		// Reported issues are frequently displayed on multiple lines.
		// Adding a leading newline makes the multi-line display easier to read.
		result.AppendError("compile expression: \n%s", iss.String())
		return result
	}
	return result
}

func lintDUTConfigConstraint(cond *metadata.DUTConfigConstraint) Result {
	dutType := decls.NewObjectType("chromiumos.config.api.test.metadata.v1.DUTConfigConstraint.DUT")
	dutValue := &metadata.DUTConfigConstraint_DUT{}
	env, err := cel.NewEnv(
		cel.Container("chromiumos.config.api"),
		// Adding the type to environment allows the expression to use fully
		// qualified type names like `chromiumos.config.api.HardwareFeatures`.
		cel.Types(dutValue),
		cel.Declarations(
			decls.NewIdent("dut", dutType, nil),
		),
	)
	if err != nil {
		// Failure to setup the environment is a programming error that we never
		// expect to recover from.
		panic(err)
	}
	result := Result{}
	result.MergeWithContext(parseAndCheckExpression(env, cond.Expression), "config")
	return result
}
