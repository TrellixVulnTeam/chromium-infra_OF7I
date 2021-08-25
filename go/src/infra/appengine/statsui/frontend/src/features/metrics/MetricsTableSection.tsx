// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useSelector } from 'react-redux';
import TableCell from '@material-ui/core/TableCell';
import TableRow from '@material-ui/core/TableRow';
import classnames from 'classnames';

import { selectCurrentSource } from '../dataSources/dataSourcesSlice';
import Icon from '@material-ui/core/Icon';
import { Section } from './MetricsTable';
import MetricsTableDataCell from './MetricsTableDataCell';
import styles from './MetricsTableSection.module.css';
import { Link } from '@material-ui/core';

interface Props {
  odd: boolean;
  rank?: number;
  indent?: boolean;
  dates: string[];
  section: Section;
  sectionLinkTemplate?: string;
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
  sectionLinkTemplate,
}: Props) => {
  const dataSource = useSelector(selectCurrentSource);

  let sectionName: React.ReactElement;
  if (section.name === '') {
    sectionName = <>-</>;
  } else {
    sectionName = (
      <>
        {section.name}
        {sectionLinkTemplate && (
          <Link
            href={sectionLinkTemplate.replaceAll(
              'SECTION_NAME',
              encodeURI(section.name)
            )}
            className={styles.link}
            data-testid="mt-row-section-name-link"
            target="_blank"
          >
            <Icon className={styles.linkIcon}>launch</Icon>
          </Link>
        )}
      </>
    );
  }

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
              {sectionName}
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
              {sectionName}
            </TableCell>
          )}
          <TableCell data-testid="mt-row-metric-name">{metric.name}</TableCell>
          {dates.map((date, i) => (
            <MetricsTableDataCell
              key={i}
              data={metric.data[date]}
              metric={dataSource.metricMap[metric.name]}
            />
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
