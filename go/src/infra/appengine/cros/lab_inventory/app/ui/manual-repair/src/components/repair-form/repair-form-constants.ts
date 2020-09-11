// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * The triggering device that led you to work on this repair. RepairTarget type
 * enum implemented based on
 * go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum RepairTargetType {
  TYPE_DUT = 0,
  TYPE_LABSTATION = 1,
  TYPE_SERVO = 2,
}

/**
 * State for tracking manual repair progress. Repair state enum implemented
 * based on go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum RepairState {
  STATE_INVALID = 0,
  STATE_NOT_STARTED = 1,
  STATE_IN_PROGRESS = 2,
  STATE_COMPLETED = 3,
}

/**
 * Interface for a Checkbox actions configuration object.
 */
export interface CheckboxActionsConfig {
  // Name of field as created in the state.
  stateName: string;
  // Map of actions to be included in the dropdown.
  actionList: Map<string, {[key: string]: number}>;
}

/**
 * Checkbox related Enums implemented based on
 * go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum CableRepairAction {
  CABLE_NA = 0,
  CABLE_SERVO_HOST_SERVO = 1,
  CABLE_SERVO_DUT = 2,
  CABLE_SERVO_SERVO_MICRO = 3,
  CABLE_OTHER = 4,
}

export enum DutRepairAction {
  DUT_NA = 0,
  DUT_REIMAGE_DEV = 1,
  DUT_REIMAGE_PROD = 2,
  DUT_POWER_CYCLE_DUT = 4,
  DUT_REBOOT_EC = 5,
  DUT_NOT_PRESENT = 6,
  DUT_REFLASH = 7,
  DUT_REPLACE = 8,
  DUT_OTHER = 9,
}

export const CHECKBOX_ACTIONS = Object.freeze({
  cableRepairActions: {
    stateName: 'cableRepairActions',
    actionList: new Map([
      [
        'Servo Host to Servo',
        {enumVal: CableRepairAction.CABLE_SERVO_HOST_SERVO, timeVal: 5},
      ],
      [
        'Servo to DUT',
        {enumVal: CableRepairAction.CABLE_SERVO_DUT, timeVal: 5},
      ],
      [
        'Servo to servo_micro',
        {enumVal: CableRepairAction.CABLE_SERVO_SERVO_MICRO, timeVal: 5},
      ],
      [
        'Other',
        {enumVal: CableRepairAction.CABLE_OTHER, timeVal: 0},
      ],
    ]),
  },
  dutRepairActions: {
    stateName: 'dutRepairActions',
    actionList: new Map([
      [
        'Reimaged to DEV mode',
        {enumVal: DutRepairAction.DUT_REIMAGE_DEV, timeVal: 8},
      ],
      [
        'Reimaged to PROD mode',
        {enumVal: DutRepairAction.DUT_REIMAGE_PROD, timeVal: 8},
      ],
      [
        'Power Cycled on DUT side',
        {enumVal: DutRepairAction.DUT_POWER_CYCLE_DUT, timeVal: 1},
      ],
      [
        'Rebooted by reset EC (F3+PWR button)',
        {enumVal: DutRepairAction.DUT_REBOOT_EC, timeVal: 1},
      ],
      [
        'Reflashed Firmware',
        {enumVal: DutRepairAction.DUT_REFLASH, timeVal: 5},
      ],
      [
        'Replaced Hardware',
        {enumVal: DutRepairAction.DUT_REPLACE, timeVal: 15},
      ],
      [
        'Not present',
        {enumVal: DutRepairAction.DUT_NOT_PRESENT, timeVal: 0},
      ],
      [
        'Other',
        {enumVal: DutRepairAction.DUT_OTHER, timeVal: 0},
      ],
    ]),
  },
})

/**
 * Interface for a dropdown actions configuration object.
 */
