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

func skuLessDeviceConfigKey(board, model string) string {
	return fmt.Sprintf("sku-less-%s-%s", board, model)
}

func parseConfigBundle(configBundle *payload.ConfigBundle) []*device.Config {
	designs := configBundle.GetDesignList()
	dcs := make(map[string]*device.Config, 0)
	for _, d := range designs {
		board := d.GetProgramId().GetValue()
		model := d.GetName()

		// Add a sku-less device config to unblock deployment
		dcs[skuLessDeviceConfigKey(board, model)] = &device.Config{
			Id: &device.ConfigId{
				PlatformId: &device.PlatformId{Value: board},
				ModelId:    &device.ModelId{Value: model},
			},
		}

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
				Storage: parseStorage(c.GetHardwareTopology(), configBundle.GetComponents()),
				Soc:     parseSoc(d.GetPlatform().GetName()),
				Cpu:     parseArchitecture(configBundle.GetComponents()),
				Ec:      parseEcType(c.GetHardwareFeatures()),

				// TODO(xixuan): GpuFamily, gpu_family in Component.Soc hasn't been set
				Power: parsePowerSupply(c.GetHardwareFeatures().GetFormFactor().GetFormFactor()),
				// Graphics: removed from boxster for now
				// TODO(xixuan): VideoAccelerationSupports, a new video acceleration topology hasn't been set
				// label-video_acceleration is not used for scheduling tests for at least 3 months: https://screenshot.googleplex.com/86h2scqNsStwoiW
				VideoAccelerationSupports: parseVideoAccelerations(d.GetPlatform().GetVideoAcceleration()),
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

func parseVideoAccelerations(vas []api.Design_Platform_VideoAcceleration) []device.Config_VideoAcceleration {
	resMap := make(map[device.Config_VideoAcceleration]bool)
	for _, va := range vas {
		switch va {
		case api.Design_Platform_H264_DECODE:
			resMap[device.Config_VIDEO_ACCELERATION_H264] = true
		case api.Design_Platform_H264_ENCODE:
			resMap[device.Config_VIDEO_ACCELERATION_ENC_H264] = true
		case api.Design_Platform_VP8_DECODE:
			resMap[device.Config_VIDEO_ACCELERATION_VP8] = true
		case api.Design_Platform_VP8_ENCODE:
			resMap[device.Config_VIDEO_ACCELERATION_ENC_VP8] = true
		case api.Design_Platform_VP9_DECODE:
			resMap[device.Config_VIDEO_ACCELERATION_VP9] = true
		case api.Design_Platform_VP9_ENCODE:
			resMap[device.Config_VIDEO_ACCELERATION_ENC_VP9] = true
		case api.Design_Platform_VP9_2_DECODE:
			resMap[device.Config_VIDEO_ACCELERATION_VP9_2] = true
		case api.Design_Platform_VP9_2_ENCODE:
			resMap[device.Config_VIDEO_ACCELERATION_ENC_VP9_2] = true
		case api.Design_Platform_H265_DECODE:
			resMap[device.Config_VIDEO_ACCELERATION_H265] = true
		case api.Design_Platform_H265_ENCODE:
			resMap[device.Config_VIDEO_ACCELERATION_ENC_H265] = true
		case api.Design_Platform_MJPG_DECODE:
			resMap[device.Config_VIDEO_ACCELERATION_MJPG] = true
		case api.Design_Platform_MJPG_ENCODE:
			resMap[device.Config_VIDEO_ACCELERATION_ENC_MJPG] = true
		}
	}

	var res []device.Config_VideoAcceleration
	for k := range resMap {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool { return int32(res[i]) < int32(res[j]) })
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

func parsePowerSupply(ff api.HardwareFeatures_FormFactor_FormFactorType) device.Config_PowerSupply {
	switch ff {
	case api.HardwareFeatures_FormFactor_CHROMEBASE, api.HardwareFeatures_FormFactor_CHROMEBOX, api.HardwareFeatures_FormFactor_CHROMEBIT:
		return device.Config_POWER_SUPPLY_AC_ONLY
	case api.HardwareFeatures_FormFactor_FORM_FACTOR_UNKNOWN:
		return device.Config_POWER_SUPPLY_UNSPECIFIED
	default:
		return device.Config_POWER_SUPPLY_BATTERY
	}
}

func parseSoc(platformName string) device.Config_SOC {
	if platformName == "" {
		return device.Config_SOC_UNSPECIFIED
	}
	// Filter out the non-parsable platform name first
	switch platformName {
	case "KABY_LAKE":
		return device.Config_SOC_KABYLAKE_U
	}

	// Check exact matching
	v, ok := device.Config_SOC_value[fmt.Sprintf("SOC_%s", strings.ToUpper(platformName))]
	if ok {
		return device.Config_SOC(v)
	}

	// Check fuzzy matching: e.g. COMET_LAKE => SOC_COMET_LAKE_U
	for k, v := range device.Config_SOC_value {
		if strings.Contains(k, platformName) {
			return device.Config_SOC(v)
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

func matchStorageType(st string) device.Config_Storage {
	switch st {
	case api.Component_Storage_NVME.String():
		return device.Config_STORAGE_NVME
	case api.Component_Storage_EMMC.String():
		return device.Config_STORAGE_MMC
	case api.Component_Storage_SATA.String():
		return device.Config_STORAGE_SSD
	}
	return device.Config_STORAGE_UNSPECIFIED
}

func parseStorage(hf *api.HardwareTopology, components []*api.Component) device.Config_Storage {
	v := matchStorageType(hf.GetNonVolatileStorage().GetHardwareFeature().GetStorage().GetStorageType().String())
	if v != device.Config_STORAGE_UNSPECIFIED {
		return v
	}

	storageComponent := make(map[string]bool)
	for _, c := range components {
		if t := c.GetStorage().GetType(); t != api.Component_Storage_STORAGE_TYPE_UNKNOWN {
			storageComponent[t.String()] = true
		}
	}
	// Verify if all storage components have the same storage type
	if len(storageComponent) == 1 {
		for k := range storageComponent {
			return matchStorageType(k)
		}
	}
	return device.Config_STORAGE_UNSPECIFIED
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
