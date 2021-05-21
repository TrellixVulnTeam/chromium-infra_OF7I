// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

import { AppState, AppThunk } from '../../app/store';
import { Period, toTzDate } from '../../utils/dateUtils';
import { Unit } from '../../utils/formatUtils';
import {
  setDataSource,
  setDates,
  setMetrics,
  setNumPeriods,
  setPeriod,
} from '../metrics/metricsSlice';

/**
 * The dataSources slice holds information about the data sources available
 * on the backend.  Specifically, it knows:
 * - The name (and pretty name) of the data source.  The name is used as the
 *   identifier for the data source internally, in URLs, and in caches.
 * - The backend API that will provide the data for that data source.
 * - What label to use for the initial section header.  For example, for the
 *   'cq-builders' data source, the section name is "Builders"
 * - The metrics that the data source supports.
 * - The periods (Day, Week, Month, etc) that the data source supports.
 *
 * This data structure may migrate to the backend eventually so that the backend
 * is responsible for letting the UI know what kind of data is available.
 *
 * The slice is organized using Redux Toolkit (https://redux-toolkit.js.org/)
 * conventions.
 */

/*
 Catalog of all available data sources, as well as the currently selected
 dataSource.
*/
export interface DataSourcesState {
  current: string;
  available: DataSource[];
  // Key is the name of the dataSource.  This is automatically built by
  // populateMaps
  availableMap: { [name: string]: DataSource };
}

/*
 Definition for a single data source.
*/
export interface DataSource {
  name: string;
  prettyName: string;
  apiDataSource: string;
  sectionName: string;
  metrics: MetricOption[];
  periods: PeriodOption[];
  // Key is the name of the metric.  This is automatically built by
  // populateMaps
  metricMap: { [name: string]: MetricOption };
}

// Specification of available metrics for a dataSource.
export interface MetricOption {
  // Name is the name of the metric and is used both as a display value and as
  // an identifier.
  name: string;
  // Unit specifies the base unit of the metric, such as a number, duration, etc
  // This is used for formatting purposes.
  unit: Unit;
  // Description is shown in the UI to explain what a given metric measures.
  description: string;
  // isDefault determines whether the metric is shown by default for a given
  // dataSource.
  isDefault?: boolean;
  // hasSubsections specifies whether the given metric is a single value or is
  // broken out by subSections.
  hasSubsections?: boolean;
}

// Specification of available periods for a dataSource.
export interface PeriodOption {
  // Name is used as a display value.
  name: string;
  // The Period enum that this option refers to.
  period: Period;
  // isDefault determins whether the metric is shown by default for a given
  // dataSource.
  isDefault?: boolean;
}

// Fills in the availableMap object on DataSourcesState, and the metricMap
// object on the DataSource.  This mutates DataSourceState, but returns it
// as well so that the function can be chained.
const populateMaps = (state: DataSourcesState): DataSourcesState => {
  state.available.forEach((dataSource) => {
    state.availableMap[dataSource.name] = dataSource;
    for (const metric of dataSource.metrics) {
      dataSource.metricMap[metric.name] = metric;
    }
  });
  return state;
};

const initialState: DataSourcesState = populateMaps({
  current: '',
  available: [
    {
      name: 'cq-builders',
      apiDataSource: 'cq_builders',
      prettyName: 'CQ Builders',
      sectionName: 'Builders',
      metrics: [
        {
          name: 'P50',
          unit: Unit.Duration,
          description: `Time from create_time to end_time (Pending + Runtime)`,
        },
        {
          name: 'P90',
          isDefault: true,
          unit: Unit.Duration,
          description: `Time from create_time and end_time (Pending + Runtime)`,
        },
        {
          name: 'Count',
          unit: Unit.Number,
          description: `How many times this builder ran`,
        },
        {
          name: 'P50 Runtime',
          unit: Unit.Duration,
          description: `Time from start_time to end_time (actual execution
          time)`,
        },
        {
          name: 'P90 Runtime',
          unit: Unit.Duration,
          description: `Time from start_time to end_time (actual execution
          time)`,
        },
        {
          name: 'P50 Pending',
          unit: Unit.Duration,
          description: `Time from create_time to start_time (builder wait
          time)`,
        },
        {
          name: 'P90 Pending',
          unit: Unit.Duration,
          description: `Time from create_time to start_time (builder wait
          time)`,
        },
        {
          name: 'P50 Phase Runtime',
          unit: Unit.Duration,
          hasSubsections: true,
          description: `Duration of the 4 main builder phases for the "with
          patch" section of the build.  Retries and some phases, such as isolate
          tests, are not included here.`,
        },
        {
          name: 'P90 Phase Runtime',
          isDefault: true,
          unit: Unit.Duration,
          hasSubsections: true,
          description: `Duration of the 4 main builder phases for the "with
          patch" section of the build.  Retries and some phases, such as isolate
          tests, are not included here.`,
        },
        {
          name: 'P50 Slow Tests',
          unit: Unit.Duration,
          hasSubsections: true,
          description: `P50 runtime of slow tests, which is defined as tests
          where the P50 runtime exceeded 5 minutes, or the P90 runtime exceeded
          10 minutes. The runtime is of the slowest shard, so if a test has 4
          shards that ran in 4m, 5m, 6m, and 7m, the duration for that test
          would be 7m.`,
        },
        {
          name: 'P90 Slow Tests',
          unit: Unit.Duration,
          hasSubsections: true,
          description: `P90 runtime of slow tests, which is defined as tests
          where the P50 runtime exceeded 5 minutes, or the P90 runtime exceeded
          10 minutes. The runtime is of the slowest shard, so if a test has 4
          shards that ran in 4m, 5m, 6m, and 7m, the duration for that test
          would be 7m.`,
        },
        {
          name: 'Count Slow Tests',
          unit: Unit.Number,
          hasSubsections: true,
          description: `How many builds had a shard that ran longer than 5
          minutes`,
        },
      ],
      periods: [
        {
          name: 'Day',
          period: Period.Day,
        },
        {
          name: 'Week',
          period: Period.Week,
          isDefault: true,
        },
      ],
      metricMap: {},
    },
  ],
  availableMap: {},
});

