// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package removalreason

import (
	"time"
)

// RemovalReason is the reason that a DUT has been removed from the inventory.
// Removal requires a buganizer or monorail bug and possibly a comment and
// expiration time.
type RemovalReason struct {
	Bug     string
	Comment string
	Expire  time.Time
}
