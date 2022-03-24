// Copyright 2022 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package updater

import (
	"context"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/descriptor"
	desc "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/googleapi"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/google/descutil"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/retry/transient"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
)

// Recommended rows per stream insert request.
// https://cloud.google.com/bigquery/quotas#streaming_inserts
const maxBatchRowCount = 500

const partitionExpirationTime = 540 * 24 * time.Hour // ~1.5y

const chromiumProject = "chromium/src"

// bqWrite ensures the provided BigQuery table then stream inserts metadata BigQuery rows to it.
// It's a noop if no BigQuery table is provided.
func (u *Updater) bqWrite(ctx context.Context, mapping *dirmd.Mapping) error {
	if u.BqTable == nil {
		return nil
	}
	schema, err := generateSchema()
	if err != nil {
		return err
	}
	tblMD := &bigquery.TableMetadata{
		TimePartitioning: &bigquery.TimePartitioning{
			Field:      "partition_time",
			Expiration: partitionExpirationTime,
		},
		Schema: schema,
	}
	if err := bq.EnsureTable(ctx, u.BqTable, tblMD); err != nil {
		return err
	}
	return writeToBQ(ctx, u.BqTable.Inserter(), mapping, u.Commit)
}

func generateSchema() (schema bigquery.Schema, err error) {
	fd, _ := descriptor.MessageDescriptorProto(&dirmdpb.DirBQRow{})
	fdmr, _ := descriptor.MessageDescriptorProto(&dirmdpb.Monorail{})
	fdwpt, _ := descriptor.MessageDescriptorProto(&dirmdpb.WPT{})
	fdbug, _ := descriptor.MessageDescriptorProto(&dirmdpb.Buganizer{})

	fdset := &desc.FileDescriptorSet{File: []*desc.FileDescriptorProto{fd, fdmr, fdwpt, fdbug}}
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
	schema, _, err = conv.Schema("chrome.dir_metadata.DirBQRow")
	return schema, err
}

// inserter is implemented by bigquery.Inserter.
// Added to make unit tests easier.
type inserter interface {
	// Put uploads one or more rows to the BigQuery service.
	Put(ctx context.Context, src interface{}) error
}

// writeToBQ writes rows to BigQuery in batches.
func writeToBQ(ctx context.Context, ins inserter, mapping *dirmd.Mapping, commit *GitCommit) error {
	batchC := make(chan []*bq.Row)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// prepare rows
		defer close(batchC)
		return generateRows(ctx, mapping, commit, batchC)
	})

	eg.Go(func() error {
		// write rows
		return writeBQRows(ctx, ins, batchC)
	})
	return eg.Wait()
}

// Find the sub project of the directory.
// Examples:
// - If the root repo is "~/chromium/src" and it contains subrepo
//   "~/chromium/src/v8", then Mapping.repos will have keys "." and "v8".
//   In this case Mapping.dirs key "foo/bar" will correspond to Repo key ".",
//   while "v8/baz" will correspond to Repo key "v8".
func subRepo(dir string, mapping *dirmd.Mapping) string {
	if dir == "." {
		return ""
	}
	parts := strings.Split(dir, "/")
	if _, ok := mapping.Repos[parts[0]]; ok {
		return parts[0]
	}
	return ""
}

func generateRows(ctx context.Context, mapping *dirmd.Mapping, commit *GitCommit, batchC chan []*bq.Row) error {
	partitionTime := timestamppb.New(clock.Now(ctx))
	rows := make([]*bq.Row, 0, maxBatchRowCount)

	for dir, md := range mapping.Dirs {
		row := &dirmdpb.DirBQRow{
			Source: &dirmdpb.Source{
				GitHost:  commit.Host,
				RootRepo: commit.Project,
				SubRepo:  subRepo(dir, mapping),
				Ref:      commit.Ref,
				Revision: commit.Revision,
			},
			Dir:             dir,
			Monorail:        md.Monorail,
			TeamEmail:       md.TeamEmail,
			Os:              md.Os,
			Buganizer:       md.Buganizer,
			BuganizerPublic: md.BuganizerPublic,
			TeamSpecificMetadata: &dirmdpb.TeamSpecific{
				Wpt: md.Wpt,
			},
			PartitionTime: partitionTime,
		}
		rows = append(rows, &bq.Row{Message: row})
		if len(rows) >= maxBatchRowCount {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case batchC <- rows:
			}
			rows = make([]*bq.Row, 0, maxBatchRowCount)
		}
	}
	if len(rows) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batchC <- rows:
		}
	}
	return nil
}

func hasReason(apiErr *googleapi.Error, reason string) bool {
	for _, e := range apiErr.Errors {
		if e.Reason == reason {
			return true
		}
	}
	return false
}

func writeBQRows(ctx context.Context, ins inserter, batchC chan []*bq.Row) error {
	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	for rows := range batchC {
		rows := rows
		eg.Go(func() error {
			err := retry.Retry(ctx, transient.Only(retry.Default), func() error {
				err := ins.Put(ctx, rows)
				switch e := err.(type) {
				case *googleapi.Error:
					if e.Code == http.StatusForbidden && hasReason(e, "quotaExceeded") {
						err = transient.Tag.Apply(err)
					}
				}
				return err
			}, retry.LogCallback(ctx, "bigquery_put"))
			return err
		})
	}
	return eg.Wait()
}
