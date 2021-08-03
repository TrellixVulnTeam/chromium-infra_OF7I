// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { getByTestId, screen, fireEvent } from '@testing-library/react';
import MetricsTable from './MetricsTable';
import {
  renderWithRedux,
  initMetricsState,
  initDataSourceWithMetrics,
} from '../../utils/testUtils';
import { Unit } from '../../utils/formatUtils';

describe('when rendering MetricsTable', () => {
  it('should render empty state if no data', () => {
    renderWithRedux(<MetricsTable />, {});
    expect(screen.getByText('No data selected')).toBeInTheDocument();
  });

  it('should render a progress bar when loading', () => {
    renderWithRedux(<MetricsTable />, {
      metrics: initMetricsState({
        visibleDates: ['2021-01-02'],
      }),
    });
    expect(screen.getByTestId('metrics-table-loading')).not.toHaveClass(
      'hidden'
    );
  });
});

describe('when rendering MetricsTable with data', () => {
  it('should render metrics properly', () => {
    renderWithRedux(<MetricsTable />, {
      dataSources: initDataSourceWithMetrics('test-ds', '', {
        name: 'M1',
        unit: Unit.Number,
      }),
      metrics: initMetricsState({
        visibleDates: ['2021-01-02'],
        visibleMetrics: ['M1'],
        visibleData: {
          A: {
            M1: {
              name: 'M1',
              data: {
                '2021-01-02': { value: 112 },
              },
            },
          },
        },
      }),
    });
    const rows = screen.getAllByTestId('mt-row');
    expect(rows).toHaveLength(1);
    expect(getByTestId(rows[0], 'mt-row-section-name')).toHaveTextContent('A');
    expect(getByTestId(rows[0], 'mt-row-metric-name')).toHaveTextContent('M1');
    expect(getByTestId(rows[0], 'mt-row-data')).toHaveTextContent('112');
  });

  it('should render metrics with sections properly', () => {
    renderWithRedux(<MetricsTable />, {
      dataSources: initDataSourceWithMetrics('test-ds', 'test-section', {
        name: 'M1',
        unit: Unit.Number,
        hasSubsections: true,
      }),
      metrics: initMetricsState({
        visibleDates: ['2021-01-02'],
        visibleMetrics: ['M1'],
        visibleData: {
          A: {
            M1: {
              name: 'M1',
              sections: {
                S2: {
                  '2021-01-02': { value: 1212 },
                },
                S1: {
                  '2021-01-02': { value: 1112 },
                },
              },
            },
          },
        },
      }),
    });
    const rows = screen.getAllByTestId('mt-row');
    expect(rows).toHaveLength(3);
    // First row should just have the section name
    expect(getByTestId(rows[0], 'mt-row-section-name')).toHaveTextContent('A');
    expect(getByTestId(rows[0], 'mt-row-metric-name')).toBeEmptyDOMElement();
    expect(getByTestId(rows[0], 'mt-row-data')).toHaveTextContent('-');

    // Second row should have the first metric section
    expect(getByTestId(rows[1], 'mt-row-section-name')).toHaveTextContent('S2');
    expect(getByTestId(rows[1], 'mt-row-metric-name')).toHaveTextContent('M1');
    expect(getByTestId(rows[1], 'mt-row-data')).toHaveTextContent('1,212');

    // Third row should have the second metric section
    expect(getByTestId(rows[2], 'mt-row-section-name')).toHaveTextContent('S1');
    expect(getByTestId(rows[2], 'mt-row-metric-name')).toHaveTextContent('M1');
    expect(getByTestId(rows[2], 'mt-row-data')).toHaveTextContent('1,112');
  });
});

