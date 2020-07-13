// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/logging"

	"infra/tools/dirmeta"
)

// baseCommandRun provides common command run functionality.
// All dirmeta subcommands must embed it directly or indirectly.
type baseCommandRun struct {
	subcommands.CommandRunBase

	// out is the filename for the output.
	// Special value "-" means stdout.
	out string
}

func (r *baseCommandRun) RegisterOutputFlag() {
	r.Flags.StringVar(&r.out, "out", "-", `Path to the output file. If "-", then print the output to stdout`)
}

func (r *baseCommandRun) done(ctx context.Context, err error) int {
	if err != nil {
		logging.Errorf(ctx, "%s", err)
		return 1
	}
	return 0
}

func (r *baseCommandRun) writeMapping(m *dirmeta.Mapping) error {
	data, err := protojson.Marshal(m.Proto())
	if err != nil {
		return err
	}

	return r.writeTextOutput(data)
}

func (r *baseCommandRun) writeTextOutput(data []byte) error {
	out := os.Stdout
	if r.out != "-" {
		var err error
		if out, err = os.Create(r.out); err != nil {
			return err
		}
		defer out.Close()
	}
	if _, err := out.Write(data); err != nil {
		return err
	}

	if len(data) > 0 && data[len(data)-1] != '\n' {
		out.WriteString("\n")
	}
	return nil
}
