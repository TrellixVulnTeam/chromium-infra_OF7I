// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

import {
  DataSet,
  fetchMetrics,
  FetchMetricsResponse,
  Metric,
} from '../../api/metrics';
import { AppDispatch, AppState, AppThunk } from '../../app/store';
import { merge, removeFrom } from '../../utils/arrayUtils';
import { calculateValidDates, Period } from '../../utils/dateUtils';

/**
 * The metrics slice does the following:
 * - Authority on what dates and metrics are visible in the UI.
 * - Stores the current data source, period, and number of periods to show.
 * - Uses the above data to load and update a cache of metrics data from the
 *   backend as necessary.
 * - Calculates the currently visible data set.
 *
 * While it has internal actions to update state, all actions have the potential
 * of triggering an async cache update, and thus are exported as thunks.
 *
 * The slice is organized using Redux Toolkit (https://redux-toolkit.js.org/)
 * conventions.
 */

/*
 The section keys are arbitrary strings that name the section.
 The metric keys are dates in the form of YYYY-MM-DD
 */
export type MetricsData = {
  [section: string]: { [metric: string]: Metric };
};

export interface MetricsState {
  // What dates to show for columns
  visibleDates: string[];
  visibleMetrics: string[];
  visibleData: MetricsData;

  // Settings
  dataSource: string;
  period: Period;
  numPeriods: number;
  maxDate: string;

  // Local data cache
  precachePeriods: number;
  cachedDates: string[];
  cachedMetrics: string[];
  loadingDates: string[];
  loadingMetrics: string[];
  cache: MetricsData;
}

const initialState: MetricsState = {
  visibleDates: [],
  visibleMetrics: [],
  visibleData: {},

  dataSource: '',
  period: Period.Undefined,
  numPeriods: 4,
  maxDate: calculateValidDates(new Date(), Period.Week, 0, 0)[0],

  precachePeriods: 3,
  cachedDates: [],
  cachedMetrics: [],
  loadingDates: [],
  loadingMetrics: [],
  cache: {},
};

// selectMaxDate determines the maximum date that can be loaded.
export const selectMaxDate = (state: AppState): string => state.metrics.maxDate;
// selectNumPeriods determines how many periods are shown in visibleData
export const selectNumPeriods = (state: AppState): number =>
  state.metrics.numPeriods;
// selectVisibleDates lists the currently visible dates
export const selectVisibleDates = (state: AppState): string[] =>
  state.metrics.visibleDates;
// selectVisibleMetrics lists the currently visible metrics
export const selectVisibleMetrics = (state: AppState): string[] =>
  state.metrics.visibleMetrics;
// selectCurrentPeriod shows the current Period (Day, Month, etc)
export const selectCurrentPeriod = (state: AppState): Period =>
  state.metrics.period;
// selectShowLoading returns whether any data is currently being loaded.
export const selectShowLoading = (state: AppState): boolean =>
  state.metrics.visibleDates.some(
    (date) => !state.metrics.cachedDates.includes(date)
  ) ||
  state.metrics.visibleMetrics.some(
    (metric) => !state.metrics.cachedMetrics.includes(metric)
  );
// selectVisibleDate returns the visible data from the cache.
export const selectVisibleData = (state: AppState): MetricsData =>
  state.metrics.visibleData;

// Creates a version of MetricsData with only the visible metrics and dates
function calculateVisibleData({
  cache,
  visibleMetrics,
  visibleDates,
}: {
  cache: MetricsData;
  visibleMetrics: string[];
  visibleDates: string[];
}): MetricsData {
  const ret: MetricsData = {};
  Object.keys(cache).forEach((sectionName) => {
    const section: { [metricName: string]: Metric } = {};
    visibleMetrics.forEach((metricName) => {
      if (metricName in cache[sectionName]) {
        const cacheMetric = cache[sectionName][metricName];
        const metric: Metric = {
          name: cacheMetric.name,
        };
        let hasData = false;
        if (cacheMetric.data != undefined) {
          metric.data = createVisibleDataSet(cacheMetric.data, visibleDates);
          hasData = hasData || Object.keys(metric.data).length > 0;
        }
        if (cacheMetric.sections != undefined) {
          metric.sections = {};
          Object.keys(cacheMetric.sections).forEach((subSectionName) => {
            if (
              cacheMetric.sections === undefined ||
              metric.sections === undefined
            )
              return; // Needed for lint
            const dataSet = createVisibleDataSet(
              cacheMetric.sections[subSectionName],
              visibleDates
            );
            if (Object.keys(dataSet).length > 0) {
              metric.sections[subSectionName] = dataSet;
              hasData = true;
            }
          });
        }
        if (hasData) {
          section[metricName] = metric;
        }
      }
    });
    if (Object.keys(section).length > 0) {
      ret[sectionName] = section;
    }
  });
  return ret;
}

