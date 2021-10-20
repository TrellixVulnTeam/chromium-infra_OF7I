// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package pbutil contains methods for manipulating Weetbix protos.
package pbutil

import (
	pb "infra/appengine/weetbix/proto/v1"

	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
)

// VariantFromResultDB returns a Weetbix TestResultId corresponding to the
// supplied ResultDB test result name.
// The format of name should be:
// "invocations/{INVOCATION_ID}/tests/{URL_ESCAPED_TEST_ID}/results/{RESULT_ID}".
func TestResultIDFromResultDB(name string) *pb.TestResultId {
	return &pb.TestResultId{System: "resultdb", Id: name}
}

// VariantFromResultDB returns a Weetbix Variant corresponding to the
// supplied ResultDB Variant.
func VariantFromResultDB(v *rdbpb.Variant) *pb.Variant {
	if v == nil {
		// Variant is optional in ResultDB.
		return &pb.Variant{Def: make(map[string]string)}
	}
	return &pb.Variant{Def: v.Def}
}

// VariantFromResultDB returns a Weetbix FailureReason corresponding to the
// supplied ResultDB FailureReason.
func FailureReasonFromResultDB(fr *rdbpb.FailureReason) *pb.FailureReason {
	if fr == nil {
		return nil
	}
	return &pb.FailureReason{
		PrimaryErrorMessage: fr.PrimaryErrorMessage,
	}
}
