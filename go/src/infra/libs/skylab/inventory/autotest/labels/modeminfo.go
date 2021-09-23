// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labels

import (
	"fmt"
	"strconv"
	"strings"

	"infra/libs/skylab/inventory"
)

func init() {
	converters = append(converters, modemInfoConverter)

	reverters = append(reverters, modemInfoReverter)
}

func modemInfoConverter(ls *inventory.SchedulableLabels) []string {
	var labels []string
	if v := ls.GetModeminfo(); v != nil {
		const plen = 11 //len("MODEM_TYPE_")
		lv := "modem_type:" + strings.ToLower(v.Type.String()[plen:])
		labels = append(labels, lv)
		lv = "modem_imei:" + strings.ToLower(v.GetImei())
		labels = append(labels, lv)
		lv = "modem_supported_bands:" + strings.ToLower(v.GetSupportedBands())
		labels = append(labels, lv)
		lv = "modem_sim_count:" + fmt.Sprintf("%d", int(v.GetSimCount()))
		labels = append(labels, lv)
	}
	return labels
}

func modemInfoReverter(ls *inventory.SchedulableLabels, labels []string) []string {
	m := inventory.NewModeminfo()

	for i := 0; i < len(labels); i++ {
		k, v := splitLabel(labels[i])
		switch k {
		case "modem_type":
			ln := "MODEM_TYPE_" + strings.ToUpper(v)
			vals := inventory.ModemType_value
			mtype := inventory.ModemType(vals[ln])
			m.Type = &mtype
		case "modem_imei":
			m.Imei = &v
		case "modem_supported_bands":
			m.SupportedBands = &v
		case "modem_sim_count":
			if j, err := strconv.ParseInt(v, 10, 32); err == nil {
				count := int32(j)
				m.SimCount = &count
			}
		default:
			continue
		}
		labels = removeLabel(labels, i)
		i--
	}

	if *m.Imei != "" {
		ls.Modeminfo = m
	}
	return labels
}
