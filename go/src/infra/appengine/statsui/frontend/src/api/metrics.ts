// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { Period } from '../utils/dateUtils';

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

// Exporting this for testing.  Tests can then replace prpcClient.call with a
// mock implementation.
// prpcClient sets credentials: omit, which breaks internal local prpc calls.
// See: https://source.chromium.org/chromium/infra/infra/+/main:crdx/packages/prpc-client/src/prpc-client.ts;l=172;
// Switching back to using fetch until prpc-client is updated.
export const prpcClient = {
  call: async function <Type>(
    service: string,
    method: string,
    message: unknown
  ): Promise<Type> {
    const url = `/prpc/${service}/${method}`;
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: JSON.stringify(message),
    });
    const text = await response.text();
    if (text.startsWith(")]}'")) {
      return JSON.parse(text.substr(4));
    } else {
      return JSON.parse(text);
    }
  },
};

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
