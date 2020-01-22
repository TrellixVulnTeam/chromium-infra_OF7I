// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This portion of the querygs package contains a list of boards and models
// that are known to be absent from metadata.json files. For instance,
// labstations are not present in any metadata.json file.

package querygs

var missingBoardWhitelist map[string]bool = stringSliceToStringSet([]string{
	"buddy_cfm",
	"caroline_arcnext",
	"caroline_ndktranslation",
	"cid",
	"cyan_kernelnext",
	"eve_arcnext",
	"eve_campfire",
	"eve_kvm",
	"fizz_accelerator",
	"fizz_cfm",
	"fizz_moblab",
	"guado_accelerator",
	"guado_cfm",
	"guado_kernelnext",
	"kalista_cfm",
	"kefka_kernelnext",
	"nyan",
	"oak",
	"rambi",
	"rikku_cfm",
	"samus_cheets",
	"samus_kernelnext",
	"storm",
	"umaro",
	"veyron_gus",
	"veyron_jerry_kernelnext",
	"veyron_minnie_cheets",
	"veyron_minnie_kernelnext",
	"veyron_pinky",
	"veyron_thea",
	"x86_alex",
	"x86_alex_he",
	"x86_mario",
	"x86_zgb",
	"x86_zgb_he",
})

var failedToLookupWhiteList map[string]bool = stringSliceToStringSet([]string{
	"fizz-labstation;fizz-labstation",
	"hatch;unprovisioned_helios",
	"hatch;unprovisioned_kindred",
	"hatch;unprovisioned_kohaku",
})

func stringSliceToStringSet(input []string) map[string]bool {
	var out = make(map[string]bool, len(input))
	for _, item := range input {
		out[item] = true
	}
	return out
}
