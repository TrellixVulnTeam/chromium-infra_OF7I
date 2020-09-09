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

export function receiveUser(user: object) {
  return {type: RECEIVE_USER, user};
};
export function receiveRecordInfo(recordInfo: object) {
  return {type: RECEIVE_RECORD_INFO, recordInfo};
};
export function receiveRecordInfoError(error: object) {
  return {type: RECEIVE_RECORD_INFO_ERROR, error};
};
export function receiveDeviceInfoError(error: object) {
  return {type: RECEIVE_DEVICE_INFO_ERROR, error};
};

/**
 * TODO: Current implementation returns first device found from RPCs. This works
 * when passing a single hostname. May need to implement hostname to device pair
 * checking in frontend.
 */
export function receiveDeviceInfo(deviceInfo: Array<Object>) {
  return {type: RECEIVE_DEVICE_INFO, deviceInfo};
};

/**
 * Asynchronous Redux actions for anything that needs to communicate with
 * anexternal API. Redux Thunk middleware provides the capability to allow for
 * side effects and async actions as we need.
 */

/**
 * Takes the getDeviceInfo and getRecordInfo promises and returns a promise when
 * both are resolved.
 */
export function getRepairRecord(
    hostname: string, headers: {[key: string]: string}) {
  return function(dispatch) {
    return Promise.all([
      dispatch(getDeviceInfo(hostname, headers)),
      dispatch(getRecordInfo(hostname, headers))
    ]);
  };
};

/**
 * Call inventory.Inventory/GetDeviceManualRepairRecord rpc for manual repair
 * record information. Response is saved to Redux state.
 *
 * @param hostname  The hostname of the device.
 * @param headers   The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns         The response from the RPC.
 */
export function getRecordInfo(
    hostname: string, headers: {[key: string]: string}) {
  return async function(dispatch) {
    const recordMsg: {[key: string]: string} = {'hostname': hostname};
    try {
      const res = await prpcClient.call(
          'inventory.Inventory', 'GetDeviceManualRepairRecord', recordMsg,
          headers);
      return dispatch(receiveRecordInfo(res));
    } catch (err) {
      return dispatch(receiveRecordInfoError(err));
    }
  };
};

/**
 * Call inventory.Inventory/GetCrosDevices rpc for manual repair record
 * information. Response is saved to Redux state.
 *
 * @param hostname  The hostname of the device.
 * @param headers   The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns         The response from the RPC.
 */
export function getDeviceInfo(
    hostname: string, headers: {[key: string]: string}) {
  return async function(dispatch) {
    const deviceMsg: {[key: string]: Array<{[key: string]: string}>} = {
      'ids': [{'hostname': hostname}]
    };
    try {
      const res = await prpcClient.call(
          'inventory.Inventory', 'GetCrosDevices', deviceMsg, headers);
      return dispatch(receiveDeviceInfo(res.data[0]));
    } catch (err) {
      return dispatch(receiveDeviceInfoError(err));
    }
  };
};

/**
 * Call inventory.Inventory/CreateDeviceManualRepairRecord rpc for manual repair
 * record information.
 *
 * @param record  Record object of the record to be created in datastore.
 * @param headers The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns       The response from the RPC.
 */
export function createRepairRecord(
    record: Object, headers: {[key: string]: string}) {
  return async function(dispatch) {
    const recordMsg: {[key: string]: Object} = {'device_repair_record': record};
    try {
      return prpcClient.call(
          'inventory.Inventory', 'CreateDeviceManualRepairRecord', recordMsg,
          headers);
    } catch (err) {
      return dispatch(receiveRecordInfoError(err));
    }
  };
};

/**
 * Call inventory.Inventory/UpdateDeviceManualRepairRecord rpc for manual repair
 * record information.
 *
 * @param recordId  Record ID of the record to be updated in datastore.
 * @param record    Record object of the record to be updated in datastore.
 * @param headers   The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns       The response from the RPC.
 */
export function updateRepairRecord(
    recordId: string, record: Object, headers: {[key: string]: string}) {
  return async function(dispatch) {
    const recordMsg: {[key: string]: Object} = {
      'id': recordId,
      'device_repair_record': record,
    };
    try {
      return prpcClient.call(
          'inventory.Inventory', 'UpdateDeviceManualRepairRecord', recordMsg,
          headers);
    } catch (err) {
      return dispatch(receiveRecordInfoError(err));
    }
  };
};
