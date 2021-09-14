// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This is the test-dev-karte script. It opens a connection to the dev Karte instance
// and creates an action. This script exists to aid local development and to debug problems
// in an environment similar to the production environment.
package main

import (
	"context"
	"fmt"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/client"
	"infra/cros/karte/internal/site"
)

// Main creates a single unimportant action in the dev karte instance for testing
// purposes.
func main() {
	ctx := context.Background()
	kClient, err := client.NewClient(ctx, client.DevConfig(site.DefaultAuthOptions))
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	_, err = kClient.CreateAction(ctx, &kartepb.CreateActionRequest{
		Action: &kartepb.Action{
			Kind: "foo",
		},
	})

	if err != nil {
		fmt.Printf("%s\n", err)
	}
}