// Creates a DataSet from the original DataSet containing only the given dates.
function createVisibleDataSet(data: DataSet, dates: string[]): DataSet {
  const ret: DataSet = {};
  for (const date of dates) {
    if (date in data) {
      ret[date] = data[date];
    }
  }
  return ret;
}

interface FetchMetricsStart {
  dates: string[];
  metrics: string[];
}

interface FetchMetricsSuccess {
  dates: string[];
  metrics: string[];
  response: FetchMetricsResponse;
}

const metricsSlice = createSlice({
  name: 'metrics',
  initialState,
  reducers: {
    updateDataSource(state, action: PayloadAction<string>) {
      state.dataSource = action.payload;
    },
    updatePeriod(state, action: PayloadAction<Period>) {
      state.period = action.payload;
      state.maxDate = calculateValidDates(new Date(), action.payload, 0, 0)[0];
    },
    updateNumPeriods(state, action: PayloadAction<number>) {
      state.numPeriods = action.payload;
    },
    updateVisibleDates(state, action: PayloadAction<string[]>) {
      state.visibleDates = action.payload;
      state.visibleData = calculateVisibleData(state);
    },
    updateVisibleMetrics(state, action: PayloadAction<string[]>) {
      state.visibleMetrics = action.payload;
      state.visibleData = calculateVisibleData(state);
    },
    resetCache(state) {
      state.loadingDates = [];
      state.loadingMetrics = [];
      state.cachedDates = [];
      state.cachedMetrics = [];
      state.cache = {};
    },
    fetchMetricsStart(state, action: PayloadAction<FetchMetricsStart>) {
      state.loadingDates = merge(state.loadingDates, action.payload.dates);
      state.loadingMetrics = merge(
        state.loadingMetrics,
        action.payload.metrics
      );
    },
    fetchMetricsSuccess(state, action: PayloadAction<FetchMetricsSuccess>) {
      state.loadingDates = removeFrom(state.loadingDates, action.payload.dates);
      state.loadingMetrics = removeFrom(
        state.loadingMetrics,
        action.payload.metrics
      );
      state.cachedDates = merge(state.cachedDates, action.payload.dates);
      state.cachedMetrics = merge(state.cachedMetrics, action.payload.metrics);

      const response = action.payload.response;
      Object.keys(response.sections).forEach((sectionName) => {
        if (!(sectionName in state.cache)) {
          state.cache[sectionName] = {};
        }
        for (const metric of response.sections[sectionName]) {
          if (!(metric.name in state.cache[sectionName])) {
            state.cache[sectionName][metric.name] = {
              name: metric.name,
            };
          }
          const cacheMetric = state.cache[sectionName][metric.name];
          if (metric.data != undefined) {
            if (cacheMetric.data == undefined) {
              cacheMetric.data = {};
            }
            // Merge dates from metric into cache
            Object.assign(cacheMetric.data, metric.data);
          } else if (metric.sections != undefined) {
            Object.keys(metric.sections).forEach((subSectionName) => {
              if (metric.sections == undefined) return;
              if (cacheMetric.sections == undefined) {
                cacheMetric.sections = {};
              }
              if (!(subSectionName in cacheMetric.sections)) {
                cacheMetric.sections[subSectionName] = {};
              }
              // Merge dates from metric sections into cache
              Object.assign(
                cacheMetric.sections[subSectionName],
                metric.sections[subSectionName]
              );
            });
          }
        }
      });
      state.visibleData = calculateVisibleData(state);
    },
  },
});

const {
  updateDataSource,
  updatePeriod,
  updateNumPeriods,
  updateVisibleDates,
  updateVisibleMetrics,
  resetCache,
  fetchMetricsStart,
  fetchMetricsSuccess,
} = metricsSlice.actions;

export const actions = metricsSlice.actions;

export default metricsSlice.reducer;

// Sets the data source to use.  This will reset but not repopulate the cache.
export const setDataSource = (dataSource: string): AppThunk => async (
  dispatch,
  getState
) => {
  const state = getState();
  if (dataSource === state.metrics.dataSource) {
    return;
  }
  await dispatch(resetCache());
  await dispatch(updateDataSource(dataSource));
};

// Updates the visible dates.  This will fetch the visible metrics for any dates
// that are not already cached.
export const setDates = (date: Date | string): AppThunk => async (
  dispatch,
  getState
) => {
  const state = getState();
  const dates = calculateValidDates(
    date,
    state.metrics.period,
    state.metrics.numPeriods - 1
  );
  await showMetrics(
    dispatch,
    state.metrics,
    dates,
    state.metrics.visibleMetrics
  );
};

