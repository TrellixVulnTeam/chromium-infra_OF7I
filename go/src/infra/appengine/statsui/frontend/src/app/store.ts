import { configureStore, ThunkAction } from '@reduxjs/toolkit';
import { Action } from 'redux';

import metricsReducer, { MetricsState } from '../features/metrics/metricsSlice';

export interface AppState {
  metrics: MetricsState;
}

const store = configureStore({
  reducer: {
    metrics: metricsReducer,
  },
});

export type AppDispatch = typeof store.dispatch;

export type AppThunk = ThunkAction<void, AppState, unknown, Action<string>>;

export default store;
