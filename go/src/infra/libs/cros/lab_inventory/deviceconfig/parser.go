// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"fmt"
	"strings"

	"go.chromium.org/chromiumos/config/go/api"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/project_mgmt"
)

type gitilesInfo struct {
	project string
	path    string
}

func parsePrograms(programs *project_mgmt.Config, gitilesHost string) ([]*gitilesInfo, error) {
	var gitInfos []*gitilesInfo
	for _, pg := range programs.GetPrograms().GetValue() {
		// Add program-level config bundle path
		project, err := parseRepo(pg.GetRepo(), gitilesHost)
		if err != nil {
			return nil, err
		}
		gitInfos = append(gitInfos, &gitilesInfo{
			project: project,
			path:    pg.GetConfigPath(),
		})
		// Add project-level config bundle path
		for _, pj := range pg.GetProjects().GetValue() {
			project, err := parseRepo(pj.GetRepo(), gitilesHost)
			if err != nil {
				return nil, err
			}
			gitInfos = append(gitInfos, &gitilesInfo{
				project: project,
				path:    pj.GetConfigPath(),
			})
		}
	}
	return gitInfos, nil
}

func parseRepo(repo, gitilesHost string) (string, error) {
	if !strings.Contains(repo, gitilesHost) {
		return "", fmt.Errorf("%s is not a valid gitiles host, should contain %s", repo, gitilesHost)
	}
	// Example paths: "https://chrome-internal.googlesource.com/chromeos/project/galaxy/milkyway"
	paths := strings.SplitAfter(repo, gitilesHost)
	// Return "chromeos/project/galaxy/milkyway" as project name
	return paths[1][1:], nil
}

func parseConfigBundle(configBundle payload.ConfigBundle) []*device.Config {
	designs := configBundle.GetDesigns().GetValue()
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
				FormFactor: parseFormFactor(c.GetHardwareFeatures().GetFormFactor().GetFormFactor()),
				// TODO: GpuFamily, gpu_family in Component.Soc hasn't been set
				// Graphics: removed from boxster for now
				HardwareFeatures: parseHardwareFeatures(configBundle.GetComponents(), c.GetHardwareFeatures()),
				// TODO: Power, a new power topology hasn't been set
				Storage: parseStorage(c.GetHardwareFeatures()),
				// TODO: VideoAccelerationSupports, a new video acceleration topology hasn't been set
				Soc: parseSoc(configBundle.GetComponents()),
				Cpu: parseArchitecture(configBundle.GetComponents()),
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
	res := make([]device.Config_HardwareFeature, 0)
	if hf.GetBluetooth() != nil {
		res = append(res, device.Config_HARDWARE_FEATURE_BLUETOOTH)
	}
	// TODO: HARDWARE_FEATURE_FLASHROM, not used
	// TODO: HARDWARE_FEATURE_HOTWORDING, field in topology.Audio hasn't been set
	// HARDWARE_FEATURE_INTERNAL_DISPLAY: Only chromeboxes have this unset
	ff := hf.GetFormFactor().GetFormFactor()
	if ff != api.HardwareFeatures_FormFactor_CHROMEBOX && ff != api.HardwareFeatures_FormFactor_FORM_FACTOR_UNKNOWN {
		res = append(res, device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY)
	}
	// TODO: HARDWARE_FEATURE_LUCID_SLEEP, which key in powerConfig?
	// HARDWARE_FEATURE_WEBCAM: hw_topo.create_camera
	if hf.GetCamera() != nil {
		res = append(res, device.Config_HARDWARE_FEATURE_WEBCAM)
	}
	if hf.GetStylus() != nil {
		res = append(res, device.Config_HARDWARE_FEATURE_STYLUS)
	}
	// HARDWARE_FEATURE_TOUCHPAD: a component
	for _, c := range components {
		if c.GetTouchpad() != nil {
			res = append(res, device.Config_HARDWARE_FEATURE_TOUCHPAD)
			// May have multiple touchpads, skip the following checks if touchpad is already set.
			break
		}
	}
	// HARDWARE_FEATURE_TOUCHSCREEN: hw_topo.create_screen(touch=True)
	if screen := hf.GetScreen(); screen != nil {
		if screen.GetTouchSupport() == api.HardwareFeatures_PRESENT {
			res = append(res, device.Config_HARDWARE_FEATURE_TOUCHSCREEN)
		}
	}
	if fp := hf.GetFingerprint(); fp != nil {
		if fp.GetLocation() != api.HardwareFeatures_Fingerprint_NOT_PRESENT {
			res = append(res, device.Config_HARDWARE_FEATURE_FINGERPRINT)
		}
	}
	if hf.GetKeyboard() != nil && hf.GetKeyboard().GetKeyboardType() == api.HardwareFeatures_Keyboard_DETACHABLE {
		res = append(res, device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD)
	}
	return res
}

func parseStorage(hf *api.HardwareFeatures) device.Config_Storage {
	// TODO: How about other storage type?
	// STORAGE_SSD
	// STORAGE_HDD
	// STORAGE_UFS
	switch hf.GetStorage().GetStorageType() {
	case api.HardwareFeatures_Storage_NVME:
		return device.Config_STORAGE_NVME
	case api.HardwareFeatures_Storage_EMMC:
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
