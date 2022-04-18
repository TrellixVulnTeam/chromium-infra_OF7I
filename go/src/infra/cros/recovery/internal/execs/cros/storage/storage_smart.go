// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package storage

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// storageState is a description of the DUT's storage state given the type of the DUT storage.
type StorageState string

const (
	// DUT storage state cannot be determined.
	StorageStateUndefined StorageState = "UNDEFINED"
	// DUT storage state is normal.
	StorageStateNormal StorageState = "NORMAL"
	// DUT storage state is warning.
	StorageStateWarning StorageState = "WARNING"
	// DUT storage state is critical.
	StorageStateCritical StorageState = "CRITICAL"
)

// storageSMART is used to store the processed information of both storage type and storage state
// after it reads from the storage-info-common.sh file on the DUT.
//
// supported storageType : MMC, NVME, SSD
// supported storageState: storageStateUndefined, storageStateNomral, storageStateWarning, storageStateCritical
type storageSMART struct {
	StorageType  tlw.StorageType
	StorageState StorageState
}

// ParseSMARTInfo reads the storage info from SMART.
// The info will be located as collection of lines
func ParseSMARTInfo(ctx context.Context, rawOutput string) (*storageSMART, error) {
	storageType, storageState, err := storageSMARTFieldValue(ctx, rawOutput)
	return &storageSMART{
		StorageType:  storageType,
		StorageState: storageState,
	}, errors.Annotate(err, "parse smart info").Err()
}

type storageStateFunc func(context.Context, []string) (StorageState, error)

var typeToStateFuncMap map[tlw.StorageType]storageStateFunc = map[tlw.StorageType]storageStateFunc{
	tlw.StorageTypeSSD:  detectSSDState,
	tlw.StorageTypeMMC:  detectMMCState,
	tlw.StorageTypeNVME: detectNVMEState,
}

// storageSMARTFieldValue takes the raw output from the command line and return the field value of the storageSMART struct.
func storageSMARTFieldValue(ctx context.Context, rawOutput string) (tlw.StorageType, StorageState, error) {
	rawOutput = strings.TrimSpace(rawOutput)
	if rawOutput == "" {
		return tlw.StorageTypeUnspecified, StorageStateUndefined, errors.Reason("storageSMART field value: storage info is empty").Err()
	}
	storageInfoSlice := strings.Split(rawOutput, "\n")
	storageType, err := extractStorageType(ctx, storageInfoSlice)
	if err != nil {
		return tlw.StorageTypeUnspecified, StorageStateUndefined, errors.Annotate(err, "storageSMART field value").Err()
	}
	funcToCall, typeInMap := typeToStateFuncMap[storageType]
	if !typeInMap {
		return storageType, StorageStateUndefined, nil
	}
	storageState, err := funcToCall(ctx, storageInfoSlice)
	if err != nil {
		return storageType, StorageStateUndefined, errors.Annotate(err, "storageSMART field value").Err()
	}
	return storageType, storageState, nil
}

const (
	// Example "SATA Version is: SATA 3.1, 6.0 Gb/s (current: 6.0 Gb/s)"
	ssdTypeStorageGlob = `SATA Version is:.*`
	// Example "   Extended CSD rev 1.7 (MMC 5.0)"
	mmcTypeStorageGlob = `\s*Extended CSD rev.*MMC (?P<version>\d+.\d+)`
	// Example "SMART/Health Information (NVMe Log 0x02, NSID 0xffffffff)"
	nvmeTypeStorageGlob = `.*NVMe Log .*`
)

