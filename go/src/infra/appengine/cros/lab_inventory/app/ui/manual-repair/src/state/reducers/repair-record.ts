// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {Action} from 'redux';
import {ThunkAction} from 'redux-thunk';

import {defaultUfsHeaders, inventoryApiVersion, inventoryClient, ufsApiVersion, ufsClient} from '../prpc';
import {AppThunk, AppThunkDispatch} from '../store';

import {ApplicationState} from './index';
import {receiveAppMessage} from './message';

/**
 * Synchronous Redux actions to update state store.
 */
export const RECEIVE_DEVICE_INFO = 'RECEIVE_DEVICE_INFO';
export const RECEIVE_RECORD_INFO = 'RECEIVE_RECORD_INFO';
export const RECEIVE_REPAIR_HISTORY = 'RECEIVE_REPAIR_HISTORY';
export const RECEIVE_DEVICE_INFO_ERROR = 'RECEIVE_DEVICE_INFO_ERROR';
export const RECEIVE_RECORD_INFO_ERROR = 'RECEIVE_RECORD_INFO_ERROR';
export const CLEAR_DEVICE_INFO = 'CLEAR_DEVICE_INFO';
export const CLEAR_RECORD_INFO = 'CLEAR_RECORD_INFO';
export const CLEAR_REPAIR_HISTORY = 'CLEAR_REPAIR_HISTORY';

