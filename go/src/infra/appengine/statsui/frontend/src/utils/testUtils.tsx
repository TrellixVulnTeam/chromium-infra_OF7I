// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { ReactElement } from 'react';
import { render, RenderResult } from '@testing-library/react';
import { configureStore, DeepPartial } from '@reduxjs/toolkit';
import { Provider } from 'react-redux';
import { MemoryRouter } from 'react-router-dom';
// Reducers
import { Period } from './dateUtils';
import { Unit } from './formatUtils';
import dataSourcesReducer, {
  DataSource,
  DataSourcesState,
  MetricOption,
} from '../features/dataSources/dataSourcesSlice';
import metricsReducer, { MetricsState } from '../features/metrics/metricsSlice';
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
  return render(
    <MemoryRouter>
      <Provider store={store}>{ui}</Provider>
    </MemoryRouter>
  );
}

const emptyMetricsState: MetricsState = {
  visibleDates: [],
  visibleMetrics: [],
  visibleData: {},

  dataSource: '',
  period: Period.Undefined,
  numPeriods: 4,
  maxDate: '2021-01-02',

  precachePeriods: 0,
  cachedDates: [],
  cachedMetrics: [],
  loadingDates: [],
  loadingMetrics: [],
  cache: {},
};

export function initMetricsState(state: Partial<MetricsState>): MetricsState {
  return Object.assign({}, emptyMetricsState, state);
}

// Sets up a current DataSource with the given metric
export function initDataSourceWithMetrics(
  name: string,
  sectionName: string,
  ...metrics: Partial<MetricOption>[]
): DataSourcesState {
  const ds: DataSource = {
    name: name,
    prettyName: name,
    apiDataSource: '',
    sectionName: sectionName,
    metrics: [],
    periods: [{ name: 'Week', period: Period.Week }],
    metricMap: {},
  };
  metrics.forEach((metric) => {
    const m: MetricOption = Object.assign(
      {
        name: '',
        unit: Unit.Number,
        description: '',
      },
      metric
    );
    ds.metrics.push(m);
    ds.metricMap[m.name] = m;
  });

  const state: DataSourcesState = {
    current: ds.name,
    available: [ds],
    availableMap: {},
  };
  state.availableMap[name] = ds;
  return state;
}
