// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

import { AppState } from '../../app/store';

const LOCAL_STORAGE_KEY = 'prefs';

/**
 * The preferences slice stores and loads UI preferences to/from local storage.
 * It is used to record which metrics/periods to show by default for each
 * data source.
 *
 * The slice is organized using Redux Toolkit (https://redux-toolkit.js.org/)
 * conventions.
 */

// Preferences stored in local storage.  This entire object is stringified
// and stored in local storage, so keep in mind that as new fields are added
// to this object, they will show up as empty on first load.
export interface PreferencesState {
  // Number of periods to show in the UI
  numPeriods?: number;
  // DataSource-specific preferences. Key is the name of the dataSource.
  dataSources: { [name: string]: DataSourcePreferences };
}

// Preferences stored for the given data source.
export interface DataSourcePreferences {
  // List of metrics to show.  This brings up what was last viewed by the user.
  metrics?: string[];
}

const initialState: PreferencesState = {
  numPeriods: 4,
  dataSources: {},
};

export interface UpdateDataSourcePreferences {
  dataSource: string;
  metrics?: string[];
}

// This selector is used by the PreferencesStore component to run
// savePreferences whenever the store is updated.
export const selectPreferences = (state: AppState): PreferencesState =>
  state.preferences;

const preferencesSlice = createSlice({
  name: 'preferences',
  initialState,
  reducers: {
    updatePreferencesNumPeriods(state, action: PayloadAction<number>) {
      state.numPeriods = action.payload;
    },
    updatePreferencesDataSourceMetrics(
      state,
      action: PayloadAction<UpdateDataSourcePreferences>
    ) {
      if (action.payload.dataSource in state.dataSources) {
        state.dataSources[action.payload.dataSource].metrics =
          action.payload.metrics;
      } else {
        state.dataSources[action.payload.dataSource] = {
          metrics: action.payload.metrics,
        };
      }
    },
  },
});

export const {
  updatePreferencesNumPeriods,
  updatePreferencesDataSourceMetrics,
} = preferencesSlice.actions;

export default preferencesSlice.reducer;

// Updates local storage with the saved preferences.  This is called whenever
// the preferences state is updated.
export function savePreferences(state: PreferencesState): void {
  const jsonPreferences = JSON.stringify(state);
  localStorage.setItem(LOCAL_STORAGE_KEY, jsonPreferences);
}

// Loads preferences from local storage.
export function loadPreferences(): PreferencesState | undefined {
  const jsonPreferences = localStorage.getItem(LOCAL_STORAGE_KEY);
  try {
    if (jsonPreferences === null) return;
    return JSON.parse(jsonPreferences) as PreferencesState;
  } catch (e) {
    return;
  }
}
