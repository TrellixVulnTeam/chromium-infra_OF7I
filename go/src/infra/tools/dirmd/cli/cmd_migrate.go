// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/sync/parallel"

	"infra/tools/dirmd"
)

func cmdMigrate() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `migrate DIR1 [DIR2...]`,
		ShortDesc: "migrate metadata in the given directories",
		LongDesc: text.Doc(`
			Migrate metadata in the given directories.

			Move metadata from OWNERS to DIR_METADATA files in the given directories
			and all subdirectories.
		`),
		CommandRun: func() subcommands.CommandRun {
			return &migrateRun{}
		},
	}
}

type migrateRun struct {
	baseCommandRun
}

func (r *migrateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return r.done(ctx, r.run(ctx, args))
}

func (r *migrateRun) run(ctx context.Context, dirs []string) error {
	return parallel.WorkPool(16, func(workC chan<- func() error) {
		for _, dir := range dirs {
			err := filepath.Walk(dir, func(dir string, info os.FileInfo, err error) error {
				switch {
				case err != nil:
					return err
				case !info.IsDir():
					return nil
				}
				workC <- func() error {
					return dirmd.MigrateMetadata(dir)
				}
				return nil
			})
			if err != nil {
				workC <- func() error {
					return err
				}
			}
		}
	})
}
