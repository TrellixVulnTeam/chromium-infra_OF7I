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
 *
 * **RepairActionString enums are the string values that will be displayed for
 * each repair action.
 */
export enum CableRepairAction {
  CABLE_NA = 0,
  CABLE_SERVO_HOST_SERVO = 1,
  CABLE_SERVO_DUT = 2,
  CABLE_SERVO_SERVO_MICRO = 3,
  CABLE_OTHER = 4,
}

export enum CableRepairActionString {
  CABLE_NA = 'N/A',
  CABLE_SERVO_HOST_SERVO = 'Servo Host to Servo',
  CABLE_SERVO_DUT = 'Servo to DUT',
  CABLE_SERVO_SERVO_MICRO = 'Servo to servo_micro',
  CABLE_OTHER = 'Other',
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

export enum DutRepairActionString {
  DUT_NA = 'N/A',
  DUT_REIMAGE_DEV = 'Reimaged to DEV mode',
  DUT_REIMAGE_PROD = 'Reimaged to PROD mode',
  DUT_POWER_CYCLE_DUT = 'Power Cycled on DUT side',
  DUT_REBOOT_EC = 'Rebooted by reset EC (F3+PWR button)',
  DUT_NOT_PRESENT = 'Not present',
  DUT_REFLASH = 'Reflashed Firmware',
  DUT_REPLACE = 'Replaced Hardware',
  DUT_OTHER = 'Other',
}

export const CHECKBOX_ACTIONS = Object.freeze({
  cableRepairActions: {
    stateName: 'cableRepairActions',
    actionList: new Map([
      [
        CableRepairActionString.CABLE_SERVO_HOST_SERVO,
        {enumVal: CableRepairAction.CABLE_SERVO_HOST_SERVO, timeVal: 5},
      ],
      [
        CableRepairActionString.CABLE_SERVO_DUT,
        {enumVal: CableRepairAction.CABLE_SERVO_DUT, timeVal: 5},
      ],
      [
        CableRepairActionString.CABLE_SERVO_SERVO_MICRO,
        {enumVal: CableRepairAction.CABLE_SERVO_SERVO_MICRO, timeVal: 5},
      ],
      [
        CableRepairActionString.CABLE_OTHER,
        {enumVal: CableRepairAction.CABLE_OTHER, timeVal: 0},
      ],
    ]),
  },
  dutRepairActions: {
    stateName: 'dutRepairActions',
    actionList: new Map([
      [
        DutRepairActionString.DUT_REIMAGE_DEV,
        {enumVal: DutRepairAction.DUT_REIMAGE_DEV, timeVal: 8},
      ],
      [
        DutRepairActionString.DUT_REIMAGE_PROD,
        {enumVal: DutRepairAction.DUT_REIMAGE_PROD, timeVal: 8},
      ],
      [
        DutRepairActionString.DUT_POWER_CYCLE_DUT,
        {enumVal: DutRepairAction.DUT_POWER_CYCLE_DUT, timeVal: 1},
      ],
      [
        DutRepairActionString.DUT_REBOOT_EC,
        {enumVal: DutRepairAction.DUT_REBOOT_EC, timeVal: 1},
      ],
      [
        DutRepairActionString.DUT_REFLASH,
        {enumVal: DutRepairAction.DUT_REFLASH, timeVal: 5},
      ],
      [
        DutRepairActionString.DUT_REPLACE,
        {enumVal: DutRepairAction.DUT_REPLACE, timeVal: 15},
      ],
      [
        DutRepairActionString.DUT_NOT_PRESENT,
        {enumVal: DutRepairAction.DUT_NOT_PRESENT, timeVal: 0},
      ],
      [
        DutRepairActionString.DUT_OTHER,
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
 *
 * **RepairActionString enums are the string values that will be displayed for
 * each repair action.
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

export enum LabstationRepairActionString {
  LABSTATION_NA = 'N/A',
  LABSTATION_POWER_CYCLE = 'Power Cycled',
  LABSTATION_REIMAGE = 'Reimaged',
  LABSTATION_UPDATE_CONFIG = 'Updated Config',
  LABSTATION_REPLACE = 'Replaced Hardware',
  LABSTATION_OTHER = 'Other',
  LABSTATION_FLASH = 'Flash Firmware',
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

export enum ServoRepairActionString {
  SERVO_NA = 'N/A',
  SERVO_POWER_CYCLE = 'Power Cycled',
  SERVO_REPLUG_USB_TO_DUT = 'Replugged USB to DUT',
  SERVO_REPLUG_TO_SERVO_HOST = 'Replugged to Servo',
  SERVO_UPDATE_CONFIG = 'Updated Config',
  SERVO_REPLACE = 'Replaced Hardware',
  SERVO_OTHER = 'Other',
}

export enum YoshiRepairAction {
  YOSHI_NA = 0,
  YOSHI_REPLUG_ON_DUT = 1,
  YOSHI_REPLUG_TO_SERVO = 2,
  YOSHI_REPLACE = 3,
  YOSHI_OTHER = 4,
}

export enum YoshiRepairActionString {
  YOSHI_NA = 'N/A',
  YOSHI_REPLUG_ON_DUT = 'Replugged on DUT Side',
  YOSHI_REPLUG_TO_SERVO = 'Replugged to Servo',
  YOSHI_REPLACE = 'Replaced Yoshi Cable',
  YOSHI_OTHER = 'Other',
}

export enum ChargerRepairAction {
  CHARGER_NA = 0,
  CHARGER_REPLUG = 1,
  CHARGER_REPLACE = 2,
  CHARGER_OTHER = 3,
}

export enum ChargerRepairActionString {
  CHARGER_NA = 'N/A',
  CHARGER_REPLUG = 'Replugged',
  CHARGER_REPLACE = 'Replaced',
  CHARGER_OTHER = 'Other',
}

export enum UsbStickRepairAction {
  USB_STICK_NA = 0,
  USB_STICK_REPLUG = 1,
  USB_STICK_REPLACE = 2,
  USB_STICK_MISSED = 3,
  USB_STICK_OTHER = 4,
}

export enum UsbStickRepairActionString {
  USB_STICK_NA = 'N/A',
  USB_STICK_REPLUG = 'Replugged',
  USB_STICK_REPLACE = 'Replaced',
  USB_STICK_MISSED = 'Missing',
  USB_STICK_OTHER = 'Other',
}

export enum RpmRepairAction {
  RPM_NA = 0,
  RPM_UPDATE_DHCP = 1,
  RPM_UPDATE_DUT_CONFIG = 2,
  RPM_REPLACE = 3,
  RPM_OTHER = 4,
}

export enum RpmRepairActionString {
  RPM_NA = 'N/A',
  RPM_UPDATE_DHCP = 'Updated DHCP',
  RPM_UPDATE_DUT_CONFIG = 'Updated in DUT Config',
  RPM_REPLACE = 'Replaced Hardware',
  RPM_OTHER = 'Other',
}

export const DROPDOWN_ACTIONS = Object.freeze({
  labstationRepairActions: {
    componentName: 'Labstation',
    stateName: 'labstationRepairActions',
    actionList: new Map([
      [
        LabstationRepairActionString.LABSTATION_NA,
        {enumVal: LabstationRepairAction.LABSTATION_NA, timeVal: 0},
      ],
      [
        LabstationRepairActionString.LABSTATION_FLASH,
        {enumVal: LabstationRepairAction.LABSTATION_FLASH, timeVal: 3},
      ],
      [
        LabstationRepairActionString.LABSTATION_POWER_CYCLE,
        {enumVal: LabstationRepairAction.LABSTATION_POWER_CYCLE, timeVal: 1},
      ],
      [
        LabstationRepairActionString.LABSTATION_REIMAGE,
        {enumVal: LabstationRepairAction.LABSTATION_REIMAGE, timeVal: 7},
      ],
      [
        LabstationRepairActionString.LABSTATION_REPLACE,
        {enumVal: LabstationRepairAction.LABSTATION_REPLACE, timeVal: 15},
      ],
      [
        LabstationRepairActionString.LABSTATION_UPDATE_CONFIG,
        {enumVal: LabstationRepairAction.LABSTATION_UPDATE_CONFIG, timeVal: 5},
      ],
      [
        LabstationRepairActionString.LABSTATION_OTHER,
        {enumVal: LabstationRepairAction.LABSTATION_OTHER, timeVal: 0},
      ],
    ]),
  },
  servoRepairActions: {
    componentName: 'Servo',
    stateName: 'servoRepairActions',
    actionList: new Map([
      [
        ServoRepairActionString.SERVO_NA,
        {enumVal: ServoRepairAction.SERVO_NA, timeVal: 0},
      ],
      [
        ServoRepairActionString.SERVO_POWER_CYCLE,
        {enumVal: ServoRepairAction.SERVO_POWER_CYCLE, timeVal: 1},
      ],
      [
        ServoRepairActionString.SERVO_REPLACE,
        {enumVal: ServoRepairAction.SERVO_REPLACE, timeVal: 10},
      ],
      [
        ServoRepairActionString.SERVO_REPLUG_TO_SERVO_HOST,
        {enumVal: ServoRepairAction.SERVO_REPLUG_TO_SERVO_HOST, timeVal: 1},
      ],
      [
        ServoRepairActionString.SERVO_REPLUG_USB_TO_DUT,
        {enumVal: ServoRepairAction.SERVO_REPLUG_USB_TO_DUT, timeVal: 1},
      ],
      [
        ServoRepairActionString.SERVO_UPDATE_CONFIG,
        {enumVal: ServoRepairAction.SERVO_UPDATE_CONFIG, timeVal: 5},
      ],
      [
        ServoRepairActionString.SERVO_OTHER,
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
        YoshiRepairActionString.YOSHI_NA,
        {enumVal: YoshiRepairAction.YOSHI_NA, timeVal: 0},
      ],
      [
        YoshiRepairActionString.YOSHI_REPLACE,
        {enumVal: YoshiRepairAction.YOSHI_REPLACE, timeVal: 5},
      ],
      [
        YoshiRepairActionString.YOSHI_REPLUG_ON_DUT,
        {enumVal: YoshiRepairAction.YOSHI_REPLUG_ON_DUT, timeVal: 5},
      ],
      [
        YoshiRepairActionString.YOSHI_REPLUG_TO_SERVO,
        {enumVal: YoshiRepairAction.YOSHI_REPLUG_TO_SERVO, timeVal: 1},
      ],
      [
        YoshiRepairActionString.YOSHI_OTHER,
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
        ChargerRepairActionString.CHARGER_NA,
        {enumVal: ChargerRepairAction.CHARGER_NA, timeVal: 0},
      ],
      [
        ChargerRepairActionString.CHARGER_REPLACE,
        {enumVal: ChargerRepairAction.CHARGER_REPLACE, timeVal: 1},
      ],
      [
        ChargerRepairActionString.CHARGER_REPLUG,
        {enumVal: ChargerRepairAction.CHARGER_REPLUG, timeVal: 1},
      ],
      [
        ChargerRepairActionString.CHARGER_OTHER,
        {enumVal: ChargerRepairAction.CHARGER_OTHER, timeVal: 0},
      ],
    ]),
  },
  usbStickRepairActions: {
    componentName: 'USB Stick',
    stateName: 'usbStickRepairActions',
    actionList: new Map([
      [
        UsbStickRepairActionString.USB_STICK_NA,
        {enumVal: UsbStickRepairAction.USB_STICK_NA, timeVal: 0},
      ],
      [
        UsbStickRepairActionString.USB_STICK_MISSED,
        {enumVal: UsbStickRepairAction.USB_STICK_MISSED, timeVal: 0},
      ],
      [
        UsbStickRepairActionString.USB_STICK_REPLACE,
        {enumVal: UsbStickRepairAction.USB_STICK_REPLACE, timeVal: 1},
      ],
      [
        UsbStickRepairActionString.USB_STICK_REPLUG,
        {enumVal: UsbStickRepairAction.USB_STICK_REPLUG, timeVal: 1},
      ],
      [
        UsbStickRepairActionString.USB_STICK_OTHER,
        {enumVal: UsbStickRepairAction.USB_STICK_OTHER, timeVal: 0},
      ],
    ]),
  },
  rpmRepairActions: {
    componentName: 'RPM',
    stateName: 'rpmRepairActions',
    actionList: new Map([
      [
        RpmRepairActionString.RPM_NA,
        {enumVal: RpmRepairAction.RPM_NA, timeVal: 0},
      ],
      [
        RpmRepairActionString.RPM_UPDATE_DHCP,
        {enumVal: RpmRepairAction.RPM_UPDATE_DHCP, timeVal: 5},
      ],
      [
        RpmRepairActionString.RPM_UPDATE_DUT_CONFIG,
        {enumVal: RpmRepairAction.RPM_UPDATE_DUT_CONFIG, timeVal: 5},
      ],
      [
        RpmRepairActionString.RPM_REPLACE,
        {enumVal: RpmRepairAction.RPM_REPLACE, timeVal: 60},
      ],
      [
        RpmRepairActionString.RPM_OTHER,
        {enumVal: RpmRepairAction.RPM_OTHER, timeVal: 0},
      ],
    ]),
  },
})
