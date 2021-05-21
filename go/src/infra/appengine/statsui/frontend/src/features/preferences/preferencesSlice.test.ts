// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import preferencesReducer, {
  updatePreferencesDataSourceMetrics,
  updatePreferencesNumPeriods,
} from './preferencesSlice';

// Tests the updatePreferencesNumPeriods reducer
describe('updatePreferencesNumPeriods', () => {
  it('updates num periods', () => {
    const action = updatePreferencesNumPeriods(6);
    const state = preferencesReducer(undefined, action);

    expect(state.numPeriods).toEqual(action.payload);
  });
});

// Tests the updatePreferencesNumPeriods reducer
describe('updatePreferencesDataSourceMetrics', () => {
  it('updates metrics', () => {
    const action = updatePreferencesDataSourceMetrics({
      dataSource: 'testDataSource',
      metrics: ['testMetric1', 'testMetric2'],
    });
    const state = preferencesReducer(undefined, action);

    const dataSource = action.payload.dataSource;
    expect(state.dataSources).toHaveProperty(dataSource);
    expect(state.dataSources[dataSource].metrics).toEqual(
      action.payload.metrics
    );
  });
});

// TODO(gatong): Add more tests here.
