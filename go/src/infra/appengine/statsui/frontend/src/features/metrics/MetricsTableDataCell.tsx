// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import TableCell from '@material-ui/core/TableCell';

import { format } from '../../utils/formatUtils';
import styles from './MetricsTableDataCell.module.css';
import { DataPoint } from './metricsSlice';
import {
  MetricOption,
  MetricOptionColor,
  MetricOptionColorType,
} from '../dataSources/dataSourcesSlice';

interface Props {
  data?: DataPoint;
  metric: MetricOption;
}

// Determines if, given the metric and the data point, whether it should be
// rendered with a different color.
function calculateColor(
  color?: MetricOptionColor,
  data?: DataPoint
): string | undefined {
  if (color === undefined) {
    return;
  }
  const currValue = data?.value || color.emptyValue;
  const prevValue = data?.previous?.value || color.emptyValue;
  if (currValue === undefined || prevValue === undefined) {
    return;
  }
  let delta = currValue - prevValue;
  if (color.type === MetricOptionColorType.DeltaPercentage) {
    delta = delta / prevValue;
  }
  const breakpoint = color.breakpoints.find((breakpoint) => {
    return (
      (breakpoint[0] < 0 && delta <= breakpoint[0]) ||
      (breakpoint[0] > 0 && delta >= breakpoint[0])
    );
  });
  if (breakpoint) {
    return breakpoint[1];
  }
  return;
}

export const testables = {
  calculateColor: calculateColor,
};

/*
  Renders a data cell
*/
const MetricsTableDataCell: React.FunctionComponent<Props> = ({
  metric,
  data,
}: Props) => {
  const style: React.CSSProperties = {};
  const color = calculateColor(metric.color, data);
  if (color !== undefined) {
    style.color = color;
  }
  return (
    <TableCell
      align="right"
      className={styles.data}
      data-testid="mt-row-data"
      style={style}
    >
      {data === undefined ? '-' : format(data.value, metric.unit)}
    </TableCell>
  );
};

export default MetricsTableDataCell;
