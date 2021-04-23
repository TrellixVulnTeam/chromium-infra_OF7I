// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import metricsReducer, {
  actions,
  MetricsData,
  MetricsState,
} from './metricsSlice';
import { Period, toTzDate } from '../../utils/dateUtils';
import MockDate from 'mockdate';
import { FetchMetricsResponse } from '../../api/metrics';

const emptyState: MetricsState = {
  visibleDates: [],
  visibleMetrics: [],
  visibleData: {},

  dataSource: '',
  period: Period.Undefined,
  numPeriods: 1,
  maxDate: '2021-01-02',

  precachePeriods: 0,
  cachedDates: [],
  cachedMetrics: [],
  loadingDates: [],
  loadingMetrics: [],
  cache: {},
};

// Tests the updateDataSource reducer
describe('updateDataSource', () => {
  it('updates state', () => {
    const action = actions.updateDataSource('testDataSource');
    const state = metricsReducer(undefined, action);

    expect(state.dataSource).toEqual(action.payload);
  });
});

// Tests the updatePeriod reducer
describe('updatePeriod', () => {
  afterEach(() => {
    MockDate.reset();
  });
  it('updates state', () => {
    MockDate.set(toTzDate('2021-01-02'));

    const action = actions.updatePeriod(Period.Day);
    const state = metricsReducer(undefined, action);

    expect(state.period).toEqual(action.payload);
    expect(state.maxDate).toEqual('2021-01-02');
  });
});

// Tests the updateNumPeriods reducer
describe('updateNumPeriods', () => {
  it('updates state', () => {
    const action = actions.updateNumPeriods(4);
    const state = metricsReducer(undefined, action);

    expect(state.numPeriods).toEqual(action.payload);
  });
});

// Tests the updateVisibleDates reducer
describe('updateVisibleDates', () => {
  it('updates dates but visible data is empty', () => {
    const action = actions.updateVisibleDates(['2021-01-02', '2021-02-03']);
    const state = metricsReducer(undefined, action);

    expect(state.visibleDates).toEqual(action.payload);
    expect(state.visibleData).toMatchObject({});
  });
});

// Tests the updateVisibleMetrics reducer
describe('updateVisibleMetrics', () => {
  it('updates metrics but visible data is empty', () => {
    const action = actions.updateVisibleMetrics(['M1', 'M2']);
    const state = metricsReducer(undefined, action);

    expect(state.visibleMetrics).toEqual(action.payload);
    expect(state.visibleData).toMatchObject({});
  });
});

// Tests that visibleData is properly populated.  Updating visibleMetrics or
// visibleDates should update visibleData if the data exists in the cache.
describe('calculateVisibleData', () => {
  const cache: MetricsData = {
    A: {
      M1: {
        name: 'M1',
        data: {
          '2021-01-01': 111,
          '2021-01-03': 113,
        },
      },
      M2: {
        name: 'M2',
        data: {
          '2021-01-01': 211,
          '2021-01-02': 212,
        },
      },
      M3: {
        name: 'M3',
        data: {
          '2021-01-02': 312,
          '2021-01-03': 313,
        },
      },
    },
  };
  const expectedVisible: MetricsData = {
    A: {
      M1: {
        name: 'M1',
        data: {
          '2021-01-03': 113,
        },
      },
      M2: {
        name: 'M2',
        data: {
          '2021-01-02': 212,
        },
      },
    },
  };

  // Tests that updateVisibleMetrics updates visibleData
  it('updateVisibleMetrics', () => {
    const action = actions.updateVisibleMetrics(['M1', 'M2']);
    const oldState = Object.assign({}, emptyState);
    oldState.visibleDates = ['2021-01-02', '2021-01-03'];
    oldState.cache = cache;
    const state = metricsReducer(oldState, action);

    expect(state.visibleMetrics).toEqual(action.payload);
    expect(state.visibleData).toMatchObject(expectedVisible);
  });

  // Tests that updateVisibleDates updates visibleData
  it('updateVisibleDates', () => {
    const action = actions.updateVisibleDates(['2021-01-02', '2021-01-03']);
    const oldState = Object.assign({}, emptyState);
    oldState.visibleMetrics = ['M1', 'M2'];
    oldState.cache = cache;
    const state = metricsReducer(oldState, action);

    expect(state.visibleDates).toEqual(action.payload);
    expect(state.visibleData).toMatchObject(expectedVisible);
  });
});

// Tests the fetchMetricsStart reducer
describe('fetchMetricsStart', () => {
  it('adds loading dates and metrics', () => {
    const action = actions.fetchMetricsStart({
      dates: ['2021-01-02', '2021-02-03'],
      metrics: ['M1', 'M2'],
    });
    const state = metricsReducer(undefined, action);

    expect(state.loadingDates).toEqual(action.payload.dates);
    expect(state.loadingMetrics).toEqual(action.payload.metrics);
  });

  // Tests that a subsequent fetch merges the new request with what's already
  // being loaded.
  it('merges loading dates and metrics', () => {
    const action = actions.fetchMetricsStart({
      dates: ['2021-02-03'],
      metrics: ['M2'],
    });
    const oldState = Object.assign({}, emptyState);
    oldState.loadingDates = ['2021-01-02'];
    oldState.loadingMetrics = ['M1'];
    const state = metricsReducer(oldState, action);

    expect(state.loadingDates).toEqual(['2021-01-02', '2021-02-03']);
    expect(state.loadingMetrics).toEqual(['M1', 'M2']);
  });
});

