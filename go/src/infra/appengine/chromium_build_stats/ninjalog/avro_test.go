// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ninjalog

import "testing"

func TestAVROCodec(t *testing.T) {
	if _, err := AVROCodec(); err != nil {
		t.Fatalf("failed to parse AVRO schema: %v", err)
	}
}
