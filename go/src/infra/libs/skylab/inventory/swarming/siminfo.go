// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"infra/libs/skylab/inventory"
	"strconv"
)

func init() {
	converters = append(converters, siminfoConverter)
	reverters = append(reverters, siminfoReverter)

}

func siminfoConverter(dims Dimensions, ls *inventory.SchedulableLabels) {

	for _, s := range ls.GetSiminfo() {
		sim_id := ""
		if v := s.GetSlotId(); v != 0 {
			sim_id = strconv.Itoa(int(v))
			appendDim(dims, "label-sim_slot_id", sim_id)

		}
		if v := s.GetType(); v != inventory.SIMType_SIM_UNKNOWN {
			lv := "label-sim_" + sim_id + "_type"
			dims[lv] = []string{v.String()}
		}
		if eid := s.GetEid(); eid != "" {
			lv := "label-sim_" + sim_id + "_eid"
			dims[lv] = []string{eid}
		}

		if s.GetTestEsim() {
			lv := "label-sim_" + sim_id + "_test_esim"
			dims[lv] = []string{"True"}
		}

		lv := "label-sim_" + sim_id + "_num_profiles"
		dims[lv] = []string{strconv.Itoa(len(s.GetProfileInfo()))}
		for j, p := range s.GetProfileInfo() {
			profile_id := strconv.Itoa(j)
			if k := p.GetIccid(); k != "" {
				lv := "label-sim_" + sim_id + "_" + profile_id + "_iccid"
				appendDim(dims, lv, k)
			}
			if k := p.GetSimPin(); k != "" {
				lv := "label-sim_" + sim_id + "_" + profile_id + "_pin"
				appendDim(dims, lv, k)
			}
			if k := p.GetSimPuk(); k != "" {
				lv := "label-sim_" + sim_id + "_" + profile_id + "_puk"
				appendDim(dims, lv, k)

			}
			if k := p.GetCarrierName(); k != inventory.NetworkProvider_NETWORK_OTHER {
				lv := "label-sim_" + sim_id + "_" + profile_id + "_carrier_name"
				appendDim(dims, lv, k.String())
			}
		}
	}
}

func siminfoReverter(ls *inventory.SchedulableLabels, d Dimensions) Dimensions {

	num_sim := len(d["label-sim_slot_id"])
	ls.Siminfo = make([]*inventory.SIMInfo, num_sim)

	for i, v := range d["label-sim_slot_id"] {
		sim_id := v
		s := inventory.NewSiminfo()
		if j, err := strconv.ParseInt(v, 10, 32); err == nil {
			id := int32(j)
			s.SlotId = &id
		}

		lv := "label-sim_" + sim_id + "_type"
		if v, ok := getLastStringValue(d, lv); ok {
			if p, ok := inventory.SIMType_value[v]; ok {
				stype := inventory.SIMType(p)
				s.Type = &stype
			}
			delete(d, lv)
		}

		lv = "label-sim_" + sim_id + "_eid"
		d = assignLastStringValueAndDropKey(d, s.Eid, lv)

		lv = "label-sim_" + sim_id + "_test_esim"
		d = assignLastBoolValueAndDropKey(d, s.TestEsim, lv)

		lv = "label-sim_" + sim_id + "_num_profiles"
		num_profiles := 0
		d = assignLastIntValueAndDropKey(d, &num_profiles, lv)

		s.ProfileInfo = make([]*inventory.SIMProfileInfo, num_profiles)
		for j := 0; j < num_profiles; j++ {
			s.ProfileInfo[j] = inventory.NewSimprofileinfo()
			profile_id := strconv.Itoa(j)
			lv = "label-sim_" + sim_id + "_" + profile_id + "_iccid"
			d = assignLastStringValueAndDropKey(d, s.ProfileInfo[j].Iccid, lv)

			lv = "label-sim_" + sim_id + "_" + profile_id + "_pin"
			d = assignLastStringValueAndDropKey(d, s.ProfileInfo[j].SimPin, lv)

			lv = "label-sim_" + sim_id + "_" + profile_id + "_puk"
			d = assignLastStringValueAndDropKey(d, s.ProfileInfo[j].SimPuk, lv)

			lv = "label-sim_" + sim_id + "_" + profile_id + "_carrier_name"
			if v, ok := getLastStringValue(d, lv); ok {
				if c, ok := inventory.NetworkProvider_value[v]; ok {
					pt := inventory.NetworkProvider(c)
					s.ProfileInfo[j].CarrierName = &pt
				}
				delete(d, lv)
			}
		}
		ls.Siminfo[i] = s
	}
	delete(d, "label-sim_slot_id")
	return d
}
