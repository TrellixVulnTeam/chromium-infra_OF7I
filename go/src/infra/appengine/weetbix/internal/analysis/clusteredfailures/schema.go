// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clusteredfailures

import (
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/golang/protobuf/descriptor"
	desc "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"infra/appengine/weetbix/internal/bqutil"
	bqpb "infra/appengine/weetbix/proto/bq"
	pb "infra/appengine/weetbix/proto/v1"
)

const partitionExpirationTime = 540 * 24 * time.Hour

const rowMessage = "weetbix.bq.ClusteredFailureRow"

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
			Fields: []string{"cluster_algorithm", "cluster_id", "test_result_system", "test_result_id"},
		},
		// Relax ensures no fields are marked "required".
		Schema: schema.Relax(),
	}
}

func generateRowSchema() (schema bigquery.Schema, err error) {
	fd, _ := descriptor.MessageDescriptorProto(&bqpb.ClusteredFailureRow{})
	// We also need to get FileDescriptorProto for StringPair, BugTrackingComponent, FailureReason
	// and PresubmitRunId because they are defined in different files.
	fdsp, _ := descriptor.MessageDescriptorProto(&pb.StringPair{})
	fdbtc, _ := descriptor.MessageDescriptorProto(&pb.BugTrackingComponent{})
	fdfr, _ := descriptor.MessageDescriptorProto(&pb.FailureReason{})
	fdprid, _ := descriptor.MessageDescriptorProto(&pb.PresubmitRunId{})
	fdset := &desc.FileDescriptorSet{File: []*desc.FileDescriptorProto{fd, fdsp, fdbtc, fdfr, fdprid}}
	return bqutil.GenerateSchema(fdset, rowMessage)
}
