// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/luci/common/errors"
)

// UnmarshalBBRecord takes a bigquery.Value and returns a parsed Skylab test result.
// If the message is empty, successfully return nothing.
func UnmarshalBBRecord(encodedData string) (*skylab_test_runner.Result, error) {
	compressed, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, errors.Annotate(err, "query::UnmarshalBBRecord: decode from base64").Err()
	}
	reader, err := zlib.NewReader(bytes.NewBuffer(compressed))
	if err != nil {
		return nil, errors.Annotate(err, "query::UnmarshalBBRecord: create zlib reader").Err()
	}
	encoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Annotate(err, "query::UnmarshalBBRecord: reading").Err()
	}
	var result skylab_test_runner.Result
	if err := proto.Unmarshal(encoded, &result); err != nil {
		return nil, errors.Annotate(err, "query::UnmarshalBBRecord: unmarshal proto").Err()
	}
	return &result, nil
}

// BBRecordMarshaler marshals buildbucket records as JSON with a four-space indent.
var bbRecordMarshaler = &jsonpb.Marshaler{
	Indent: "    ",
}

// JSONEncodeBBRecord takes a Skylab test runner result and encodes it as JSON.
func JSONEncodeBBRecord(res *skylab_test_runner.Result) (string, error) {
	msg, err := bbRecordMarshaler.MarshalToString(res)
	if err != nil {
		return "", err
	}
	return msg, nil
}

// ExtractGSURL is a convenience method that extracts the Google storage root URL
// from a skylab test runner result.
func ExtractGSURL(res *skylab_test_runner.Result) (string, error) {
	if res == nil {
		return "", errors.New("ExtractGSUrl: res cannot be nil")
	}
	return res.GetLogData().GetGsUrl(), nil
}

// ExtractStainlessURL is a convenience method that extracts the Stainless URL
// from a skylab test runner result.
func ExtractStainlessURL(res *skylab_test_runner.Result) (string, error) {
	if res == nil {
		return "", errors.New("ExtractGSUrl: res cannot be nil")
	}
	return res.GetLogData().GetStainlessUrl(), nil
}
