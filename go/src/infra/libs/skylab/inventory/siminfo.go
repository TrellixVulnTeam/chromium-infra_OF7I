// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

// NewSiminfo returns a new zero value instance of Siminfo.
func NewSiminfo() *SIMInfo {
	return &SIMInfo{
		SlotId:   new(int32),
		Type:     new(SIMType),
		Eid:      new(string),
		TestEsim: new(bool),
	}
}

// NewSimprofileinfo returns a new zero value instance of Simprofileinfo.
func NewSimprofileinfo() *SIMProfileInfo {
	return &SIMProfileInfo{
		Iccid:       new(string),
		SimPin:      new(string),
		SimPuk:      new(string),
		CarrierName: new(NetworkProvider),
	}
}
