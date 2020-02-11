// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"github.com/golang/protobuf/proto"

	"go.chromium.org/chromiumos/infra/proto/go/device"
)

// BoardToPlatformMap refers to the mapping from autotest board to device reference board:
// https://docs.google.com/spreadsheets/d/1R6ycgnIJnoSWVpN9pVvnrPUacady-vYFkjMrW-pHZ9c/edit#gid=0
// Current DUTs' boards and models are not 1:1 matched to platforms and models in device config.
// BoardToPlatformMap and ModelMap are used temporarily to do the mapping, so that no duts will
// skip device config retrieval.
var BoardToPlatformMap = map[string]string{
	"auron_paine":             "auron",
	"auron_yuna":              "auron",
	"buddy":                   "auron",
	"gandof":                  "auron",
	"lulu":                    "auron",
	"samus":                   "auron",
	"monroe":                  "beltino",
	"panther":                 "beltino",
	"tricky":                  "beltino",
	"zako":                    "beltino",
	"caroline":                "glados",
	"caroline-ndktranslation": "glados",
	"cave":                    "glados",
	"chell":                   "glados",
	"bob":                     "gru",
	"kevin":                   "gru",
	"kahlee":                  "grunt",
	"fizz-labstation":         "fizz",
	"fizz-moblab":             "fizz",
	"guado-kernelnext":        "jecht",
	"guado_labstation":        "jecht",
	"guado":                   "jecht",
	"rikku":                   "jecht",
	"tidus":                   "jecht",
	"asuka":                   "kunimitsu",
	"lars":                    "kunimitsu",
	"sentry":                  "kunimitsu",
	"nyan_big":                "nyan",
	"nyan_blaze":              "nyan",
	"nyan_kitty":              "nyan",
	"elm":                     "oak",
	"hana":                    "oak",
	"atlas":                   "poppy",
	"eve":                     "poppy",
	"nautilus":                "poppy",
	"nocturne":                "poppy",
	"soraka":                  "poppy",
	"banjo":                   "rambi",
	"candy":                   "rambi",
	"enguarde":                "rambi",
	"gnawty":                  "rambi",
	"heli":                    "rambi",
	"kip":                     "rambi",
	"ninja":                   "rambi",
	"orco":                    "rambi",
	"quawks":                  "rambi",
	"squawks":                 "rambi",
	"sumo":                    "rambi",
	"swanky":                  "rambi",
	"winky":                   "rambi",
	"pyro":                    "reef",
	"sand":                    "reef",
	"snappy":                  "reef",
	"arcada":                  "sarien",
	"sarien-kvm":              "sarien",
	"falco":                   "slippy",
	"falco_li":                "slippy",
	"peppy":                   "slippy",
	"banon":                   "strago",
	"celes":                   "strago",
	"cyan":                    "strago",
	"edgar":                   "strago",
	"kefka":                   "strago",
	"reks":                    "strago",
	"relm":                    "strago",
	"sabin":                   "strago",
	"setzer":                  "strago",
	"terra":                   "strago",
	"ultima":                  "strago",
	"wizpig":                  "strago",
	"veyron_fievel":           "veyron_pinky",
	"veyron_jaq":              "veyron_pinky",
	"veyron_jerry":            "veyron_pinky",
	"veyron_mickey":           "veyron_pinky",
	"veyron_mighty":           "veyron_pinky",
	"veyron_minnie":           "veyron_pinky",
	"veyron_rialto":           "veyron_pinky",
	"veyron_speedy":           "veyron_pinky",
	"veyron_tiger":            "veyron_pinky",
}

// ModelMap refers to the mapping from autotest model to device model:
// https://docs.google.com/spreadsheets/d/1R6ycgnIJnoSWVpN9pVvnrPUacady-vYFkjMrW-pHZ9c/edit#gid=0
var ModelMap = map[string]string{
	"arcada_signed":    "arcada",
	"falco_li":         "falco",
	"guado-kernelnext": "guado",
	"sarien_signed":    "sarien",
}

// GetValidPlatform maps some autotest boards to valid device config platform values.
func GetValidPlatform(platform string) string {
	if mapped, ok := BoardToPlatformMap[platform]; ok {
		return mapped
	}
	return platform
}

// GetValidModel maps some autotest models to valid device config model values.
func GetValidModel(model string) string {
	if mapped, ok := ModelMap[model]; ok {
		return mapped
	}
	return model
}

// ConvertValidDeviceConfigID maps existing device config ID from autotest to valid one.
func ConvertValidDeviceConfigID(dcID *device.ConfigId) *device.ConfigId {
	newID := proto.Clone(dcID).(*device.ConfigId)
	oldPlatform := dcID.GetPlatformId().GetValue()
	if oldPlatform != "" {
		newID.PlatformId.Value = GetValidPlatform(oldPlatform)
	}
	oldModel := dcID.GetModelId().GetValue()
	if oldModel != "" {
		newID.ModelId.Value = GetValidModel(oldModel)
	}
	return newID
}
