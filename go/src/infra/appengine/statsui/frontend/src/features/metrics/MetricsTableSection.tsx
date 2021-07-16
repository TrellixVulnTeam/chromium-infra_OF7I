// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useSelector } from 'react-redux';
import TableCell from '@material-ui/core/TableCell';
import TableRow from '@material-ui/core/TableRow';
import classnames from 'classnames';

import { format } from '../../utils/formatUtils';
import { selectCurrentSource } from '../dataSources/dataSourcesSlice';
import { Section } from './MetricsTable';

import styles from './MetricsTableSection.module.css';

interface Props {
  odd: boolean;
  rank?: number;
  indent?: boolean;
  dates: string[];
  section: Section;
}

/*
  Renders a single section, which may include multiple rows if there are
  multiple metrics, or if the metric(s) have subsections.
*/
const MetricsTableSection: React.FunctionComponent<Props> = ({
  odd,
  rank,
  indent,
  dates,
  section,
}: Props) => {
  const dataSource = useSelector(selectCurrentSource);

  return (
    <>
      {
        // This block adds a row for the section if a section has no metrics,
        // but has subsections.
        section.metrics.length === 0 && (
          <TableRow
            key="header-empty"
            className={classnames(styles.section, odd ? styles.odd : '')}
            data-testid="mt-row"
          >
            <TableCell className={styles.rank} data-testid="mt-row-rank">
              {rank}
            </TableCell>
            <TableCell
              className={styles.name}
              data-testid="mt-row-section-name"
            >
              {section.name === '' ? '-' : section.name}
            </TableCell>
            <TableCell data-testid="mt-row-metric-name"></TableCell>
            {dates.map((date, i) => (
              <TableCell
                key={i}
                align="right"
                className={styles.data}
                data-testid="mt-row-data"
              >
                -
              </TableCell>
            ))}
          </TableRow>
        )
      }
      {section.metrics.map((metric, i) => (
        <TableRow
          key={metric.name}
          className={classnames(
            styles.section,
            odd ? styles.odd : '',
            indent ? styles.indent : ''
          )}
          data-testid="mt-row"
        >
          {i === 0 && (
            <TableCell
              className={styles.rank}
              rowSpan={section.metrics.length}
              data-testid="mt-row-rank"
            >
              {rank}
            </TableCell>
          )}
          {i === 0 && (
            <TableCell
              className={classnames(styles.name)}
              rowSpan={section.metrics.length}
              data-testid="mt-row-section-name"
            >
              {section.name === '' ? '-' : section.name}
            </TableCell>
          )}
          <TableCell data-testid="mt-row-metric-name">{metric.name}</TableCell>
          {dates.map((date, i) => (
            <TableCell
              key={i}
              align="right"
              className={styles.data}
              data-testid="mt-row-data"
            >
              {date in metric.data
                ? format(
                    metric.data[date],
                    dataSource.metricMap[metric.name].unit
                  )
                : '-'}
            </TableCell>
          ))}
        </TableRow>
      ))}
      {section.subSections.map((subSection) => (
        <MetricsTableSection
          key={subSection.name}
          odd={odd}
          indent={true}
          dates={dates}
          section={subSection}
        />
      ))}
    </>
  );
};

export default MetricsTableSection;
