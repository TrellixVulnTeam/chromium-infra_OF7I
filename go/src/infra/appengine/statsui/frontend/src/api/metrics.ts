// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { Period } from '../utils/dateUtils';
import { PrpcClient } from '@chopsui/prpc-client';

export type FetchDataSet = { [date: string]: number };

export interface FetchMetric {
  name: string;
  data?: FetchDataSet;
  sections?: { [section: string]: FetchDataSet };
}

export interface FetchMetricsResponse {
  sections: { [section: string]: FetchMetric[] };
}

interface pbFetchMetricsRequest {
  // eslint-disable-next-line camelcase
  data_source: string;
  period: number;
  dates: string[];
  metrics: string[];
}

interface pbFetchMetricsResponse {
  sections: pbFetchMetricsSection[];
}

interface pbFetchMetricsSection {
  name: string;
  metrics: pbFetchMetricsMetric[];
}

interface pbFetchMetricsMetric {
  name: string;
  data?: pbFetchMetricsDataSet;
  sections?: { [section: string]: pbFetchMetricsDataSet };
}

interface pbFetchMetricsDataSet {
  data: FetchDataSet;
}

function pbPeriod(p: Period): number {
  switch (p) {
    case Period.Week:
      return 1;
    case Period.Day:
      return 2;
    case Period.Month:
      return 3;
  }
  return 0;
}

// Exporting this for testing.  Tests can then replace prpcClient with a mock
// implementation.
export const prpcClient = new PrpcClient({
  insecure: location.protocol === 'http:',
});

export async function fetchMetrics(
  dataSource: string,
  period: Period,
  dates: string[],
  metrics: string[]
): Promise<FetchMetricsResponse> {
  if (dataSource === '' || dates.length === 0 || metrics.length === 0) {
    // Nothing to fetch.  Return an empty response
    return { sections: {} };
  }
  const req: pbFetchMetricsRequest = {
    data_source: dataSource,
    period: pbPeriod(period),
    dates: dates,
    metrics: metrics,
  };
  const resp: pbFetchMetricsResponse = await prpcClient.call(
    'statsui.Stats',
    'FetchMetrics',
    req
  );
  const ret: FetchMetricsResponse = { sections: {} };
  resp.sections.forEach((section) => {
    ret.sections[section.name] = section.metrics.map((metric) => {
      const m: FetchMetric = { name: metric.name };
      if (metric.data != undefined) {
        m.data = metric.data.data;
      }
      if (metric.sections != undefined) {
        m.sections = {};
        Object.keys(metric.sections).forEach((sectionName) => {
          if (m.sections === undefined || metric.sections === undefined) return; // Needed for lint
          m.sections[sectionName] = metric.sections[sectionName].data;
        });
      }
      return m;
    });
  });
  return ret;
}
