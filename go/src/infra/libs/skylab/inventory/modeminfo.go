// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

// NewModeminfo returns a new zero value instance of Modeminfo.
func NewModeminfo() *ModemInfo {
	return &ModemInfo{
		Type:           new(ModemType),
		Imei:           new(string),
		SupportedBands: new(string),
		SimCount:       new(int32),
	}
}
