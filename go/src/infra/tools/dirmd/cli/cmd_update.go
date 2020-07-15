// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/tools/dirmd/cli/updater"
)

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
}

func (r *chromiumUpdateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)

	if r.ChromiumCheckout == "" {
		return r.done(ctx, errors.Reason("-chromium-checkout is required").Err())
	}
	if r.OutDir == "" && r.bucket == "" && r.bucketLegacy == "" {
		return r.done(ctx, errors.Reason("at least one of -out-dir, -bucket or -bucket-legacy is required").Err())
	}

	if r.bucket != "" || r.bucketLegacy != "" {
		gcs, err := r.newGCSClient(ctx)
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

	return r.done(ctx, r.Updater.Run(ctx))
}

func (r *chromiumUpdateRun) newGCSClient(ctx context.Context) (*storage.Client, error) {
	authenticator := auth.NewAuthenticator(ctx, auth.SilentLogin, r.params.Auth)
	ts, err := authenticator.TokenSource()
	if err != nil {
		return nil, err
	}
	return storage.NewClient(ctx, option.WithTokenSource(ts))
}