export interface DropdownActionsConfig {
  // Name of the component the dropdown will be for.
  componentName: string;
  // Name of field as created in the state.
  stateName: string;
  // Array of actions to be included in the dropdown.
  actionList: Map<string, {[key: string]: number}>;
  // (Optional) Persistent helper text for dropdown.
  helperText?: string;
}

/**
 * Dropdown related Enums implemented based on
 * go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum LabstationRepairAction {
  LABSTATION_NA = 0,
  LABSTATION_POWER_CYCLE = 1,
  LABSTATION_REIMAGE = 2,
  LABSTATION_UPDATE_CONFIG = 3,
  LABSTATION_REPLACE = 4,
  LABSTATION_OTHER = 5,
  LABSTATION_FLASH = 6,
}

export enum ServoRepairAction {
  SERVO_NA = 0,
  SERVO_POWER_CYCLE = 1,
  SERVO_REPLUG_USB_TO_DUT = 2,
  SERVO_REPLUG_TO_SERVO_HOST = 3,
  SERVO_UPDATE_CONFIG = 4,
  SERVO_REPLACE = 5,
  SERVO_OTHER = 6,
}

export enum YoshiRepairAction {
  YOSHI_NA = 0,
  YOSHI_REPLUG_ON_DUT = 1,
  YOSHI_REPLUG_TO_SERVO = 2,
  YOSHI_REPLACE = 3,
  YOSHI_OTHER = 4,
}

export enum ChargerRepairAction {
  CHARGER_NA = 0,
  CHARGER_REPLUG = 1,
  CHARGER_REPLACE = 2,
  CHARGER_OTHER = 3,
}

export enum UsbStickRepairAction {
  USB_STICK_NA = 0,
  USB_STICK_REPLUG = 1,
  USB_STICK_REPLACE = 2,
  USB_STICK_MISSED = 3,
  USB_STICK_OTHER = 4,
}

export enum RpmRepairAction {
  RPM_NA = 0,
  RPM_UPDATE_DHCP = 1,
  RPM_UPDATE_DUT_CONFIG = 2,
  RPM_REPLACE = 3,
  RPM_OTHER = 4,
}

export const DROPDOWN_ACTIONS = Object.freeze({
  labstationRepairActions: {
    componentName: 'Labstation',
    stateName: 'labstationRepairActions',
    actionList: new Map([
      [
        'N/A',
        {enumVal: LabstationRepairAction.LABSTATION_NA, timeVal: 0},
      ],
      [
        'Flash Firmware',
        {enumVal: LabstationRepairAction.LABSTATION_FLASH, timeVal: 3},
      ],
      [
        'Power Cycled',
        {enumVal: LabstationRepairAction.LABSTATION_POWER_CYCLE, timeVal: 1},
      ],
      [
        'Reimaged',
        {enumVal: LabstationRepairAction.LABSTATION_REIMAGE, timeVal: 7},
      ],
      [
        'Replaced Hardware',
        {enumVal: LabstationRepairAction.LABSTATION_REPLACE, timeVal: 15},
      ],
      [
        'Updated Config',
        {enumVal: LabstationRepairAction.LABSTATION_UPDATE_CONFIG, timeVal: 5},
      ],
      [
        'Other',
        {enumVal: LabstationRepairAction.LABSTATION_OTHER, timeVal: 0},
      ],
    ]),
  },
  servoRepairActions: {
    componentName: 'Servo',
    stateName: 'servoRepairActions',
    actionList: new Map([
      [
        'N/A',
        {enumVal: ServoRepairAction.SERVO_NA, timeVal: 0},
      ],
      [
        'Power Cycled',
        {enumVal: ServoRepairAction.SERVO_POWER_CYCLE, timeVal: 1},
      ],
      [
        'Replaced Hardware',
        {enumVal: ServoRepairAction.SERVO_REPLACE, timeVal: 10},
      ],
      [
        'Replugged to Servo',
        {enumVal: ServoRepairAction.SERVO_REPLUG_TO_SERVO_HOST, timeVal: 1},
      ],
      [
        'Replugged USB to DUT',
        {enumVal: ServoRepairAction.SERVO_REPLUG_USB_TO_DUT, timeVal: 1},
      ],
      [
        'Updated Config',
        {enumVal: ServoRepairAction.SERVO_UPDATE_CONFIG, timeVal: 5},
      ],
      [
        'Other',
        {enumVal: ServoRepairAction.SERVO_OTHER, timeVal: 0},
      ],
    ]),
    helperText: 'servo_v4 or servo_v3',
  },
  yoshiRepairActions: {
    componentName: 'Yoshi Cable',
    stateName: 'yoshiRepairActions',
    actionList: new Map([
      [
        'N/A',
        {enumVal: YoshiRepairAction.YOSHI_NA, timeVal: 0},
      ],
      [
        'Replaced Yoshi Cable',
        {enumVal: YoshiRepairAction.YOSHI_REPLACE, timeVal: 5},
      ],
      [
        'Replugged on DUT Side',
        {enumVal: YoshiRepairAction.YOSHI_REPLUG_ON_DUT, timeVal: 5},
      ],
      [
        'Replugged to Servo',
        {enumVal: YoshiRepairAction.YOSHI_REPLUG_TO_SERVO, timeVal: 1},
      ],
      [
        'Other',
        {enumVal: YoshiRepairAction.YOSHI_OTHER, timeVal: 0},
      ],
    ]),
    helperText: 'ribbon or servo_micro',
  },
  chargerRepairActions: {
    componentName: 'Charger',
    stateName: 'chargerRepairActions',
    actionList: new Map([
      [
        'N/A',
        {enumVal: ChargerRepairAction.CHARGER_NA, timeVal: 0},
      ],
      [
        'Replaced',
        {enumVal: ChargerRepairAction.CHARGER_REPLACE, timeVal: 1},
      ],
      [
        'Replugged',
        {enumVal: ChargerRepairAction.CHARGER_REPLUG, timeVal: 1},
      ],
      [
        'Other',
        {enumVal: ChargerRepairAction.CHARGER_OTHER, timeVal: 0},
      ],
    ]),
  },
  usbStickRepairActions: {
    componentName: 'USB Stick',
    stateName: 'usbStickRepairActions',
    actionList: new Map([
      [
        'N/A',
        {enumVal: UsbStickRepairAction.USB_STICK_NA, timeVal: 0},
      ],
      [
        'Missing',
        {enumVal: UsbStickRepairAction.USB_STICK_MISSED, timeVal: 0},
      ],
      [
        'Replaced',
        {enumVal: UsbStickRepairAction.USB_STICK_REPLACE, timeVal: 1},
      ],
      [
        'Replugged',
        {enumVal: UsbStickRepairAction.USB_STICK_REPLUG, timeVal: 1},
      ],
      [
        'Other',
        {enumVal: UsbStickRepairAction.USB_STICK_OTHER, timeVal: 0},
      ],
    ]),
  },
  rpmRepairActions: {
    componentName: 'RPM',
    stateName: 'rpmRepairActions',
    actionList: new Map([
      [
        'N/A',
        {enumVal: RpmRepairAction.RPM_NA, timeVal: 0},
      ],
      [
        'Updated DHCP',
        {enumVal: RpmRepairAction.RPM_UPDATE_DHCP, timeVal: 5},
      ],
      [
        'Updated in DUT Config',
        {enumVal: RpmRepairAction.RPM_UPDATE_DUT_CONFIG, timeVal: 5},
      ],
      [
        'Replaced Hardware',
        {enumVal: RpmRepairAction.RPM_REPLACE, timeVal: 60},
      ],
      [
        'Other',
        {enumVal: RpmRepairAction.RPM_OTHER, timeVal: 0},
      ],
    ]),
  },
})
