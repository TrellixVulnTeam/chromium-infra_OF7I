// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {isEmpty} from 'lodash';
import {TYPE_DUT, TYPE_LABSTATION, TYPE_UNKNOWN} from '../../components/constants';
import * as repairConst from '../../components/repair-form/repair-form-constants';
import {RepairHistoryList, RepairHistoryRow, rspActions} from '../../components/repair-history/repair-history-constants';


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

/**
 * Takes a component name and returns the list of repair action strings
 * associated with it.
 *
 * @param component Name of component.
 * @returns         Component name and enum of all repair action display strings
 *     for the component.
 */
export function getActionStrEnum(component: string) {
  switch (component) {
    case 'labstationRepairActions':
      return {
        component: 'Labstation',
        actionList: repairConst.LabstationRepairActionString,
      };
    case 'servoRepairActions':
      return {
        component: 'Servo',
        actionList: repairConst.ServoRepairActionString,
      };
    case 'yoshiRepairActions':
      return {
        component: 'Yoshi Cable',
        actionList: repairConst.YoshiRepairActionString,
      };
    case 'chargerRepairActions':
      return {
        component: 'Charger',
        actionList: repairConst.ChargerRepairActionString,
      };
    case 'usbStickRepairActions':
      return {
        component: 'USB Stick',
        actionList: repairConst.UsbStickRepairActionString,
      };
    case 'cableRepairActions':
      return {
        component: 'Other Cables',
        actionList: repairConst.CableRepairActionString,
      };
    case 'rpmRepairActions':
      return {
        component: 'RPM',
        actionList: repairConst.RpmRepairActionString,
      };
    case 'dutRepairActions':
      return {
        component: 'DUT',
        actionList: repairConst.DutRepairActionString,
      };
    default: {
      return null;
    }
  }
}

/**
 * Takes a standard timestamp and formats it into YYYY-MM-DD HH:MM:SS.
 *
 * @param ts  Timestamp in the format YYYY-MM-DDTHH:MM:SS.
 * @returns   Formatted date.
 */
export function formatRecordTimestamp(ts: string): string {
  const noNano = ts.split('.')[0];
  const res = noNano.split('T').join(' ');
  return res;
}

/**
 * flattenRecordsActions takes the GRPC response of
 * inventory.Inventory/ListManualRepairRecords and flattens the records into a
 * RepairHistoryList of date, component, and action objects.
 */
export function flattenRecordsActions(repairHistoryRsp): RepairHistoryList {
  let repairHistoryList: RepairHistoryList = [];

  repairHistoryRsp.repairRecords.forEach(el => {
    for (const key of rspActions) {
      const actionStrEnum = getActionStrEnum(key);
      if (!actionStrEnum) {
        continue;
      }

      for (const val of el[key]) {
        let actStr = actionStrEnum.actionList[val];
        if (actStr == 'N/A') {
          continue;
        }

        let rh: RepairHistoryRow = {
          date: formatRecordTimestamp(el.updatedTime),
          component: actionStrEnum.component,
          action: actStr,
        };
        repairHistoryList.push(rh);
      }
    }
  });

  return repairHistoryList;
}