// Updates the visible metrics.  This will fetch metrics for any visible dates
// that are not already cached.
export const setMetrics = (metrics: string[]): AppThunk => async (
  dispatch,
  getState
) => {
  const state = getState();
  await showMetrics(
    dispatch,
    state.metrics,
    state.metrics.visibleDates,
    metrics
  );
};

// Change the period shown.  This will reset and repopulate the cache for the
// new period.
export const setPeriod = (period: Period): AppThunk => async (
  dispatch,
  getState
) => {
  const state = getState();
  if (period === state.metrics.period) {
    return;
  }
  await dispatch(resetCache());
  await dispatch(updatePeriod(period));
  if (state.metrics.visibleDates.length > 0) {
    const maxVisibleDate =
      state.metrics.visibleDates[state.metrics.visibleDates.length - 1];
    const targetDate =
      state.metrics.maxDate == maxVisibleDate ? new Date() : maxVisibleDate;
    dispatch(setDates(targetDate));
  }
};

// Set the number of periods to show.  This will change how many visible dates
// there are, as well as fetch any needed data.
export const setNumPeriods = (numPeriods: number): AppThunk => async (
  dispatch,
  getState
) => {
  const state = getState();
  await dispatch(updateNumPeriods(numPeriods));
  if (state.metrics.visibleDates.length > 0) {
    dispatch(
      setDates(
        state.metrics.visibleDates[state.metrics.visibleDates.length - 1]
      )
    );
  }
};

// Increment the visible dates by the period.  This will fetch and cache any
// missing data.
export const incrementDates = (): AppThunk => async (dispatch, getState) => {
  const state = getState();
  const dates = calculateValidDates(
    state.metrics.visibleDates[0],
    state.metrics.period,
    0,
    state.metrics.numPeriods,
    false
  );
  await showMetrics(
    dispatch,
    state.metrics,
    dates,
    state.metrics.visibleMetrics
  );
};

// Decrement the visible dates by the period.  This will fetch and cache any
// missing data.
export const decrementDates = (): AppThunk => async (dispatch, getState) => {
  const state = getState();
  const dates = calculateValidDates(
    state.metrics.visibleDates[0],
    state.metrics.period,
    1,
    state.metrics.numPeriods - 2
  );
  await showMetrics(
    dispatch,
    state.metrics,
    dates,
    state.metrics.visibleMetrics
  );
};

async function showMetrics(
  dispatch: AppDispatch,
  state: MetricsState,
  dates: string[],
  metrics: string[]
) {
  const datesToLoad = merge(
    dates.slice(),
    calculateValidDates(
      dates[0],
      state.period,
      state.precachePeriods,
      0,
      false
    ),
    calculateValidDates(
      dates[dates.length - 1],
      state.period,
      0,
      state.precachePeriods,
      false
    )
  );
  const metricsToLoad = metrics.slice();

  const datesNeeded = removeFrom(
    datesToLoad,
    state.cachedDates,
    state.loadingDates
  );
  const metricsNeeded = removeFrom(
    metricsToLoad,
    state.cachedMetrics,
    state.loadingMetrics
  );

  dispatch(updateVisibleDates(dates));
  dispatch(updateVisibleMetrics(metrics));

  if (datesNeeded.length > 0 && metricsNeeded.length > 0) {
    // Need both new dates and new metrics.  Do a cache reset
    dispatch(resetCache());
    await fetchMetricsAndPopulateCache(
      dispatch,
      state.dataSource,
      state.period,
      datesToLoad,
      metricsToLoad
    );
  } else if (datesNeeded.length > 0) {
    // Load new dates for all cached & loading metrics
    await fetchMetricsAndPopulateCache(
      dispatch,
      state.dataSource,
      state.period,
      datesNeeded,
      merge(state.cachedMetrics, state.loadingMetrics)
    );
  } else if (metricsNeeded.length > 0) {
    // Load new metrics for all cached & loading dates
    await fetchMetricsAndPopulateCache(
      dispatch,
      state.dataSource,
      state.period,
      merge(state.cachedDates, state.loadingDates),
      metricsNeeded
    );
  }
}

async function fetchMetricsAndPopulateCache(
  dispatch: AppDispatch,
  dataSource: string,
  period: Period,
  dates: string[],
  metrics: string[]
) {
  try {
    dispatch(fetchMetricsStart({ dates: dates, metrics: metrics }));
    const response = await fetchMetrics(dataSource, period, dates, metrics);
    dispatch(
      fetchMetricsSuccess({
        dates: dates,
        metrics: metrics,
        response: response,
      })
    );
  } catch (err) {
    // TODO: Handle errors
    console.log(err);
  }
}