describe('when sorting MetricsTable data', () => {
  const state = {
    dataSources: initDataSourceWithMetrics('test-ds', '', {
      name: 'M1',
      unit: Unit.Number,
    }),
    metrics: initMetricsState({
      visibleDates: ['2021-01-02'],
      visibleMetrics: ['M1'],
      visibleData: {
        A: {
          M1: {
            name: 'M1',
            data: {
              '2021-01-02': { value: 1 },
            },
          },
        },
        B: {
          M1: {
            name: 'M1',
            data: {
              '2021-01-02': { value: 3 },
            },
          },
        },
        C: {
          M1: {
            name: 'M1',
            data: {
              '2021-01-02': { value: 2 },
            },
          },
        },
      },
    }),
  };

  it('default sort should be descending', () => {
    renderWithRedux(<MetricsTable />, state);
    const rows = screen.getAllByTestId('mt-row');
    expect(rows).toHaveLength(3);
    expect(getByTestId(rows[0], 'mt-row-section-name')).toHaveTextContent('B');
    expect(getByTestId(rows[1], 'mt-row-section-name')).toHaveTextContent('C');
    expect(getByTestId(rows[2], 'mt-row-section-name')).toHaveTextContent('A');
  });

  it('changing the date sort should switch to ascending', () => {
    renderWithRedux(<MetricsTable />, state);
    fireEvent.click(screen.getByTestId('mt-sortby-date'));
    const rows = screen.getAllByTestId('mt-row');
    expect(rows).toHaveLength(3);
    expect(getByTestId(rows[0], 'mt-row-section-name')).toHaveTextContent('A');
    expect(getByTestId(rows[1], 'mt-row-section-name')).toHaveTextContent('C');
    expect(getByTestId(rows[2], 'mt-row-section-name')).toHaveTextContent('B');
  });

  it('changing to section sort should sort by section', () => {
    renderWithRedux(<MetricsTable />, state);
    fireEvent.click(screen.getByTestId('mt-sortby-section'));
    const rows = screen.getAllByTestId('mt-row');
    expect(rows).toHaveLength(3);
    expect(getByTestId(rows[0], 'mt-row-section-name')).toHaveTextContent('C');
    expect(getByTestId(rows[1], 'mt-row-section-name')).toHaveTextContent('B');
    expect(getByTestId(rows[2], 'mt-row-section-name')).toHaveTextContent('A');
  });
});

describe('when filtering MetricsTable data', () => {
  it('should show only the filtered section', () => {
    renderWithRedux(<MetricsTable initialFilter="A" />, {
      dataSources: initDataSourceWithMetrics('test-ds', '', {
        name: 'M1',
        unit: Unit.Number,
      }),
      metrics: initMetricsState({
        visibleDates: ['2021-01-02'],
        visibleMetrics: ['M1'],
        visibleData: {
          A: {
            M1: {
              name: 'M1',
              data: {
                '2021-01-02': { value: 2 },
              },
            },
          },
          B: {
            M1: {
              name: 'M1',
              data: {
                '2021-01-02': { value: 1 },
              },
            },
          },
        },
      }),
    });
    const rows = screen.getAllByTestId('mt-row');
    expect(rows).toHaveLength(1);
    expect(getByTestId(rows[0], 'mt-row-section-name')).toHaveTextContent('A');
  });

  it('should show the filtered subsection', () => {
    renderWithRedux(<MetricsTable initialFilter="S2" />, {
      dataSources: initDataSourceWithMetrics('test-ds', 'test-section', {
        name: 'M1',
        unit: Unit.Number,
        hasSubsections: true,
      }),
      metrics: initMetricsState({
        visibleDates: ['2021-01-02'],
        visibleMetrics: ['M1'],
        visibleData: {
          A: {
            M1: {
              name: 'M1',
              sections: {
                S1: {
                  '2021-01-02': { value: 1 },
                },
              },
            },
          },
          B: {
            M1: {
              name: 'M1',
              sections: {
                S2: {
                  '2021-01-02': { value: 2 },
                },
              },
            },
          },
        },
      }),
    });
    const rows = screen.getAllByTestId('mt-row');
    expect(rows).toHaveLength(2);
    // First row should just have the section name
    expect(getByTestId(rows[0], 'mt-row-section-name')).toHaveTextContent('B');
    // Second row should have the sub section
    expect(getByTestId(rows[1], 'mt-row-section-name')).toHaveTextContent('S2');
  });
});
