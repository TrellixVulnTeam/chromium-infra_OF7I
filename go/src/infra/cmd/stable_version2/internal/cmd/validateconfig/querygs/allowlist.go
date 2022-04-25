// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This portion of the querygs package contains a list of boards and models
// that are known to be absent from metadata.json files. For instance,
// labstations are not present in any metadata.json file.

package querygs

var missingBoardAllowList map[string]bool = stringSliceToStringSet([]string{
	// The boards atlas_arm64 through x86_zgb_he are legacy allowlist exceptions.
	// Remove them when possible.
	"atlas_arm64",
	"atlas_kvm",
	"buddy_cfm",
	"caroline_arcnext",
	"caroline_kernelnext",
	"caroline_ndktranslation",
	"cid",
	"cyan_kernelnext",
	"elm_kernelnext",
	"eve_arcnext",
	"eve_campfire",
	"eve_kvm",
	"fizz_accelerator",
	"fizz_cfm",
	"fizz_moblab",
	"grunt_kernelnext",
	"guado_accelerator",
	"guado_cfm",
	"guado_kernelnext",
	"hana_kernelnext",
	"hatch_diskswap",
	"kalista_cfm",
	"kefka_kernelnext",
	"nami_kvm",
	"nocturne_arm64",
	"nyan",
	"oak",
	"rambi",
	"rikku_cfm",
	"samus_cheets",
	"samus_kernelnext",
	"sarien_kvm",
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
	// For more details on reven, see
	// - b:212595053
	// - b:213332740
	"reven",
})

var failedToLookupAllowList map[string]bool = stringSliceToStringSet([]string{
	"fizz-labstation;fizz-labstation",
	"hatch;unprovisioned_helios",
	"hatch;unprovisioned_kindred",
	"hatch;unprovisioned_kohaku",
	// For more details see b:228229403
	"kefka;sabin",
	"puff;unprovisioned_puff",
	"zork;dalboz",
	"zork;trembyle",
})

// These are the DUTs that are exempted from version mismatches.
// Currently (2020 Q2), we just confirm that the firmware version and OS version are
// part of the same build. This assumption is not always met during bring-up.
// Add the models that are exempted here.
//
// Entries in this list have the form "board;model".
var invalidVersionAllowList map[string]bool = stringSliceToStringSet([]string{
	"zork;dalboz",
})

func stringSliceToStringSet(input []string) map[string]bool {
	var out = make(map[string]bool, len(input))
	for _, item := range input {
		out[item] = true
	}
	return out
}
