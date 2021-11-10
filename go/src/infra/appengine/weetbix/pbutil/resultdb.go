// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package pbutil contains methods for manipulating Weetbix protos.
package pbutil

import (
	pb "infra/appengine/weetbix/proto/v1"

	"go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
)

// TestResultIDFromResultDB returns a Weetbix TestResultId corresponding to the
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

// VariantToResultDB returns a ResultDB Variant corresponding to the
// supplied Weetbix Variant.
func VariantToResultDB(v *pb.Variant) *rdbpb.Variant {
	if v == nil {
		return &rdbpb.Variant{Def: make(map[string]string)}
	}
	return &rdbpb.Variant{Def: v.Def}
}

// VariantHash returns a hash of the variant.
func VariantHash(v *pb.Variant) string {
	return pbutil.VariantHash(VariantToResultDB(v))
}

// FailureReasonFromResultDB returns a Weetbix FailureReason corresponding to the
// supplied ResultDB FailureReason.
func FailureReasonFromResultDB(fr *rdbpb.FailureReason) *pb.FailureReason {
	if fr == nil {
		return nil
	}
	return &pb.FailureReason{
		PrimaryErrorMessage: fr.PrimaryErrorMessage,
	}
}

// TestMetadataFromResultDB converts a ResultDB TestMetadata to a Weetbix
// TestMetadata.
func TestMetadataFromResultDB(rdbTmd *rdbpb.TestMetadata) *pb.TestMetadata {
	if rdbTmd == nil {
		return nil
	}

	tmd := &pb.TestMetadata{
		Name: rdbTmd.Name,
	}
	loc := tmd.GetLocation()
	if loc != nil {
		tmd.Location = &pb.TestLocation{
			Repo:     loc.Repo,
			FileName: loc.FileName,
			Line:     loc.Line,
		}
	}

	return tmd
}
