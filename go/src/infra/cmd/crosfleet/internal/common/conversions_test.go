// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
)

func TestMapToStruct(t *testing.T) {
	mixedMap := map[string]interface{}{
		"string": "stringVal",
		"num":    1,
		"nestedMap": map[string]interface{}{
			"bool": true,
		},
		"protoMessage": (&test_platform.Request{
			Params: &test_platform.Request_Params{
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "foo-cros"}},
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "sample-image"}},
				},
			},
		}).ProtoReflect().Interface(), // Convert to protoreflect.ProtoMessage for easier type comparison
	}
	// This struct demonstrates the difficulties of use structpb.Struct, and
	// consequently the need for the MapToStruct() function.
	wantStruct := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"string": {
				Kind: &structpb.Value_StringValue{StringValue: "stringVal"},
			},
			"num": {
				Kind: &structpb.Value_NumberValue{NumberValue: 1},
			},
			"nestedMap": {
				Kind: &structpb.Value_StructValue{
					StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"bool": {
								Kind: &structpb.Value_BoolValue{BoolValue: true},
							},
						},
					},
				},
			},
			"protoMessage": {
				Kind: &structpb.Value_StructValue{
					StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"params": {
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"softwareDependencies": {
												Kind: &structpb.Value_ListValue{
													ListValue: &structpb.ListValue{
														Values: []*structpb.Value{
															{
																Kind: &structpb.Value_StructValue{
																	StructValue: &structpb.Struct{
																		Fields: map[string]*structpb.Value{
																			"chromeosBuild": {
																				Kind: &structpb.Value_StringValue{StringValue: "foo-cros"},
																			},
																		},
																	},
																},
															},
															{
																Kind: &structpb.Value_StructValue{
																	StructValue: &structpb.Struct{
																		Fields: map[string]*structpb.Value{
																			"chromeosBuild": {
																				Kind: &structpb.Value_StringValue{StringValue: "sample-image"},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	gotStruct, err := MapToStruct(mixedMap)
	if err != nil {
		t.Fatalf("unexpected error calling MapToStruct(%v): %s", mixedMap, err)
	}
	if diff := cmp.Diff(wantStruct, gotStruct, CmpOpts); diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}