const empty: DataSource = {
  name: '',
  apiDataSource: '',
  prettyName: '',
  sectionName: '',
  metrics: [],
  periods: [],
  metricMap: {},
};

// Shows the currently selected source, or the empty DataSource if one
// is not selected.
export const selectCurrentSource = (state: AppState): DataSource => {
  if (state.dataSources.current === '') return empty;
  return state.dataSources.availableMap[state.dataSources.current];
};

// Returns the currently available data sources.
export const selectAvailable = (state: AppState): DataSource[] =>
  state.dataSources.available;

const dataSourcesSlice = createSlice({
  name: 'dataSources',
  initialState,
  reducers: {
    updateCurrent(state, action: PayloadAction<string>) {
      state.current = action.payload;
    },
  },
});

const { updateCurrent } = dataSourcesSlice.actions;

export const actions = dataSourcesSlice.actions;

export default dataSourcesSlice.reducer;

// Sets the current data source.  This will either use defaults for what metrics
// and period to show, or pull them from local storage preferences if available.
export const setCurrent = (
  name: string,
  params?: URLSearchParams
): AppThunk => async (dispatch, getState) => {
  const state = getState();
  if (!(name in state.dataSources.availableMap)) {
    return;
  }
  if (state.dataSources.current === name) {
    return;
  }

  const dataSource = state.dataSources.availableMap[name];

  // Sets the metrics to show based on URL parameters, local storage
  // preferences, or whatever is default for the data source.
  let metrics: string[] = [];
  if (params !== undefined && params.has('metric')) {
    metrics = params
      .getAll('metric')
      .filter((metric) => metric in dataSource.metricMap);
  } else if (
    name in state.preferences.dataSources &&
    state.preferences.dataSources[name].metrics !== undefined
  ) {
    metrics = (state.preferences.dataSources[name].metrics as string[]).filter(
      (metric) => metric in dataSource.metricMap
    );
  } else {
    for (const metric of dataSource.metrics) {
      if (metric.isDefault) {
        metrics.push(metric.name);
      }
    }
  }

  let period = dataSource.periods[0].period;
  if (params !== undefined && params.has('period')) {
    period = params.get('period') as Period;
  } else {
    for (const p of dataSource.periods) {
      if (p.isDefault) {
        period = p.period;
      }
    }
  }

  if (params !== undefined && params.has('periods')) {
    const periods = params.get('periods');
    if (periods != null) {
      const numPeriods = parseInt(periods);
      await dispatch(setNumPeriods(numPeriods));
    }
  } else if (state.preferences.numPeriods !== undefined) {
    await dispatch(setNumPeriods(state.preferences.numPeriods));
  }

  let date = new Date();
  if (params !== undefined && params.has('date')) {
    date = toTzDate(params.get('date') as string);
  }

  // TODO(gatong): This is a really ugly set of dispatches that needs to be
  // refactored. Should be replaced with a single update.
  await dispatch(updateCurrent(name));
  await dispatch(setDataSource(dataSource.apiDataSource));
  await dispatch(setPeriod(period));
  await dispatch(setMetrics(metrics));
  await dispatch(setDates(date));
};
