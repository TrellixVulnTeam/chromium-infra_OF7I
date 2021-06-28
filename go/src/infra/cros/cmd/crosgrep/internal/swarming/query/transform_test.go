// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// TestBBRecordRoundTrip tests that serializing and deserializing a record successfully
// produces the original output. Compressing and decompressing and encoding and
// decoding from base64, and converting to and from the proto wire format are all invertible
// transformations. Therefore their composition should be invertible too.
func TestBBRecordRoundTrip(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		data *skylab_test_runner.Result
	}{
		{
			name: "empty",
			data: &skylab_test_runner.Result{},
		},
		{
			name: "nontrivial harness",
			data: &skylab_test_runner.Result{
				Harness: &skylab_test_runner.Result_AutotestResult{},
			},
		},
		{
			name: "nontrivial autotest result",
			data: &skylab_test_runner.Result{
				Harness: &skylab_test_runner.Result_AutotestResult{
					AutotestResult: &skylab_test_runner.Result_Autotest{
						Incomplete: true,
						TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
							{
								Name:    "fake test",
								Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL,
							},
						},
					},
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			original := testCase.data
			encoded, err := marshalBBRecord(original)
			if err != nil {
				t.Error(err)
			}
			decoded, err := UnmarshalBBRecord(encoded)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(original, decoded, protocmp.Transform()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// MarshalBBRecord takes a Skylab test runner result and encodes it as a base64-encoded string
// containing a compressed proto.
func marshalBBRecord(testRunnerResult *skylab_test_runner.Result) (string, error) {
	if testRunnerResult == nil {
		return "", errors.New("SerializeBBRecord: res cannot be nil")
	}
	protoBytes, err := proto.Marshal(testRunnerResult)
	if err != nil {
		return "", errors.Annotate(err, "writing proto").Err()
	}
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(protoBytes); err != nil {
		panic("Writing to in-memory buffer should not fail!")
	}
	if err := writer.Close(); err != nil {
		panic("Closing in-memory buffer should not fail!")
	}
	return base64.StdEncoding.EncodeToString(compressed.Bytes()), nil
}
