// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {prpcClient} from './prpc';

// Synchronous Redux actions to update state store.
export const RECEIVE_USER = 'RECEIVE_USER';
export const RECEIVE_RECORDS = 'RECEIVE_RECORDS';

export const receiveUser = user => ({type: RECEIVE_USER, user});
export const receiveRecords = records => ({type: RECEIVE_RECORDS, records});

// Asynchronous Redux actions for anything that needs to communicate with an
// external API. Redux Thunk middleware provides the capability to allow for
// side effects and async actions as we need.
// TODO: Replace console log with error in UI.
export const getRecords = hostname => dispatch => {
  const msg: {[key: string]: string} = {'hostname': hostname};
  return prpcClient
      .call('inventory.Inventory', 'GetDeviceManualRepairRecord', msg)
      .then(res => dispatch(receiveRecords(res.json())))
      .catch(err => console.error(err));
};
