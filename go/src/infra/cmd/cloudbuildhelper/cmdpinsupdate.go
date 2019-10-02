// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"sync"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cloudbuildhelper/dockerfile"
	"infra/cmd/cloudbuildhelper/registry"
)

var cmdPinsUpdate = &subcommands.Command{
	UsageLine: "pins-update <pins-yaml-path>",
	ShortDesc: "updates digests in the image pins YAML file",
	LongDesc: `Updates digests in the image pins YAML file.

Resolves tags in all entries not marked as frozen and writes new SHA256 digests
back into the file.

To freeze an entry (and thus exclude it from the update process) add "freeze"
field specifying the reason why it is frozen.

Rewrites the YAML file destroying any custom formatting or comments there.
If you want to comment an entry, manually add "comment" field.
`,

	CommandRun: func() subcommands.CommandRun {
		c := &cmdPinsUpdateRun{}
		c.init()
		return c
	},
}

type cmdPinsUpdateRun struct {
	commandBase

	pins string
}

func (c *cmdPinsUpdateRun) init() {
	c.commandBase.init(c.exec, true, true, []*string{
		&c.pins,
	})
}

func (c *cmdPinsUpdateRun) exec(ctx context.Context) error {
	pins, err := readPins(c.pins)
	if err != nil {
		return err
	}

	ts, err := c.tokenSource(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to setup auth").Err()
	}
	registry := &registry.Client{TokenSource: ts}

	var m sync.Mutex
	var out struct {
		Unchanged []string `json:"unchanged,omitempty"`
		Updated   []string `json:"updated,omitempty"`
		Skipped   []string `json:"skipped,omitempty"`
		Failed    []string `json:"failed,omitempty"`
	}

	report := func(s *[]string, p *dockerfile.Pin) {
		m.Lock()
		defer m.Unlock()
		*s = append(*s, p.ImageRef())
	}

	err = pins.Visit(func(p *dockerfile.Pin) error {
		if p.Freeze != "" {
			logging.Infof(ctx, "Skipping frozen %s: %s", p.ImageRef(), p.Freeze)
			report(&out.Skipped, p)
			return nil
		}
		switch resolved, err := registry.GetImage(ctx, p.ImageRef()); {
		case err != nil:
			logging.Errorf(ctx, "When resolving %s: %s", p.ImageRef(), err)
			report(&out.Failed, p)
			return err
		case resolved.Digest != p.Digest:
			logging.Infof(ctx, "Updating %s: %s -> %s", p.ImageRef(), p.Digest, resolved.Digest)
			p.Digest = resolved.Digest
			report(&out.Updated, p)
		default:
			report(&out.Unchanged, p)
		}
		return nil
	})

	logging.Infof(ctx, "Summary:")
	logging.Infof(ctx, "    Unchanged: %d", len(out.Unchanged))
	logging.Infof(ctx, "    Updated:   %d", len(out.Updated))
	logging.Infof(ctx, "    Skipped:   %d", len(out.Skipped))
	logging.Infof(ctx, "    Failed:    %d", len(out.Failed))

	if jerr := c.writeJSONOutput(&out); jerr != nil {
		return errors.Annotate(jerr, "failed to write JSON output").Err()
	}
	if err != nil {
		return errors.Annotate(err, "failed to resolve pin(s)").Err()
	}
	if len(out.Updated) != 0 {
		return errors.Annotate(writePins(c.pins, pins), "writing pins file").Err()
	}
	return nil
}
