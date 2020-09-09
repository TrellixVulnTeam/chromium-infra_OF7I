// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * The triggering device that led you to work on this repair. RepairTarget type
 * enum implemented based on
 * go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum RepairTargetType {
  TYPE_DUT,
  TYPE_LABSTATION,
  TYPE_SERVO,
}

/**
 * State for tracking manual repair progress. Repair state enum implemented
 * based on go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum RepairState {
  STATE_INVALID,
  STATE_NOT_STARTED,
  STATE_IN_PROGRESS,
  STATE_COMPLETED,
}

/**
 * Interface for a Checkbox actions configuration object.
 */
export interface CheckboxActionsConfig {
  // ID of the checkbox collection div.
  idName: string;
  // Name of field as created in the state.
  stateName: string;
  // Map of actions to be included in the dropdown.
  actionList: Map<string, number>;
}

/**
 * Checkbox related Enums implemented based on
 * go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum CableRepairAction {
  CABLE_NA,
  CABLE_SERVO_HOST_SERVO,
  CABLE_SERVO_DUT,
  CABLE_SERVO_SERVO_MICRO,
  CABLE_OTHER,
}

export enum DutRepairAction {
  DUT_NA,
  DUT_REIMAGE_DEV,
  DUT_REIMAGE_PROD,
  DUT_POWER_CYCLE_RPM,
  DUT_POWER_CYCLE_DUT,
  DUT_REBOOT_EC,
  DUT_NOT_PRESENT,
  DUT_REFLASH,
  DUT_REPLACE,
  DUT_OTHER,
}

export const CHECKBOX_ACTIONS = Object.freeze({
  cableRepairActions: {
    idName: 'cable',
    stateName: 'cableRepairActions',
    actionList: new Map([
      ['Servo Host to Servo', CableRepairAction.CABLE_SERVO_HOST_SERVO],
      ['Servo to DUT', CableRepairAction.CABLE_SERVO_DUT],
      ['Servo to servo_micro', CableRepairAction.CABLE_SERVO_SERVO_MICRO],
      ['Other', CableRepairAction.CABLE_OTHER],
    ]),
  },
  dutRepairActions: {
    idName: 'dut',
    stateName: 'dutRepairActions',
    actionList: new Map([
      ['Reimaged (DEV mode)', DutRepairAction.DUT_REIMAGE_DEV],
      ['Reimaged (PROD mode)', DutRepairAction.DUT_REIMAGE_PROD],
      ['Power Cycled by RPM', DutRepairAction.DUT_POWER_CYCLE_RPM],
      ['Power Cycled on DUT side', DutRepairAction.DUT_POWER_CYCLE_DUT],
      ['Rebooted by reset EC (F3+PWR button)', DutRepairAction.DUT_REBOOT_EC],
      ['Reflashed', DutRepairAction.DUT_REFLASH],
      ['Replaced', DutRepairAction.DUT_REPLACE],
      ['Not present', DutRepairAction.DUT_NOT_PRESENT],
      ['Other', DutRepairAction.DUT_OTHER],
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
  actionList: Map<string, number>;
  // (Optional) Persistent helper text for dropdown.
  helperText?: string;
}

/**
 * Dropdown related Enums implemented based on
 * go/src/infra/libs/cros/lab_inventory/protos/repair_record.proto
 */
export enum LabstationRepairAction {
  LABSTATION_NA,
  LABSTATION_POWER_CYCLE,
  LABSTATION_REIMAGE,
  LABSTATION_UPDATE_CONFIG,
  LABSTATION_REPLACE,
  LABSTATION_OTHER,
}

export enum ServoRepairAction {
  SERVO_NA,
  SERVO_POWER_CYCLE,
  SERVO_REPLUG_USB_TO_DUT,
  SERVO_REPLUG_TO_SERVO_HOST,
  SERVO_UPDATE_CONFIG,
  SERVO_REPLACE,
  SERVO_OTHER,
}

export enum YoshiRepairAction {
  YOSHI_NA,
  YOSHI_REPLUG_ON_DUT,
  YOSHI_REPLUG_TO_SERVO,
  YOSHI_REPLACE,
  YOSHI_OTHER,
}

