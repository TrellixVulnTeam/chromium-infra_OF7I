// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

const programName = "stable_version2"
const omahaStatusFile = "omaha_status.json"
const omahaGsPath = "gs://chromeos-build-release-console/omaha_status.json"

var (
	unmarshaller = jsonpb.Unmarshaler{AllowUnknownFields: true}
	marshaller   = jsonpb.Marshaler{}
)

func printError(w io.Writer, err error) {
	fmt.Fprintf(w, "%s: %s\n", programName, err)
}

func setupLogging(ctx context.Context) context.Context {
	return logging.SetLevel(ctx, logging.Debug)
}

func newAuthenticatedTransport(ctx context.Context, f *authcli.Flags) (http.RoundTripper, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "create authenticated transport").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, o)
	return a.Transport()
}
