// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

// NewLicense returns a new zero value instance of License.
func NewLicense() *License {
	return &License{
		Type:       new(LicenseType),
		Identifier: new(string),
	}
}
