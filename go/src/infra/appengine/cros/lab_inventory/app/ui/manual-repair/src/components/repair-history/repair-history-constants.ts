// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export type RepairHistoryRow = {
  date: string,
  component: string,
  action: string,
}

export type RepairHistoryList = Array<RepairHistoryRow>;

export const rspActions = [
  'servoRepairActions',
  'yoshiRepairActions',
  'chargerRepairActions',
  'usbStickRepairActions',
  'cableRepairActions',
  'rpmRepairActions',
  'dutRepairActions',
];
