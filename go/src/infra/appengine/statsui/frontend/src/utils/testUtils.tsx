// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { ReactElement } from 'react';
import { render, RenderResult } from '@testing-library/react';
import { configureStore, DeepPartial } from '@reduxjs/toolkit';
import { Provider } from 'react-redux';
// Reducers
import dataSourcesReducer from '../features/dataSources/dataSourcesSlice';
import metricsReducer from '../features/metrics/metricsSlice';
import preferencesReducer from '../features/preferences/preferencesSlice';
import { AppState } from '../app/store';

// Replacement for react's render function that wraps the component in a redux
// store. Used for testing components that use redux.
export function renderWithRedux(
  ui: ReactElement,
  initialState: DeepPartial<AppState> = {}
): RenderResult {
  const store = configureStore({
    reducer: {
      dataSources: dataSourcesReducer,
      metrics: metricsReducer,
      preferences: preferencesReducer,
    },
    preloadedState: initialState,
  });
  return render(<Provider store={store}>{ui}</Provider>);
}
