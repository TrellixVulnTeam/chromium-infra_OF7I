// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

var (
	unmarshaller = jsonpb.Unmarshaler{AllowUnknownFields: true}
	marshaller   = jsonpb.Marshaler{}
)

func newAuthenticatedTransport(ctx context.Context, f *authcli.Flags) (http.RoundTripper, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "create authenticated transport").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, o)
	return a.Transport()
}

func readRequest(inFile string, request proto.Message) error {
	r, err := os.Open(inFile)
	if err != nil {
		return errors.Annotate(err, "read request").Err()
	}
	defer r.Close()
	if err := unmarshaller.Unmarshal(r, request); err != nil {
		return errors.Annotate(err, "read request").Err()
	}
	return nil
}

// exitCode computes the exit code for this tool.
func exitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case partialErrorTag.In(err):
		return 2
	default:
		return 1
	}
}

// writeResponseWithError writes response as JSON encoded protobuf to outFile.
//
// If errorSoFar is non-nil, this function considers the response to be partial
// and tags the returned error to that effect.
func writeResponseWithError(ctx context.Context, outFile string, response proto.Message, errorSoFar error) error {
	w, err := os.Create(outFile)
	if err != nil {
		return errors.MultiError{errorSoFar, errors.Annotate(err, "write response").Err()}
	}
	defer w.Close()
	if err := marshaller.Marshal(w, response); err != nil {
		return errors.MultiError{errorSoFar, errors.Annotate(err, "write response").Err()}
	}
	logResponse(ctx, response)
	return partialErrorTag.Apply(errorSoFar)
}

func logResponse(ctx context.Context, response proto.Message) {
	s, err := marshaller.MarshalToString(response)
	if err != nil {
		// It's not worth returning an error if this failed, as this is only for
		// debugging purposes.
		logging.Infof(ctx, "failed to marshal response for debug logging")
	} else {
		logging.Infof(ctx, "Wrote output:\n%s", s)
	}
}

// writeResponseWithError writes response as JSON encoded protobuf to outFile.
func writeResponse(ctx context.Context, outFile string, response proto.Message) error {
	return writeResponseWithError(ctx, outFile, response, nil)
}

// Use partialErrorTag to indicate when partial response is written to the
// output file. Use returnCode() to return the corresponding return code on
// process exit.
var partialErrorTag = errors.BoolTag{Key: errors.NewTagKey("partial results are available despite this error")}

func setupLogging(ctx context.Context) context.Context {
	return logging.SetLevel(ctx, logging.Info)
}

// logApplicationError logs the error returned to the entry function of an
// application.
func logApplicationError(ctx context.Context, a subcommands.Application, err error) {
	errors.Log(ctx, err)
	// Also log to error stream, since logs are directed at the main output
	// stream.
	fmt.Fprintf(a.GetErr(), "%s\n", err)
}
