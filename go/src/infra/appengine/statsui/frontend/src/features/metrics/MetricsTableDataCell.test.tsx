// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { render, screen } from '@testing-library/react';
import MetricsTableDataCell, { testables } from './MetricsTableDataCell';
import { Unit } from '../../utils/formatUtils';
import {
  MetricOption,
  MetricOptionColor,
  MetricOptionColorType,
} from '../dataSources/dataSourcesSlice';

describe('when rendering MetricsTableDataCell', () => {
  const metric: MetricOption = {
    name: 'M1',
    unit: Unit.Number,
    description: '',
  };
  it('should render empty state if no data', () => {
    render(
      <table>
        <tbody>
          <tr>
            <MetricsTableDataCell metric={metric} />
          </tr>
        </tbody>
      </table>
    );
    const cell = screen.getByTestId('mt-row-data');
    expect(cell).toHaveTextContent('-');
  });

  it('should render formatted data', () => {
    render(
      <table>
        <tbody>
          <tr>
            <MetricsTableDataCell data={{ value: 1000 }} metric={metric} />
          </tr>
        </tbody>
      </table>
    );
    const cell = screen.getByTestId('mt-row-data');
    expect(cell).toHaveTextContent('1,000');
  });

  it('should render absolute positive delta', () => {
    const colorMetric = Object.assign({}, metric);
    colorMetric.color = {
      type: MetricOptionColorType.DeltaAbsolute,
      breakpoints: [
        [10, '#fff'],
        [20, '#000'],
      ],
    };
    render(
      <table>
        <tbody>
          <tr>
            <MetricsTableDataCell
              data={{ value: 20, previous: { value: 10 } }}
              metric={colorMetric}
            />
          </tr>
        </tbody>
      </table>
    );
    const cell = screen.getByTestId('mt-row-data');
    expect(cell).toHaveStyle({ color: '#fff' });
  });
});

describe('when calculating colors', () => {
  it('should return color for negative percentage delta', () => {
    const color: MetricOptionColor = {
      type: MetricOptionColorType.DeltaPercentage,
      breakpoints: [
        [-0.5, '#fff'],
        [-1, '#000'],
      ],
    };
    expect(
      testables.calculateColor(color, { value: 10, previous: { value: 20 } })
    ).toStrictEqual('#fff');
  });
  it('should return color for empty value', () => {
    const color: MetricOptionColor = {
      type: MetricOptionColorType.DeltaAbsolute,
      breakpoints: [[10, '#fff']],
      emptyValue: 0,
    };
    expect(testables.calculateColor(color, { value: 10 })).toStrictEqual(
      '#fff'
    );
  });
});
