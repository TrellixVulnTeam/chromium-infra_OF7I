// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"

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
