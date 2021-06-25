// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"fmt"
	"net/http"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/karte/site"
)

// CheckServer checks the status of the Karte server.
var CheckServer = &subcommands.Command{
	UsageLine: `check-server`,
	ShortDesc: `health check for server`,
	LongDesc:  `Check the health of the Karte server.`,
	CommandRun: func() subcommands.CommandRun {
		r := &checkServerRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		// TODO(gregorynisbet): add envFlags.
		return r
	},
}

// CheckServerRun stores the arguments for the check-server command.
// Its lifetime is the lifetime of the check-server command.
type checkServerRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run runs the check-server command and returns an exit status.
func (c *checkServerRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun runs the check-server command and returns a go-level error.
func (c *checkServerRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	// TODO(gregorynisbet): Replace hardcoded Karte server with arguments.
	resp, err := http.Get("http://localhost:8800/hello-world")
	if err != nil {
		return errors.Annotate(err, "get request").Err()
	}
	switch resp.StatusCode {
	case 200:
		fmt.Fprintf(a.GetOut(), "200\n")
		return nil
	}
	_, err = fmt.Fprintf(a.GetOut(), "%#v", resp)
	if err != nil {
		return errors.Annotate(err, "printing HTTP response").Err()
	}
	return fmt.Errorf("check-server: bad status code %d", resp.StatusCode)
}
