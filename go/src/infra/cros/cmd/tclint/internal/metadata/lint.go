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

	"infra/cros/cmd/tclint/internal/diagnostics"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"go.chromium.org/chromiumos/config/go/api/test/metadata/v1"
	"go.chromium.org/luci/common/data/stringset"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// Lint checks a given metadata specification for violations of requirements
// stated in the API definition.
func Lint(spec *metadata.Specification) diagnostics.Result {
	l := linter{}
	return l.Lint(spec)
}

type linter struct {
	result diagnostics.Result
}

func (l *linter) Lint(spec *metadata.Specification) diagnostics.Result {
	// (Re)start with an empty result.
	l.result = diagnostics.Result{}
	if len(spec.RemoteTestDrivers) == 0 {
		l.result.AppendError("Specification must contain at least one RemoteTestDriver")
		return l.result
	}
	for _, rtd := range spec.RemoteTestDrivers {
		l.lintRTD(rtd)
	}
	l.verifyUniqueRemoteTestDriverNames(spec.RemoteTestDrivers)
	return l.result
}

func (l *linter) verifyUniqueRemoteTestDriverNames(rtds []*metadata.RemoteTestDriver) {
	ns := make([]string, len(rtds))
	for i, rtd := range rtds {
		ns[i] = rtd.GetName()
	}
	if repeated := formatRepeated(ns); repeated != "" {
		l.result.AppendError("RemoteTestDriver names must be unique, found repeated name(s): %s", repeated)
	}
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

func (l *linter) lintRTD(rtd *metadata.RemoteTestDriver) {
	l.lintRTDName(rtd.GetName())
	for _, t := range rtd.Tests {
		l.lintTest(t, rtd.GetName())
	}
	drop := l.result.PushContext(fmt.Sprintf("RemoteTestDriver '%s'", rtd.GetName()))
	defer drop()

	l.verifyUniqueTestNames(rtd.Tests)
}

func (l *linter) verifyUniqueTestNames(tests []*metadata.Test) {
	ns := make([]string, len(tests))
	for i, test := range tests {
		ns[i] = test.GetName()
	}
	if repeated := formatRepeated(ns); repeated != "" {
		l.result.AppendError("Test names must be unique, found repeated name(s): %s", repeated)
	}
}

const (
	rtdCollection  = "remoteTestDrivers"
	testCollection = "tests"
)

func (l *linter) lintRTDName(name string) {
	drop := l.result.PushContext(fmt.Sprintf("RemoteTestDriver '%s'", name))
	defer drop()

	if !l.lintResourceName(name) {
		return
	}

	parts := strings.SplitN(name, "/", 3)
	switch len(parts) {
	case 0:
		l.result.AppendError("name must be of the form '%s/{remoteTestDriver}', found empty string", rtdCollection)
	case 1:
		if parts[0] == rtdCollection {
			l.result.AppendError("name must be of the form '%s/{remoteTestDriver}', missing name after '%s'", rtdCollection, rtdCollection)
		} else {
			l.result.AppendError("name must be of the form '%s/{remoteTestDriver}', missing prefix '%s'", rtdCollection, rtdCollection)
		}
	case 2:
		if parts[0] != rtdCollection {
			l.result.AppendError("name must be of the form '%s/{remoteTestDriver}', missing prefix '%s'", rtdCollection, rtdCollection)
		}
	default:
		l.result.AppendError("name must be of the form '%s/{remoteTestDriver}', found trailing suffix '%s'", rtdCollection, parts[2])
	}
}

func (l *linter) lintTest(test *metadata.Test, rtdName string) {
	drop := l.result.PushContext(fmt.Sprintf("Test '%s'", test.GetName()))
	defer drop()

	l.lintTestName(test.GetName(), rtdName)
	if test.DutConstraint != nil {
		l.lintDutConstraint(test.DutConstraint)
	}
}

func (l *linter) lintTestName(name string, rtdName string) {
	if !l.lintResourceName(name) {
		return
	}
	prefix := fmt.Sprintf("%s/", rtdName)
	if !strings.HasPrefix(name, prefix) {
		l.result.AppendError("name must be prefixed with RemoteTestDriver name '%s'", rtdName)
		return
	}
	name = strings.TrimPrefix(name, prefix)
	parts := strings.SplitN(name, "/", 3)
	switch len(parts) {
	case 0:
		l.result.AppendError("name must be of the form '%s/{test}', found empty string", testCollection)
	case 1:
		if parts[0] == testCollection {
			l.result.AppendError("name must be of the form '%s/{test}', missing name after '%s'", testCollection, testCollection)
		} else {
			l.result.AppendError("name must be of the form '%s/{test}', missing prefix '%s'", testCollection, testCollection)
		}
	case 2:
		if parts[0] != testCollection {
			l.result.AppendError("name must be of the form '%s/{test}', missing prefix '%s'", testCollection, testCollection)
		}
	default:
		l.result.AppendError("name must be of the form '%s/{test}', found trailing suffix '%s'", testCollection, parts[2])
	}
}

// lintResourceName lints resource names.
//
// This lint enforces some rules in addition to the recommendations in
// https://aip.dev/122.
//
// Returns true if argument was a valid resource name, false otherwise.
// Callers should not further lint an invalid resource name because it usually
// leads to spurious diagnostic messages.
//
// The argument _is not_ added as a context in diagnostic messages, because the
// caller can provide better context about the object being named (e.g.
// "RemoteTestDriver <name>" instead of "<name>").
func (l *linter) lintResourceName(name string) bool {
	result := diagnostics.Result{}
	defer func() {
		l.result.Merge(result)
	}()

	if name == "" {
		result.AppendError("name must be non-empty (https://aip.dev/122)")
		return result.IsValid()
	}

	u, err := url.Parse(name)
	if err != nil {
		drop := l.result.PushContext("parse name")
		defer drop()

		result.AppendError(err.Error())
		return result.IsValid()
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
		return result.IsValid()
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
	return result.IsValid()
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func (l *linter) lintDutConstraint(cond *metadata.DUTConstraint) {
	drop := l.result.PushContext("dut_constraint")
	defer drop()

	if cond.Setup == nil && cond.Config == nil {
		l.result.AppendError("some constraint must be set")
		return
	}
	if cond.Setup != nil {
		l.lintDUTSetupConstraint(cond.Setup)
	}
	if cond.Config != nil {
		l.lintDUTConfigConstraint(cond.Config)
	}

}

func (l *linter) lintDUTSetupConstraint(cond *metadata.DUTSetupConstraint) {
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

	drop := l.result.PushContext("setup")
	defer drop()

	l.parseAndCheckExpression(env, cond.Expression)
}

func (l *linter) parseAndCheckExpression(env *cel.Env, expr string) {
	if expr == "" {
		l.result.AppendError("expression must be non-empty")
		return
	}
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		// Reported issues are frequently displayed on multiple lines.
		// Adding a leading newline makes the multi-line display easier to read.
		l.result.AppendError("compile expression: \n%s", iss.String())
		return
	}
	if ast.ResultType().GetPrimitive() != exprpb.Type_BOOL {
		l.result.AppendError("expression must evaluate to a boolean, found %s", ast.ResultType().String())
	}
}

func (l *linter) lintDUTConfigConstraint(cond *metadata.DUTConfigConstraint) {
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

	drop := l.result.PushContext("config")
	defer drop()

	l.parseAndCheckExpression(env, cond.Expression)
}
