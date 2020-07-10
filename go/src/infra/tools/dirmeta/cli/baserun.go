// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/logging"
)

// baseCommandRun provides common command run functionality.
// All dirmeta subcommands must embed it directly or indirectly.
type baseCommandRun struct {
	subcommands.CommandRunBase
}

func (r *baseCommandRun) done(ctx context.Context, err error) int {
	if err != nil {
		logging.Errorf(ctx, "%s", err)
		return 1
	}
	return 0
}
