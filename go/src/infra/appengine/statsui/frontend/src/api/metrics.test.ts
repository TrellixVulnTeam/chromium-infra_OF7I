// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { prpcClient, fetchMetrics, FetchMetricsResponse } from './metrics';
import { Period } from '../utils/dateUtils';

// Tests the fetchMetrics function
describe('fetchMetrics', () => {
  // Tests that a response with metrics gets parsed properly
  it('returns metrics', async () => {
    const mockCall = jest.spyOn(prpcClient, 'call').mockResolvedValue({
      sections: [
        {
          name: 'A',
          metrics: [
            {
              name: 'M1',
              data: {
                data: {
                  '2021-01-02': 112,
                },
              },
            },
          ],
        },
      ],
    });
    const expected: FetchMetricsResponse = {
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

    const resp = await fetchMetrics('ds', Period.Day, ['2021-01-02'], ['M1']);

    expect(mockCall.mock.calls.length).toBe(1);
    expect(mockCall.mock.calls[0].length).toBe(3);
    expect(mockCall.mock.calls[0][0]).toBe('statsui.Stats');
    expect(mockCall.mock.calls[0][1]).toBe('FetchMetrics');
    expect(mockCall.mock.calls[0][2]).toEqual({
      data_source: 'ds',
      period: 2,
      dates: ['2021-01-02'],
      metrics: ['M1'],
    });

    expect(resp).toEqual(expected);
  });

  // Tests that a response with subsections gets parsed properly
  it('returns subsection metrics', async () => {
    const mockCall = jest.spyOn(prpcClient, 'call').mockResolvedValue({
      sections: [
        {
          name: 'A',
          metrics: [
            {
              name: 'M2',
              sections: {
                A1: {
                  data: {
                    '2021-01-02': 1112,
                  },
                },
              },
            },
          ],
        },
      ],
    });
    const expected: FetchMetricsResponse = {
      sections: {
        A: [
          {
            name: 'M2',
            sections: {
              A1: {
                '2021-01-02': 1112,
              },
            },
          },
        ],
      },
    };

    const resp = await fetchMetrics('ds', Period.Day, ['2021-01-02'], ['M2']);

    expect(mockCall.mock.calls.length).toBe(1);
    expect(mockCall.mock.calls[0].length).toBe(3);
    expect(mockCall.mock.calls[0][2]).toEqual({
      data_source: 'ds',
      period: 2,
      dates: ['2021-01-02'],
      metrics: ['M2'],
    });

    expect(resp).toEqual(expected);
  });
});