// extractStorageType extracts the storage type information from storageInfoSlice.
// return error if the regular expression cannot compile.
func extractStorageType(ctx context.Context, storageInfoSlice []string) (tlw.StorageType, error) {
	log.Debugf(ctx, "Extracting storage type")
	ssdTypeRegexp, err := regexp.Compile(ssdTypeStorageGlob)
	if err != nil {
		return tlw.StorageTypeUnspecified, errors.Annotate(err, "extract storage type").Err()
	}
	mmcTypeRegexp, err := regexp.Compile(mmcTypeStorageGlob)
	if err != nil {
		return tlw.StorageTypeUnspecified, errors.Annotate(err, "extract storage type").Err()
	}
	nvmeTypeRegexp, err := regexp.Compile(nvmeTypeStorageGlob)
	if err != nil {
		return tlw.StorageTypeUnspecified, errors.Annotate(err, "extract storage type").Err()
	}
	for _, line := range storageInfoSlice {
		// check if storage type is SSD
		if ssdTypeRegexp.MatchString(line) {
			return tlw.StorageTypeSSD, nil
		}
		// check if storage type is MMC
		mMMC, err := regexpSubmatchToMap(mmcTypeRegexp, line)
		if err == nil {
			log.Infof(ctx, "Found line => "+line)
			if version, ok := mMMC["version"]; ok {
				log.Infof(ctx, "Found eMMC device, version: %s", version)
			}
			return tlw.StorageTypeMMC, nil
		}
		// check if storage type is nvme
		if nvmeTypeRegexp.MatchString(line) {
			return tlw.StorageTypeNVME, nil
		}
	}
	return tlw.StorageTypeUnspecified, nil
}

const (
	// Field meaning and example line that have failing attribute
	// https://en.wikipedia.org/wiki/S.M.A.R.T.
	// ID# ATTRIBUTE_NAME     FLAGS    VALUE WORST THRESH FAIL RAW_VALUE
	// 184 End-to-End_Error   PO--CK   001   001   097    NOW  135
	ssdFailGlob            = `\s*(?P<param>\S+\s\S+)\s+[P-][O-][S-][R-][C-][K-](\s+\d{3}){3}\s+NOW`
	ssdRelocateSectorsGlob = `\s*\d\sReallocated_Sector_Ct\s*[P-][O-][S-][R-][C-][K-]\s*(?P<value>\d{3})\s*(?P<worst>\d{3})\s*(?P<thresh>\d{3})`
)

// detectSSDState read the info to detect state for SSD storage.
// return error if the regular expression cannot compile.
func detectSSDState(ctx context.Context, storageInfoSlice []string) (StorageState, error) {
	log.Infof(ctx, "Extraction metrics for SSD storage")
	ssdFailRegexp, err := regexp.Compile(ssdFailGlob)
	if err != nil {
		return StorageStateUndefined, errors.Annotate(err, "detect ssd state").Err()
	}
	ssdRelocateSectorsRegexp, err := regexp.Compile(ssdRelocateSectorsGlob)
	if err != nil {
		return StorageStateUndefined, errors.Annotate(err, "detect ssd state").Err()
	}
	for _, line := range storageInfoSlice {
		_, err := regexpSubmatchToMap(ssdFailRegexp, line)
		if err == nil {
			log.Debugf(ctx, "Found critical line => %q", line)
			return StorageStateCritical, nil
		}
		mRelocate, err := regexpSubmatchToMap(ssdRelocateSectorsRegexp, line)
		if err == nil {
			log.Debugf(ctx, "Found warning line => %q", line)
			value, _ := strconv.ParseFloat(mRelocate["value"], 64)
			// manufacture set default value 100, if number started to grow then it is time to mark it.
			if value > 100 {
				return StorageStateWarning, nil
			}
		}
	}
	return StorageStateNormal, nil
}

const (
	// Ex:
	// Device life time type A [DEVICE_LIFE_TIME_EST_TYP_A: 0x01]
	// 0x00~9 means 0-90% band
	// 0x0a means 90-100% band
	// 0x0b means over 100% band
	mmcFailLifeGlob = `.*(?P<param>DEVICE_LIFE_TIME_EST_TYP_.)]?: 0x0(?P<val>\S)` // life time persentage
	// Ex "Pre EOL information [PRE_EOL_INFO: 0x01]"
	// 0x00 - not defined
	// 0x01 - Normal
	// 0x02 - Warning, consumed 80% of the reserved blocks
	// 0x03 - Urgent, consumed 90% of the reserved blocks
	mmcFailEolGlob = `.*(?P<param>PRE_EOL_INFO)]?: 0x0(?P<val>\d)`
)

