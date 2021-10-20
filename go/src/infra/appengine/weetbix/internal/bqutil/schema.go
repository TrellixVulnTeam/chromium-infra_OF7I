// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bqutil

import (
	"cloud.google.com/go/bigquery"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/google/descutil"

	desc "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// GenerateSchema generates BigQuery schema for the given proto message
// using the given set of message definitions.
func GenerateSchema(fdset *desc.FileDescriptorSet, message string) (schema bigquery.Schema, err error) {
	conv := bq.SchemaConverter{
		Desc:           fdset,
		SourceCodeInfo: make(map[*desc.FileDescriptorProto]bq.SourceCodeInfoMap, len(fdset.File)),
	}
	for _, f := range fdset.File {
		conv.SourceCodeInfo[f], err = descutil.IndexSourceCodeInfo(f)
		if err != nil {
			return nil, errors.Annotate(err, "failed to index source code info in file %q", f.GetName()).Err()
		}
	}
	schema, _, err = conv.Schema(message)
	return schema, err
}
