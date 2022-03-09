// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"bytes"
	"encoding/json"
	"infra/cros/recovery/internal/planpb"
	"io"
	"log"
)

func mustCreateConfigJSON(c *planpb.Configuration) io.Reader {
	b, err := json.Marshal(c)
	if err != nil {
		log.Fatalf("Failed to create JSON config: %v", err)
	}
	return bytes.NewBuffer(b)
}
