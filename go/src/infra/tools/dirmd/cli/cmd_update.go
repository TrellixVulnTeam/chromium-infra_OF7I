// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"regexp"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/tools/dirmd/cli/updater"
)

var bqTableRe = regexp.MustCompile(`^([^.]+)\.([^.]+)\.([^.]+)$`)

func cmdChromiumUpdate(p *Params) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `chromium-update`,
		ShortDesc: "INTERNAL tool. Do not use it unless you know what you are doing.",
		Advanced:  true,
		CommandRun: func() subcommands.CommandRun {
			r := &chromiumUpdateRun{params: p}
			r.Flags.StringVar(&r.ChromiumCheckout, "chromium-checkout", "", "Path to the chromium/src.git checkout")
			r.Flags.StringVar(&r.OutDir, "out-dir", "", "Path to a directory where to write output files")
			r.Flags.StringVar(&r.bucket, "bucket", "", "Name of the bucket where to upload metadata")
			r.Flags.StringVar(&r.bucketLegacy, "bucket-legacy", "", "Name of the bucket where to upload metadata in legacy format")
			r.Flags.StringVar(&r.bqTable, "bigquery-table", "", `Name of the bigquery table (in the format of "<cloud_project>.<dataset>.<table>") where to upload metadata`)
			r.Flags.StringVar(&r.gitHost, "git-host", "", `Name of the git host`)
			r.Flags.StringVar(&r.gitProject, "git-project", "", `Name of the git project`)
			r.Flags.StringVar(&r.ref, "ref", "", `commit ref`)
			r.Flags.StringVar(&r.revision, "revision", "", `commit HEX SHA1`)
			return r
		},
	}
}

type chromiumUpdateRun struct {
	baseCommandRun
	params *Params
	updater.Updater
	bucket       string
	bucketLegacy string
	bqTable      string

	// git info
	gitHost    string
	gitProject string
	ref        string
	revision   string
}

func (r *chromiumUpdateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)

	if r.ChromiumCheckout == "" {
		return r.done(ctx, errors.Reason("-chromium-checkout is required").Err())
	}
	if r.OutDir == "" && r.bucket == "" && r.bucketLegacy == "" && r.bqTable == "" {
		return r.done(ctx, errors.Reason("at least one of -out-dir, -bucket, -bucket-legacy, or -bigquery-table is required").Err())
	}
	if r.bqTable != "" && !bqTableRe.MatchString(r.bqTable) {
		return r.done(ctx, errors.Reason("-bigquery-table does not match <cloud_project>.<dataset>.<table>").Err())
	}

	if r.bqTable != "" && (r.gitHost == "" || r.gitProject == "" || r.ref == "" || r.revision == "") {
		return r.done(ctx, errors.Reason("-bigquery-table requires git commit information").Err())
	}

	if r.bucket == "" && r.bucketLegacy == "" && r.bqTable == "" {
		return r.done(ctx, r.Updater.Run(ctx))
	}

	ts, err := r.newTokenSource(ctx)
	if err != nil {
		return r.done(ctx, errors.Annotate(err, "failed to create a token source").Err())
	}

	if r.bucket != "" || r.bucketLegacy != "" {
		gcs, err := storage.NewClient(ctx, option.WithTokenSource(ts))
		if err != nil {
			return r.done(ctx, errors.Annotate(err, "failed to create a GCS client").Err())
		}
		if r.bucket != "" {
			r.GCSBucket = gcs.Bucket(r.bucket)
		}
		if r.bucketLegacy != "" {
			r.GCSBucketLegacy = gcs.Bucket(r.bucketLegacy)
		}
	}

	if r.bqTable != "" {
		m := bqTableRe.FindStringSubmatch(r.bqTable)
		if m == nil || len(m) != 4 {
			return r.done(ctx, errors.Reason("-bigquery-table does not match <cloud_project>.<dataset>.<table>").Err())
		}
		bc, err := bigquery.NewClient(ctx, m[1], option.WithTokenSource(ts))
		if err != nil {
			r.done(ctx, errors.Annotate(err, "failed to create a bigquery client").Err())
		}
		r.BqTable = bc.Dataset(m[2]).Table(m[3])
	}

	r.Commit = &updater.GitCommit{
		Host:     r.gitHost,
		Project:  r.gitProject,
		Ref:      r.ref,
		Revision: r.revision,
	}

	return r.done(ctx, r.Updater.Run(ctx))
}

func (r *chromiumUpdateRun) newTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	authenticator := auth.NewAuthenticator(ctx, auth.SilentLogin, r.params.Auth)
	return authenticator.TokenSource()
}
