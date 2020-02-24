// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmdlib

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/libs/skylab/common/errctx"
)

// JSONPBMarshaller marshals protobufs as JSON.
var JSONPBMarshaller = &jsonpb.Marshaler{
	EmitDefaults: true,
}

// JSONPBUnmarshaller unmarshals JSON and creates corresponding protobufs.
var JSONPBUnmarshaller = jsonpb.Unmarshaler{
	AllowUnknownFields: true,
}

type commonFlags struct {
	debug bool
}

// Register adds shared flags.
func (f *commonFlags) Register(fl *flag.FlagSet) {
	fl.BoolVar(&f.debug, "debug", false, "Enable debug output.")
}

func (f commonFlags) DebugLogger(a subcommands.Application) *log.Logger {
	out := ioutil.Discard
	if f.debug {
		out = a.GetErr()
	}
	return log.New(out, a.GetName(), log.LstdFlags|log.Lshortfile)
}

// NewAuthenticator creates a new authenticator based on flags.
func NewAuthenticator(ctx context.Context, f *authcli.Flags) (*auth.Authenticator, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "create authenticator").Err()
	}
	return auth.NewAuthenticator(ctx, auth.SilentLogin, o), nil
}

// NewHTTPClient returns an HTTP client with authentication set up.
func NewHTTPClient(ctx context.Context, f *authcli.Flags) (*http.Client, error) {
	a, err := NewAuthenticator(ctx, f)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create HTTP client").Err()
	}
	c, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "failed to create HTTP client").Err()
	}
	return c, nil
}

// NewAuthenticatedTransport creates a new authenticated transport
func NewAuthenticatedTransport(ctx context.Context, f *authcli.Flags) (http.RoundTripper, error) {
	at, err := NewAuthenticator(ctx, f)
	if err != nil {
		return nil, errors.Annotate(err, "create authenticated transport").Err()
	}
	return at.Transport()
}

// UserErrorReporter reports a detailed error message to the user.
//
// PrintError() uses a UserErrorReporter to print multi-line user error details
// along with the actual error.
type UserErrorReporter interface {
	// Report a user-friendly error through w.
	ReportUserError(w io.Writer)
}

// PrintError reports errors back to the user.
//
// Detailed error information is printed if err is a UserErrorReporter.
func PrintError(a subcommands.Application, err error) {
	if u, ok := err.(UserErrorReporter); ok {
		u.ReportUserError(a.GetErr())
	} else {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
	}
}

// NewUsageError creates a new error that also reports flags usage error
// details.
func NewUsageError(flags flag.FlagSet, format string, a ...interface{}) error {
	return &usageError{
		error: fmt.Errorf(format, a...),
		flags: flags,
	}
}

type usageError struct {
	error
	flags flag.FlagSet
}

func (e *usageError) ReportUserError(w io.Writer) {
	fmt.Fprintf(w, "%s\n\nUsage:\n\n", e.error)
	e.flags.Usage()
}

// MaybeWithTimeout creates a new context that has a timeout if the provided timeout is positive.
func MaybeWithTimeout(ctx context.Context, timeoutMins int) (context.Context, func(error)) {
	if timeoutMins >= 0 {
		return errctx.WithTimeout(ctx, time.Duration(timeoutMins)*time.Minute,
			fmt.Errorf("timed out after %d minutes while waiting for task(s) to complete", timeoutMins))
	}
	return errctx.WithCancel(ctx)
}
