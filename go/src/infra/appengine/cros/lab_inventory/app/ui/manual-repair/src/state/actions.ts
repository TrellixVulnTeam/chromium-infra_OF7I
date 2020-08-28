// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {prpcClient} from './prpc';

/**
 * Synchronous Redux actions to update state store.
 */
export const RECEIVE_USER = 'RECEIVE_USER';
export const RECEIVE_DEVICE_INFO = 'RECEIVE_DEVICE_INFO';
export const RECEIVE_RECORD_INFO = 'RECEIVE_RECORD_INFO';
export const RECEIVE_RECORD_INFO_ERROR = 'RECEIVE_RECORD_INFO_ERROR';
export const RECEIVE_DEVICE_INFO_ERROR = 'RECEIVE_DEVICE_INFO_ERROR';

export const receiveUser = (user: Object) => ({type: RECEIVE_USER, user});
export const receiveRecordInfo = (recordInfo: Object) =>
    ({type: RECEIVE_RECORD_INFO, recordInfo});
export const receiveRecordInfoError = (error: Object) =>
    ({type: RECEIVE_RECORD_INFO_ERROR, error});
export const receiveDeviceInfoError = (error: Object) =>
    ({type: RECEIVE_DEVICE_INFO_ERROR, error});

/**
 * TODO: Current implementation returns first device found from RPCs. This works
 * when passing a single hostname. May need to implement hostname to device pair
 * checking in frontend.
 */
export const receiveDeviceInfo = (deviceInfo: Array<Object>) =>
    ({type: RECEIVE_DEVICE_INFO, deviceInfo});

/**
 * Asynchronous Redux actions for anything that needs to communicate with
 * anexternal API. Redux Thunk middleware provides the capability to allow for
 * side effects and async actions as we need.
 */

/**
 * Takes the getDeviceInfo and getRecordInfo promises and returns a promise when
 * both are resolved.
 */
export const getRepairRecord =
    (hostname: string, headers: {[key: string]: string}) => dispatch => {
      Promise.all([
        dispatch(getDeviceInfo(hostname, headers)),
        dispatch(getRecordInfo(hostname, headers))
      ]);
    };

/**
 * Call inventory.Inventory/GetDeviceManualRepairRecord rpc for manual repair
 * record information. Response is saved to Redux state.
 *
 * @param hostname - The hostname of the device.
 * @param headers - The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns The response from the RPC.
 */
export const getRecordInfo =
    (hostname: string, headers: {[key: string]: string}) => dispatch => {
      const recordMsg: {[key: string]: string} = {'hostname': hostname};
      return prpcClient
          .call(
              'inventory.Inventory', 'GetDeviceManualRepairRecord', recordMsg,
              headers)
          .then(res => dispatch(receiveRecordInfo(res)))
          .catch(err => dispatch(receiveRecordInfoError(err)));
    };

/**
 * Call inventory.Inventory/GetCrosDevices rpc for manual repair record
 * information. Response is saved to Redux state.
 *
 * @param hostname - The hostname of the device.
 * @param headers - The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns The response from the RPC.
 */
export const getDeviceInfo =
    (hostname: string, headers: {[key: string]: string}) => dispatch => {
      const deviceMsg: {[key: string]: Array<{[key: string]: string}>} = {
        'ids': [{'hostname': hostname}]
      };
      return prpcClient
          .call('inventory.Inventory', 'GetCrosDevices', deviceMsg, headers)
          .then(res => dispatch(receiveDeviceInfo(res.data[0])))
          .catch(err => dispatch(receiveDeviceInfoError(err)));
    };
