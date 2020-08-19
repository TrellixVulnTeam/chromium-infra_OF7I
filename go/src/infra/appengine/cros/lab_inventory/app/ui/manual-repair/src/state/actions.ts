// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {prpcClient} from './prpc';

export const RECEIVE_RECORDS = 'RECEIVE_RECORDS';

export const receiveRecords = records => ({type: RECEIVE_RECORDS, records});

// async dispatch functions
// TODO: Replace console log with error in UI.
export const getRecords = hostname => dispatch => {
  const msg: {[key: string]: string} = {'hostname': hostname};
  return prpcClient
      .call('inventory.Inventory', 'GetDeviceManualRepairRecord', msg)
      .then(res => dispatch(receiveRecords(res.json())))
      .catch(err => console.error(err));
};
