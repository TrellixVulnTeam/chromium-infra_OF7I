// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package transform contains tools for transforming CTP build
// to test_platform/analytics/TestPlanRun proto.
package transform

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
)

func inferExecutionURL(c *result_flow.BuildbucketConfig, bID int64) string {
	return fmt.Sprintf(
		"https://ci.chromium.org/p/%s/builders/%s/%s/b%d",
		c.GetProject(),
		c.GetBucket(),
		c.GetBuilder(),
		bID,
	)
}

func getOutputPropertiesValue(b *bbpb.Build, field string) (*structpb.Value, bool) {
	op := b.GetOutput().GetProperties().GetFields()
	if op == nil {
		return nil, false
	}
	v, ok := op[field]
	return v, ok
}

func unmarshalStructPB(from *structpb.Value, to proto.Message) (proto.Message, error) {
	m := jsonpb.Marshaler{}
	json, err := m.MarshalToString(from)
	if err != nil {
		return nil, errors.Annotate(err, "unmarshal Struct PB").Err()
	}
	if err := jsonpb.UnmarshalString(json, to); err != nil {
		return nil, errors.Annotate(err, "unmarshal Struct PB").Err()
	}
	return to, nil
}

func structPBStructToMap(from *structpb.Value) (map[string]*structpb.Value, error) {
	switch m := from.Kind.(type) {
	case *structpb.Value_StructValue:
		if m.StructValue == nil {
			return nil, errors.Reason("struct has no fields").Err()
		}
		return m.StructValue.Fields, nil
	default:
		return nil, errors.Reason("not a Struct type").Err()
	}
}

func unmarshalCompressedString(from string, to proto.Message) (proto.Message, error) {
	var err error
	if from == "" {
		return nil, nil
	}
	bs, err := base64.StdEncoding.DecodeString(from)
	if err != nil {
		return nil, errors.Annotate(err, "unmarshal compressed string to PB").Err()
	}
	reader, err := zlib.NewReader(bytes.NewReader(bs))
	if err != nil {
		return nil, errors.Annotate(err, "unmarshal compressed string to PB").Err()
	}
	bs, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Annotate(err, "unmarshal compressed string to PB").Err()
	}
	if err := proto.Unmarshal(bs, to); err != nil {
		return nil, errors.Annotate(err, "unmarshal compressed string to PB").Err()
	}
	return to, nil
}

// setDefaultStructValues defaults nil or empty values inside the given
// structpb.Struct. Needed because structpb.Value cannot be marshaled to JSON
// unless there is a kind set. More details are in crbug/1093683.
func setDefaultStructValues(s *structpb.Struct) {
	for k, v := range s.GetFields() {
		switch {
		case v == nil:
			s.Fields[k] = &structpb.Value{
				Kind: &structpb.Value_NullValue{},
			}
		case v.Kind == nil:
			v.Kind = &structpb.Value_NullValue{}
		case v.GetStructValue() != nil:
			setDefaultStructValues(v.GetStructValue())
		}
	}
}
