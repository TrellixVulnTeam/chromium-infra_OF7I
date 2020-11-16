// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {isEmpty} from 'lodash';
import {TYPE_DUT, TYPE_LABSTATION, TYPE_UNKNOWN} from '../../components/constants';
import * as repairConst from '../../components/repair-form/repair-form-constants';

/**
 * Checks the type of the device that is managed in state by this form. It
 * returns a type constant defined in '../../components/constants'.
 *
 * @param deviceInfo  Device info object received from state.
 * @returns           Enum of device type.
 */
export function checkDeviceType(deviceInfo): string {
  if (isEmpty(deviceInfo)) return TYPE_UNKNOWN;

  if (TYPE_DUT in deviceInfo.labConfig) {
    return TYPE_DUT;
  } else if (TYPE_LABSTATION in deviceInfo.labConfig) {
    return TYPE_LABSTATION;
  }
  return TYPE_UNKNOWN;
}

/**
 * Based on device type, return target type. Returns
 * repairConst.RepairTargetType.TYPE_DUT as default.
 *
 * @param deviceInfo  Device info object received from state.
 * @returns           Enum of device type.
 */
export function getRepairTargetType(deviceInfo): number {
  if (checkDeviceType(deviceInfo) === TYPE_LABSTATION) {
    return repairConst.RepairTargetType.TYPE_LABSTATION;
  }
  return repairConst.RepairTargetType.TYPE_DUT;
}

/**
 * Based on device type, return hostname from deviceInfo. Returns empty string
 * if hostname is not found.
 *
 * @param deviceInfo  Device info object received from state.
 * @returns           Hostname of device.
 */
export function getHostname(deviceInfo): string {
  if (isEmpty(deviceInfo)) return '';

  if (checkDeviceType(deviceInfo) === TYPE_DUT) {
    return deviceInfo.labConfig?.dut?.hostname;
  } else if (checkDeviceType(deviceInfo) === TYPE_LABSTATION) {
    return deviceInfo.labConfig?.labstation?.hostname;
  }
  return '';
}

/**
 * Parse deviceInfo object and return asset tag.
 *
 * @param deviceInfo  Device info object received from state.
 * @returns           Asset tag of device.
 */
export function getAssetTag(deviceInfo): string {
  return deviceInfo.labConfig?.id?.value;
}
