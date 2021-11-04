// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package uuid

import (
	"github.com/google/uuid"

	"infra/cros/karte/internal/errors"
)

// UUID creates a new UUID as a string or returns an error if we failed to
// produce a UUID.
func UUID() (string, error) {
	out, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Annotate(err, "uuid").Err()
	}
	return out.String(), nil
}
