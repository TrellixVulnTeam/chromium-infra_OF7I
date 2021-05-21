// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { configureStore, ThunkAction } from '@reduxjs/toolkit';
import { Action } from 'redux';

import dataSourcesReducer from '../features/dataSources/dataSourcesSlice';
import { DataSourcesState } from '../features/dataSources/dataSourcesSlice';
import metricsReducer, { MetricsState } from '../features/metrics/metricsSlice';
import preferencesReducer, {
  loadPreferences,
  PreferencesState,
} from '../features/preferences/preferencesSlice';

export interface AppState {
  dataSources: DataSourcesState;
  metrics: MetricsState;
  preferences: PreferencesState;
}

const preloadedState = {
  preferences: loadPreferences(),
};

const store = configureStore({
  reducer: {
    dataSources: dataSourcesReducer,
    metrics: metricsReducer,
    preferences: preferencesReducer,
  },
  preloadedState,
});

export type AppDispatch = typeof store.dispatch;

export type AppThunk = ThunkAction<void, AppState, unknown, Action<string>>;

export default store;
