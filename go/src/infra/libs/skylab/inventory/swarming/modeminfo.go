// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"infra/libs/skylab/inventory"
	"strconv"
)

func init() {
	converters = append(converters, modeminfoConverter)
	reverters = append(reverters, modeminfoReverter)

}

func modeminfoConverter(dims Dimensions, ls *inventory.SchedulableLabels) {
	m := ls.GetModeminfo()

	if v := m.GetType(); v != inventory.ModemType_MODEM_TYPE_UNSPECIFIED {
		dims["label-modem_type"] = []string{v.String()}
	}

	if v := m.GetImei(); v != "" {
		dims["label-modem_imei"] = []string{v}
	}

	if v := m.GetSupportedBands(); v != "" {
		dims["label-modem_supported_bands"] = []string{v}
	}

	if v := m.GetSimCount(); v != 0 {
		dims["label-modem_sim_count"] = []string{strconv.Itoa(int(v))}
	}
}

func modeminfoReverter(ls *inventory.SchedulableLabels, d Dimensions) Dimensions {

	m := inventory.NewModeminfo()

	d = assignLastStringValueAndDropKey(d, m.Imei, "label-modem_imei")
	d = assignLastStringValueAndDropKey(d, m.SupportedBands, "label-modem_supported_bands")
	d = assignLastInt32ValueAndDropKey(d, m.SimCount, "label-modem_sim_count")
	if v, ok := getLastStringValue(d, "label-modem_type"); ok {
		if p, ok := inventory.ModemType_value[v]; ok {
			mtype := inventory.ModemType(p)
			m.Type = &mtype
		}
		delete(d, "label-modem_type")
	}

	if *m.Imei != "" {
		ls.Modeminfo = m
	}

	return d
}