export enum ChargerRepairAction {
  CHARGER_NA,
  CHARGER_REPLUG,
  CHARGER_REPLACE,
  CHARGER_OTHER,
}

export enum UsbStickRepairAction {
  USB_STICK_NA,
  USB_STICK_REPLUG,
  USB_STICK_REPLACE,
  USB_STICK_MISSED,
  USB_STICK_OTHER,
}

export enum RpmRepairAction {
  RPM_NA,
  RPM_UPDATE_DHCP,
  RPM_UPDATE_DUT_CONFIG,
  RPM_REPLACE,
  RPM_OTHER,
}

export const DROPDOWN_ACTIONS = Object.freeze({
  labstationRepairActions: {
    componentName: 'Labstation',
    stateName: 'labstationRepairActions',
    actionList: new Map([
      ['N/A', LabstationRepairAction.LABSTATION_NA],
      ['Power Cycled', LabstationRepairAction.LABSTATION_POWER_CYCLE],
      ['Reimaged', LabstationRepairAction.LABSTATION_REIMAGE],
      ['Replaced', LabstationRepairAction.LABSTATION_REPLACE],
      ['Updated Config', LabstationRepairAction.LABSTATION_UPDATE_CONFIG],
      ['Other', LabstationRepairAction.LABSTATION_OTHER],
    ]),
  },
  servoRepairActions: {
    componentName: 'Servo',
    stateName: 'servoRepairActions',
    actionList: new Map([
      ['N/A', ServoRepairAction.SERVO_NA],
      ['Power Cycled', ServoRepairAction.SERVO_POWER_CYCLE],
      ['Replaced', ServoRepairAction.SERVO_REPLACE],
      ['Replugged to Servo', ServoRepairAction.SERVO_REPLUG_TO_SERVO_HOST],
      ['Replugged USB to DUT', ServoRepairAction.SERVO_REPLUG_USB_TO_DUT],
      ['Other', ServoRepairAction.SERVO_OTHER],
    ]),
    helperText: 'servo_v4 or servo_v3',
  },
  yoshiRepairActions: {
    componentName: 'Yoshi Cable',
    stateName: 'yoshiRepairActions',
    actionList: new Map([
      ['N/A', YoshiRepairAction.YOSHI_NA],
      ['Replaced', YoshiRepairAction.YOSHI_REPLACE],
      ['Replugged on DUT Side', YoshiRepairAction.YOSHI_REPLUG_ON_DUT],
      ['Replugged to Servo', YoshiRepairAction.YOSHI_REPLUG_TO_SERVO],
      ['Other', YoshiRepairAction.YOSHI_OTHER],
    ]),
    helperText: 'ribbon or servo_micro',
  },
  chargerRepairActions: {
    componentName: 'Charger',
    stateName: 'chargerRepairActions',
    actionList: new Map([
      ['N/A', ChargerRepairAction.CHARGER_NA],
      ['Replaced', ChargerRepairAction.CHARGER_REPLACE],
      ['Replugged', ChargerRepairAction.CHARGER_REPLUG],
      ['Other', ChargerRepairAction.CHARGER_OTHER],
    ]),
  },
  usbStickRepairActions: {
    componentName: 'USB Stick',
    stateName: 'usbStickRepairActions',
    actionList: new Map([
      ['N/A', UsbStickRepairAction.USB_STICK_NA],
      ['Missed', UsbStickRepairAction.USB_STICK_MISSED],
      ['Replaced', UsbStickRepairAction.USB_STICK_REPLACE],
      ['Replugged', UsbStickRepairAction.USB_STICK_REPLUG],
      ['Other', UsbStickRepairAction.USB_STICK_OTHER],
    ]),
  },
  rpmRepairActions: {
    componentName: 'RPM',
    stateName: 'rpmRepairActions',
    actionList: new Map([
      ['N/A', RpmRepairAction.RPM_NA],
      ['Updated DHCP', RpmRepairAction.RPM_UPDATE_DHCP],
      ['Updated in DUT Config', RpmRepairAction.RPM_UPDATE_DUT_CONFIG],
      ['Replaced', RpmRepairAction.RPM_REPLACE],
      ['Other', RpmRepairAction.RPM_OTHER],
    ]),
  },
})
