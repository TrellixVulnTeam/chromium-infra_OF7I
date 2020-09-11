// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {combineReducers, Reducer} from 'redux';

import {repairRecordReducer, RepairRecordStateType} from './repair-record';
import {userReducer, UserStateType} from './user';

export interface ApplicationState {
  record: RepairRecordStateType;
  user: UserStateType;
}

export const reducers: Reducer<ApplicationState> =
    combineReducers<ApplicationState>({
      record: repairRecordReducer,
      user: userReducer,
    });