// Tests the fetchMetricsSuccess reducer, which handles merging the fetched
// data into the cache.
describe('fetchMetricsSuccess', () => {
  // Tests various scenarios given a empty response.
  describe('empty response', () => {
    const response: FetchMetricsResponse = {
      sections: {},
    };

    // Tests that an empty response adds loaded dates and metrics the lookup
    // of what's been cached.
    it('updates cached dates and metrics', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const state = metricsReducer(undefined, action);

      expect(state.cachedDates).toEqual(['2021-01-02']);
      expect(state.cachedMetrics).toEqual(['M1']);
    });

    // Tests that an empty response removes loaded dates and metrics from
    // what is currently being loaded.
    it('updates loading dates and metrics', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const oldState = Object.assign({}, emptyState);
      oldState.loadingDates = ['2021-01-02', '2021-02-03'];
      oldState.loadingMetrics = ['M1', 'M2'];
      const state = metricsReducer(oldState, action);

      expect(state.loadingDates).toEqual(['2021-02-03']);
      expect(state.loadingMetrics).toEqual(['M2']);
    });
  });

  // Tests various scenarios given a response with a single data point.
  describe('metrics response', () => {
    const response: FetchMetricsResponse = {
      sections: {
        A: [
          {
            name: 'M1',
            data: {
              '2021-01-02': 112,
            },
          },
        ],
      },
    };

    // Tests that the response is properly added into the cache
    it('updates the cache', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const state = metricsReducer(undefined, action);

      const expected: MetricsData = {
        A: {
          M1: {
            name: 'M1',
            data: {
              '2021-01-02': 112,
            },
          },
        },
      };
      expect(state.cache).toEqual(expected);
    });

    // Tests that a response properly updates visible data
    it('updates visible data', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const oldState = Object.assign({}, emptyState);
      oldState.visibleDates = ['2021-01-02'];
      oldState.visibleMetrics = ['M1'];
      const state = metricsReducer(oldState, action);

      const expected: MetricsData = {
        A: {
          M1: {
            name: 'M1',
            data: {
              '2021-01-02': 112,
            },
          },
        },
      };
      expect(state.visibleData).toEqual(expected);
    });

    // Tests that data from a response is merged into what's already in the
    // cache.
    it('merges with existing cache', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const oldState = Object.assign({}, emptyState);
      oldState.cache = {
        A: {
          M1: {
            name: 'M1',
            data: {
              '2021-01-03': 113,
            },
          },
        },
      };
      const state = metricsReducer(oldState, action);

      const expected: MetricsData = {
        A: {
          M1: {
            name: 'M1',
            data: {
              '2021-01-02': 112,
              '2021-01-03': 113,
            },
          },
        },
      };
      expect(state.cache).toEqual(expected);
    });
  });

  // Tests various scenarios given a response with subsections.
  describe('metrics with sub sections response', () => {
    const response: FetchMetricsResponse = {
      sections: {
        A: [
          {
            name: 'M1',
            sections: {
              A1: {
                '2021-01-02': 1112,
              },
            },
          },
        ],
      },
    };

    // Tests that the response is properly added into the cache
    it('updates the cache', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const state = metricsReducer(undefined, action);

      const expected: MetricsData = {
        A: {
          M1: {
            name: 'M1',
            sections: {
              A1: {
                '2021-01-02': 1112,
              },
            },
          },
        },
      };
      expect(state.cache).toEqual(expected);
    });

    // Tests that a response properly updates visible data
    it('updates visible data', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const oldState = Object.assign({}, emptyState);
      oldState.visibleDates = ['2021-01-02'];
      oldState.visibleMetrics = ['M1'];
      const state = metricsReducer(oldState, action);

      const expected: MetricsData = {
        A: {
          M1: {
            name: 'M1',
            sections: {
              A1: {
                '2021-01-02': 1112,
              },
            },
          },
        },
      };
      expect(state.visibleData).toEqual(expected);
    });

    // Tests that data from a response is merged into what's already in the
    // cache.
    it('merges with existing cache', () => {
      const action = actions.fetchMetricsSuccess({
        dates: ['2021-01-02'],
        metrics: ['M1'],
        response: response,
      });
      const oldState = Object.assign({}, emptyState);
      oldState.cache = {
        A: {
          M1: {
            name: 'M1',
            sections: {
              A1: {
                '2021-01-03': 1113,
              },
            },
          },
        },
      };
      const state = metricsReducer(oldState, action);

      const expected: MetricsData = {
        A: {
          M1: {
            name: 'M1',
            sections: {
              A1: {
                '2021-01-02': 1112,
                '2021-01-03': 1113,
              },
            },
          },
        },
      };
      expect(state.cache).toEqual(expected);
    });
  });
});