export function receiveRecordInfo(recordInfo: object) {
  return {type: RECEIVE_RECORD_INFO, recordInfo};
};
export function receiveRepairHistory(repairHistory: object) {
  return {type: RECEIVE_REPAIR_HISTORY, repairHistory};
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

export function clearDeviceInfo() {
  return {type: CLEAR_DEVICE_INFO};
};
export function clearRecordInfo() {
  return {type: CLEAR_RECORD_INFO};
};
export function clearRepairHistory() {
  return {type: CLEAR_REPAIR_HISTORY};
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
  return function(dispatch: AppThunkDispatch) {
    return Promise.all([
      dispatch(getDeviceInfo(hostname, headers)),
      dispatch(getRecordInfo(hostname, headers)),
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
    hostname: string, headers: {[key: string]: string}):
    ThunkAction<void, ApplicationState, unknown, Action<string>> {
  return function(dispatch: AppThunkDispatch) {
    const recordMsg: {[key: string]: string} = {'hostname': hostname};
    return inventoryClient
        .call(
            inventoryApiVersion, 'GetDeviceManualRepairRecord', recordMsg,
            headers)
        .then(
            res => dispatch(receiveRecordInfo(res)),
            err => dispatch(receiveRecordInfoError(err)),
        );
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
    hostname: string, headers: {[key: string]: string}):
    ThunkAction<void, ApplicationState, unknown, Action<string>> {
  return function(dispatch: AppThunkDispatch) {
    const deviceMsg: {[key: string]: string} = {
      'hostname': hostname,
    };
    return ufsClient
        .call(
            ufsApiVersion, 'GetChromeOSDeviceData', deviceMsg,
            Object.assign({}, defaultUfsHeaders, headers))
        .then(
            res => {dispatch(receiveDeviceInfo(res))},
            err => Promise.all([
              dispatch(receiveDeviceInfoError(err)),
              dispatch(receiveAppMessage(err.description)),
            ]),
        );
  };
};

/**
 * Call inventory.Inventory/CreateDeviceManualRepairRecord rpc for manual repair
 * record information. Error is propagated to the next level for the component
 * to handle.
 *
 * @param record  Record object of the record to be created in datastore.
 * @param headers The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns       The response from the RPC.
 */
export function createRepairRecord(
    record: Object,
    headers: {[key: string]: string}): AppThunk<Promise<object>> {
  return function(dispatch: AppThunkDispatch) {
    const recordMsg: {[key: string]: Object} = {'device_repair_record': record};
    return inventoryClient
        .call(
            inventoryApiVersion, 'CreateDeviceManualRepairRecord', recordMsg,
            headers)
        .then(
            res => res,
            err => {
              Promise.all([
                dispatch(receiveRecordInfoError(err)),
                dispatch(receiveAppMessage(err.description)),
              ]);
              throw Error(err.description);
            },
        );
  };
};

/**
 * Call inventory.Inventory/UpdateDeviceManualRepairRecord rpc for manual repair
 * record information. Error is propagated to the next level for the component
 * to handle.
 *
 * @param recordId  Record ID of the record to be updated in datastore.
 * @param record    Record object of the record to be updated in datastore.
 * @param headers   The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns         The response from the RPC.
 */
export function updateRepairRecord(
    recordId: string, record: Object,
    headers: {[key: string]: string}): AppThunk<Promise<object>> {
  return function(dispatch: AppThunkDispatch) {
    const recordMsg: {[key: string]: Object} = {
      'id': recordId,
      'device_repair_record': record,
    };
    return inventoryClient
        .call(
            inventoryApiVersion, 'UpdateDeviceManualRepairRecord', recordMsg,
            headers)
        .then(
            res => res,
            err => {
              Promise.all([
                dispatch(receiveRecordInfoError(err)),
                dispatch(receiveAppMessage(err.description)),
              ]);
              throw Error(err.description);
            },
        );
  };
};

/**
 * Clears all device, repair info, and repair history. Sets all to empty states
 * and returns a promise when both are resolved.
 */
export function clearRepairRecord() {
  return function(dispatch: AppThunkDispatch) {
    return Promise.all([
      dispatch(clearDeviceInfo()),
      dispatch(clearRecordInfo()),
      dispatch(clearRepairHistory()),
    ]);
  };
};

/**
 * Call inventory.Inventory/ListManualRepairRecords rpc for manual repair
 * record history or a DUT. Response is saved to Redux state.
 *
 * @param hostname  The hostname of the device.
 * @param assetTag  The asset tag of the device.
 * @param headers   The additional HTML headers to be passed. These will include
 *     auth headers for user auth.
 * @returns         The response from the RPC.
 */
export function getRepairHistory(
    hostname: string, assetTag: string,
    headers: {[key: string]: string}): AppThunk<Promise<object>> {
  return function(dispatch: AppThunkDispatch) {
    const recordMsg: {[key: string]: any} = {
      'hostname': hostname,
      'asset_tag': assetTag,
      'limit': -1,
    };
    return inventoryClient
        .call(
            inventoryApiVersion, 'ListManualRepairRecords', recordMsg, headers)
        .then(
            res => dispatch(receiveRepairHistory(res)),
            err => {
              dispatch(receiveAppMessage(err.description));
              throw Error(err.description);
            },
        );
  };
};

export type RepairRecordStateType = {
  info: {
    deviceInfo: object,
    recordInfo: object,
    recordId: string,
    repairHistory: object,
  },
  errors: {
    deviceInfoError: object,
    recordInfoError: object,
  },
}

const emptyState: RepairRecordStateType = {
  info: {
    deviceInfo: {},
    recordInfo: {},
    recordId: '',
    repairHistory: {},
  },
  errors: {
    deviceInfoError: {},
    recordInfoError: {},
  },
};

export function repairRecordReducer(state = emptyState, action) {
  switch (action.type) {
    case RECEIVE_DEVICE_INFO:
      return {
        ...state,
        info: {
          deviceInfo: action.deviceInfo,
          recordInfo: state.info.recordInfo,
          recordId: state.info.recordId,
          repairHistory: state.info.repairHistory,
        }
      };
    case RECEIVE_RECORD_INFO:
      return {
        ...state,
        info: {
          deviceInfo: state.info.deviceInfo,
          recordInfo: action.recordInfo.deviceRepairRecord,
          recordId: action.recordInfo.id,
          repairHistory: state.info.repairHistory,
        }
      };
    case RECEIVE_REPAIR_HISTORY:
      return {
        ...state,
        info: {
          deviceInfo: state.info.deviceInfo,
          recordInfo: state.info.recordInfo,
          recordId: state.info.recordId,
          repairHistory: action.repairHistory,
        }
      };
    case RECEIVE_RECORD_INFO_ERROR:
      return {
        ...state,
        info: {
          deviceInfo: state.info.deviceInfo,
          recordInfo: emptyState.info.recordInfo,
          recordId: emptyState.info.recordId,
          repairHistory: emptyState.info.repairHistory,
        },
        errors: {
          ...state.errors,
          recordInfoError: action.error,
        },
      };
    case RECEIVE_DEVICE_INFO_ERROR:
      return {
        ...state,
        info: {
          deviceInfo: emptyState.info.deviceInfo,
          recordInfo: state.info.recordInfo,
          recordId: state.info.recordId,
          repairHistory: state.info.repairHistory,
        },
        errors: {
          ...state.errors,
          deviceInfoError: action.error,
        },
      };
    case CLEAR_DEVICE_INFO:
      return {
        ...state,
        info: {
          deviceInfo: emptyState.info.deviceInfo,
          recordInfo: state.info.recordInfo,
          recordId: state.info.recordId,
          repairHistory: state.info.repairHistory,
        },
      };
    case CLEAR_RECORD_INFO:
      return {
        ...state,
        info: {
          deviceInfo: state.info.deviceInfo,
          recordInfo: emptyState.info.recordInfo,
          recordId: emptyState.info.recordId,
          repairHistory: state.info.repairHistory,
        },
      };
    case CLEAR_REPAIR_HISTORY:
      return {
        ...state,
        info: {
          deviceInfo: state.info.deviceInfo,
          recordInfo: state.info.recordInfo,
          recordId: state.info.recordId,
          repairHistory: emptyState.info.repairHistory,
        },
      };
    default:
      return state;
  };
};
