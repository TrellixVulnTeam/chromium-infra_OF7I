// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

import (
	"os"

	"infra/cros/cmd/lucifer/internal/logdog"
)

func ExampleClient_step_tracking() {
	lg := logdog.NewTextLogger(os.Stdout)
	logdog.ConfigForTest(lg)
	c := Client{
		logger: lg,
	}
	s := c.Step("step1")
	defer s.Close()
	func(c *Client) {
		s := c.Step("substep2")
		defer s.Close()
		func(c *Client) {
			s := c.Step("substep3")
			defer s.Close()
		}(c)
		func(c *Client) {
			s := c.Step("substep4")
			defer s.Close()
		}(c)
	}(&c)
	// Output:
	// example: STEP step1
	// example: STEP step1::substep2
	// example: STEP step1::substep2::substep3
	// example: STEP step1::substep2::substep3 OK
	// example: STEP step1::substep2::substep4
	// example: STEP step1::substep2::substep4 OK
	// example: STEP step1::substep2 OK
	// example: STEP step1 OK
}
