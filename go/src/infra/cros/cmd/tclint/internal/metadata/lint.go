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

	"go.chromium.org/chromiumos/config/go/api/test/metadata/v1"
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
	return result
}

func lintRTD(rtd *metadata.RemoteTestDriver) Result {
	return lintRTDName(rtd.GetName())
}

const (
	rtdCollection = "remoteTestDrivers"
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
