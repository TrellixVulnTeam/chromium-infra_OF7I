// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {RECEIVE_DEVICE_INFO, RECEIVE_DEVICE_INFO_ERROR, RECEIVE_RECORD_INFO, RECEIVE_RECORD_INFO_ERROR, RECEIVE_USER} from './actions';

const INITIAL_STATE = {
  user: {signedIn: false, profile: null, authHeaders: null},
  repairRecord: {
    deviceInfo: null,
    recordInfo: null,
  },
  errors: {
    deviceInfoError: null,
    recordInfoError: null,
  },
};

// TODO: Split this reducer into multiple reducers.
export const reducer = (state = INITIAL_STATE, action) => {
  switch (action.type) {
    case RECEIVE_USER:
      return {...state, user: action.user};
    case RECEIVE_DEVICE_INFO:
      return {
        ...state,
        repairRecord: {
          deviceInfo: action.deviceInfo,
          recordInfo: state.repairRecord.recordInfo,
        }
      };
    case RECEIVE_RECORD_INFO:
      return {
        ...state,
        repairRecord: {
          deviceInfo: state.repairRecord.deviceInfo,
          recordInfo: action.recordInfo,
        }
      };
    case RECEIVE_RECORD_INFO_ERROR:
      return {
        ...state,
        errors: {...state.errors, recordInfoError: action.error}
      };
    case RECEIVE_DEVICE_INFO_ERROR:
      return {
        ...state,
        errors: {...state.errors, deviceInfoError: action.error}
      };
    default:
      return state;
  }
}
