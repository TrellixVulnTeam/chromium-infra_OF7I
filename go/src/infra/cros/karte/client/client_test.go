// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	context "context"
	"testing"
)

// Test that insantiating a new client with an empty config fails.
func TestNewClient(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, nil)

	if client != nil {
		t.Error("client unexpectedly created!")
	}
	if err == nil {
		t.Error("expected creating client to fail", err)
	}
}
