// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	ufspb "infra/unifiedfleet/api/v1/models"
)

func parseGetDutLabelResponse(resp *ufspb.GetDutLabelResponse) *map[string]string {
	data := map[string]string{}
	for _, l := range resp.GetDutLabel().GetLabels() {
		data[l.GetName()] = l.GetValue()
	}
	return &data
}