// detectMMCState read the info to detect state for MMC storage.
// return error if the regular expression cannot compile.
func detectMMCState(ctx context.Context, storageInfoSlice []string) (StorageState, error) {
	log.Infof(ctx, "Extraction metrics for MMC storage")
	mmcFailLevRegexp, err := regexp.Compile(mmcFailLifeGlob)
	if err != nil {
		return StorageStateUndefined, errors.Annotate(err, "detect mmc state").Err()
	}
	mmcFailEolRegexp, err := regexp.Compile(mmcFailEolGlob)
	if err != nil {
		return StorageStateUndefined, errors.Annotate(err, "detect mmc state").Err()
	}
	eolValue := 0
	lifeValue := -1
	for _, line := range storageInfoSlice {
		mLife, err := regexpSubmatchToMap(mmcFailLevRegexp, line)
		if err == nil {
			param := mLife["val"]
			log.Debugf(ctx, "Found line for lifetime estimate => %q", line)
			var val int
			if param == "a" {
				val = 100
			} else if param == "b" {
				val = 101
			} else {
				parsedVal, parseIntErr := strconv.ParseInt(param, 10, 64)
				if parseIntErr != nil {
					log.Errorf(ctx, parseIntErr.Error())
				}
				val = int(parsedVal * 10)
			}
			if val > lifeValue {
				lifeValue = val
			}
			continue
		}
		mEol, err := regexpSubmatchToMap(mmcFailEolRegexp, line)
		if err == nil {
			param := mEol["val"]
			log.Debugf(ctx, "Found line for end-of-life => %q", line)
			parsedVal, parseIntErr := strconv.ParseInt(param, 10, 64)
			if parseIntErr != nil {
				log.Errorf(ctx, parseIntErr.Error())
			}
			eolValue = int(parsedVal)
			break
		}
	}
	// set state based on end-of-life
	if eolValue == 3 {
		return StorageStateCritical, nil
	} else if eolValue == 2 {
		return StorageStateWarning, nil
	} else if eolValue == 1 {
		return StorageStateNormal, nil
	}
	// set state based on life of estimates
	if lifeValue < 90 {
		return StorageStateNormal, nil
	} else if lifeValue < 100 {
		return StorageStateWarning, nil
	}
	return StorageStateCritical, nil
}

const (
	// Ex "Percentage Used:         100%"
	nvmeFailGlob = `Percentage Used:\s+(?P<param>(\d{1,3}))%`
)

// detectNVMEState read the info to detect state for NVMe storage.
// return error if the regular expression cannot compile
func detectNVMEState(ctx context.Context, storageInfoSlice []string) (StorageState, error) {
	log.Infof(ctx, "Extraction metrics for NVMe storage")
	nvmeFailRegexp, err := regexp.Compile(nvmeFailGlob)
	if err != nil {
		return StorageStateUndefined, errors.Annotate(err, "detect nvme state").Err()
	}
	var usedValue int = -1
	for _, line := range storageInfoSlice {
		m, err := regexpSubmatchToMap(nvmeFailRegexp, line)
		if err == nil {
			log.Debugf(ctx, "Found line for usage => %q", line)
			val, convertErr := strconv.ParseInt(m["param"], 10, 64)
			if convertErr == nil {
				usedValue = int(val)
			} else {
				log.Debugf(ctx, "Could not cast: %s to int", m["param"])
			}
			break
		}
	}
	if usedValue < 91 {
		log.Infof(ctx, "NVME storage usage value: %v", usedValue)
		return StorageStateNormal, nil
	}
	return StorageStateWarning, nil
}

// regexpSubmatchToMap takes pattern of regex and the source string and returns
// the map containing the groups defined in the regex expression.
// Assumes the pattern can compile.
// return error if it doesn't find any match
func regexpSubmatchToMap(r *regexp.Regexp, source string) (map[string]string, error) {
	m := make(map[string]string)
	matches := r.FindStringSubmatch(source)
	if len(matches) < 1 {
		return m, errors.Reason("regexp submatch to map: no match found").Err()
	}
	// there is at least 1 match found
	names := r.SubexpNames()
	for i := range names {
		if i != 0 {
			m[names[i]] = matches[i]
		}
	}
	return m, nil
}
