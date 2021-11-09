// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/descriptor"
	desc "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"infra/appengine/weetbix/internal/bqutil"
	bqpb "infra/appengine/weetbix/proto/bq"
	pb "infra/appengine/weetbix/proto/v1"
)

const rowMessage = "weetbix.bq.TestVariantRow"

const partitionExpirationTime = 540 * 24 * time.Hour

var tableMetadata *bigquery.TableMetadata

func init() {
	var err error
	var schema bigquery.Schema
	if schema, err = generateRowSchema(); err != nil {
		panic(err)
	}
	tableMetadata = &bigquery.TableMetadata{
		TimePartitioning: &bigquery.TimePartitioning{
			Type:       bigquery.DayPartitioningType,
			Expiration: partitionExpirationTime,
			Field:      "partition_time",
		},
		Clustering: &bigquery.Clustering{
			Fields: []string{"partition_time", "realm", "test_id", "variant_hash"},
		},
		// Relax ensures no fields are marked "required".
		Schema: schema.Relax(),
	}
}

func generateRowSchema() (schema bigquery.Schema, err error) {
	fd, _ := descriptor.MessageDescriptorProto(&bqpb.TestVariantRow{})
	fdfs, _ := descriptor.MessageDescriptorProto(&pb.FlakeStatistics{})
	fdsp, _ := descriptor.MessageDescriptorProto(&pb.StringPair{})
	fdtmd, _ := descriptor.MessageDescriptorProto(&pb.TestMetadata{})
	fdtr, _ := descriptor.MessageDescriptorProto(&pb.TimeRange{})
	fdset := &desc.FileDescriptorSet{File: []*desc.FileDescriptorProto{fd, fdfs, fdsp, fdtmd, fdtr}}
	return bqutil.GenerateSchema(fdset, rowMessage)
}
