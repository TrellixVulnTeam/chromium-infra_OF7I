// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/config/go/api"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	luciproto "go.chromium.org/luci/common/proto"

	"infra/libs/git"
)

var (
	unmarshaller = jsonpb.Unmarshaler{AllowUnknownFields: true}
)

type gitilesInfo struct {
	project string
	path    string
}

// Programs defines the structure of a DLM program list.
type Programs struct {
	Programs []struct {
		Name           string `json:"name,omitempty"`
		Repo           *Repo  `json:"repo,omitempty"`
		DeviceProjects []struct {
			Repo *Repo `json:"repo,omitempty"`
		} `json:"deviceProjects,omitempty"`
	} `json:"programs,omitempty"`
}

// Repo defines the repo info in DLM configs.
type Repo struct {
	Name       string `json:"name,omitempty"`
	RepoPath   string `json:"repoPath,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
}

func fixFieldMaskForConfigBundleList(b []byte) ([]byte, error) {
	var payload payload.ConfigBundleList
	t := reflect.TypeOf(payload)
	buf, err := luciproto.FixFieldMasksBeforeUnmarshal(b, t)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func getDeviceConfigs(ctx context.Context, gc git.ClientInterface, joinedConfigPath string) ([]*device.Config, error) {
	logging.Infof(ctx, "reading device configs from %s", joinedConfigPath)
	content, err := gc.GetFile(ctx, joinedConfigPath)
	if err != nil {
		return nil, err
	}
	var payloads payload.ConfigBundleList
	buf, err := fixFieldMaskForConfigBundleList([]byte(content))
	if err != nil {
		return nil, errors.Annotate(err, "fail to fix field mask for %s", joinedConfigPath).Err()
	}
	if err := unmarshaller.Unmarshal(bytes.NewBuffer(buf), &payloads); err != nil {
		return nil, errors.Annotate(err, "fail to unmarshal %s", joinedConfigPath).Err()
	}

	var allCfgs []*device.Config
	for _, payload := range payloads.GetValues() {
		dcs := parseConfigBundle(payload)
		allCfgs = append(allCfgs, dcs...)
	}
	return allCfgs, nil
}

func correctProjectName(n string) string {
	return strings.Replace(n, "+", "plus", -1)
}
func correctConfigPath(p string) string {
	return strings.Replace(p, "config.jsonproto", "joined.jsonproto", -1)
}

func validRepo(r *Repo) bool {
	return r != nil && r.Name != "" && r.ConfigPath != ""
}

func parseConfigBundle(configBundle *payload.ConfigBundle) []*device.Config {
	designs := configBundle.GetDesignList()
	dcs := make(map[string]*device.Config, 0)
	for _, d := range designs {
		board := d.GetProgramId().GetValue()
		model := d.GetName()
		for _, c := range d.GetConfigs() {
			dcs[c.GetId().GetValue()] = &device.Config{
				Id: &device.ConfigId{
					PlatformId: &device.PlatformId{Value: board},
					ModelId:    &device.ModelId{Value: model},
				},
				FormFactor:       parseFormFactor(c.GetHardwareFeatures().GetFormFactor().GetFormFactor()),
				HardwareFeatures: parseHardwareFeatures(configBundle.GetComponents(), c.GetHardwareFeatures()),
				// Note: no STORAGE_SSD, STORAGE_HDD, STORAGE_UFS storage
				// label-storage is not used for scheduling tests for at least 3 months: https://screenshot.googleplex.com/B8spRMj22aUWkbb
				Storage: parseStorage(c.GetHardwareFeatures()),
				Soc:     parseSoc(configBundle.GetComponents()),
				Cpu:     parseArchitecture(configBundle.GetComponents()),
				Ec:      parseEcType(c.GetHardwareFeatures()),

				// TODO(xixuan): GpuFamily, gpu_family in Component.Soc hasn't been set
				// TODO(xixuan): Power, a new power topology hasn't been set
				// label-power is used in swarming now: https://screenshot.googleplex.com/8EAUwGeoVeBtez7
				// Graphics: removed from boxster for now
				// TODO(xixuan): VideoAccelerationSupports, a new video acceleration topology hasn't been set
				// label-video_acceleration is not used for scheduling tests for at least 3 months: https://screenshot.googleplex.com/86h2scqNsStwoiW
			}
		}
	}
	// Setup the sku
	for _, sc := range configBundle.GetSoftwareConfigs() {
		designCID := sc.GetDesignConfigId().GetValue()
		dcs[designCID].Id.VariantId = &device.VariantId{Value: fmt.Sprint(sc.GetIdScanConfig().GetFirmwareSku())}
	}
	res := make([]*device.Config, len(dcs))
	i := 0
	for _, v := range dcs {
		res[i] = v
		i++
	}
	return res
}

func parseFormFactor(ff api.HardwareFeatures_FormFactor_FormFactorType) device.Config_FormFactor {
	switch ff {
	case api.HardwareFeatures_FormFactor_CLAMSHELL:
		return device.Config_FORM_FACTOR_CLAMSHELL
	case api.HardwareFeatures_FormFactor_CONVERTIBLE:
		return device.Config_FORM_FACTOR_CONVERTIBLE
	case api.HardwareFeatures_FormFactor_DETACHABLE:
		return device.Config_FORM_FACTOR_DETACHABLE
	case api.HardwareFeatures_FormFactor_CHROMEBASE:
		return device.Config_FORM_FACTOR_CHROMEBASE
	case api.HardwareFeatures_FormFactor_CHROMEBOX:
		return device.Config_FORM_FACTOR_CHROMEBOX
	case api.HardwareFeatures_FormFactor_CHROMEBIT:
		return device.Config_FORM_FACTOR_CHROMEBIT
	case api.HardwareFeatures_FormFactor_CHROMESLATE:
		return device.Config_FORM_FACTOR_CHROMESLATE
	default:
		return device.Config_FORM_FACTOR_UNSPECIFIED
	}
}

func parseSoc(components []*api.Component) device.Config_SOC {
	for _, c := range components {
		if soc := c.GetSoc(); soc != nil {
			familyName := c.GetSoc().GetFamily().GetName()
			v, ok := device.Config_SOC_value[fmt.Sprintf("SOC_%s", strings.ToUpper(familyName))]
			if ok {
				return device.Config_SOC(v)
			}
		}
	}
	return device.Config_SOC_UNSPECIFIED
}

func parseHardwareFeatures(components []*api.Component, hf *api.HardwareFeatures) []device.Config_HardwareFeature {
	resMap := make(map[device.Config_HardwareFeature]bool)
	// Use bluetooth/camera/touchpad/touchscreen component to check
	for _, c := range components {
		if c.GetBluetooth() != nil {
			resMap[device.Config_HARDWARE_FEATURE_BLUETOOTH] = true
		}
		// How to determine it's webcam or not?
		if c.GetCamera() != nil {
			resMap[device.Config_HARDWARE_FEATURE_WEBCAM] = true
		}
		if c.GetTouchpad() != nil {
			resMap[device.Config_HARDWARE_FEATURE_TOUCHPAD] = true
		}
		if c.GetTouchscreen() != nil {
			resMap[device.Config_HARDWARE_FEATURE_TOUCHSCREEN] = true
		}
	}

	// HARDWARE_FEATURE_INTERNAL_DISPLAY: Only chromeboxes have this UNSET
	ff := hf.GetFormFactor().GetFormFactor()
	if ff != api.HardwareFeatures_FormFactor_CHROMEBOX && ff != api.HardwareFeatures_FormFactor_FORM_FACTOR_UNKNOWN {
		resMap[device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY] = true
	}

	// HARDWARE_FEATURE_STYLUS: Ensure stylus is not an empty object, e.g. "stylus": {}
	if hf.GetStylus() != nil {
		switch hf.GetStylus().GetStylus() {
		case api.HardwareFeatures_Stylus_STYLUS_UNKNOWN, api.HardwareFeatures_Stylus_NONE:
		default:
			resMap[device.Config_HARDWARE_FEATURE_STYLUS] = true
		}
	}
	// HARDWARE_FEATURE_FINGERPRINT: needs to be present
	if fp := hf.GetFingerprint(); fp != nil {
		if fp.GetLocation() != api.HardwareFeatures_Fingerprint_NOT_PRESENT {
			resMap[device.Config_HARDWARE_FEATURE_FINGERPRINT] = true
		}
	}
	// HARDWARE_FEATURE_DETACHABLE_KEYBOARD
	if hf.GetKeyboard() != nil && hf.GetKeyboard().GetKeyboardType() == api.HardwareFeatures_Keyboard_DETACHABLE {
		resMap[device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD] = true
	}

	// TODO: HARDWARE_FEATURE_FLASHROM, not used
	// TODO: HARDWARE_FEATURE_HOTWORDING, field in topology.Audio hasn't been set
	// TODO: HARDWARE_FEATURE_LUCID_SLEEP, which key in powerConfig?

	// Deduplicate & sort
	res := make([]device.Config_HardwareFeature, 0)
	for k := range resMap {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool { return int32(res[i]) < int32(res[j]) })
	return res
}

func parseStorage(hf *api.HardwareFeatures) device.Config_Storage {
	switch hf.GetStorage().GetStorageType() {
	case api.Component_Storage_NVME:
		return device.Config_STORAGE_NVME
	case api.Component_Storage_EMMC:
		return device.Config_STORAGE_MMC
	default:
		return device.Config_STORAGE_UNSPECIFIED
	}
}

func parseArchitecture(components []*api.Component) device.Config_Architecture {
	for _, c := range components {
		if soc := c.GetSoc(); soc != nil {
			switch soc.GetFamily().GetArch() {
			case api.Component_Soc_ARM:
				return device.Config_ARM
			case api.Component_Soc_ARM64:
				return device.Config_ARM64
			case api.Component_Soc_X86:
				return device.Config_X86
			case api.Component_Soc_X86_64:
				return device.Config_X86_64
			default:
				return device.Config_ARCHITECTURE_UNDEFINED
			}
		}
	}
	return device.Config_ARCHITECTURE_UNDEFINED
}

func parseEcType(hf *api.HardwareFeatures) device.Config_EC {
	switch hf.GetEmbeddedController().GetEcType() {
	case api.HardwareFeatures_EmbeddedController_EC_CHROME:
		return device.Config_EC_CHROME
	case api.HardwareFeatures_EmbeddedController_EC_WILCO:
		return device.Config_EC_WILCO
	}
	return device.Config_EC_UNSPECIFIED
}
