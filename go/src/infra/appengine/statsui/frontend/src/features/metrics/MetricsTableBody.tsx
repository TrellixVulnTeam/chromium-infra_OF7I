// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useSelector } from 'react-redux';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableRow from '@material-ui/core/TableRow';

import MetricsTableSection from './MetricsTableSection';
import { selectCurrentSource } from '../dataSources/dataSourcesSlice';

import styles from './MetricsTableBody.module.css';
import { Section } from './MetricsTable';

interface Props {
  dates: string[];
  sections: Section[];
  colSpan: number;
  showLoading: boolean;
  hasMetrics: boolean;
}

/*
  MetricsTableBody component, which renders the body of the metrics table.
*/
const MetricsTableBody: React.FunctionComponent<Props> = (props: Props) => {
  const dataSource = useSelector(selectCurrentSource);

  if (props.sections.length === 0) {
    return (
      <TableBody>
        <TableRow>
          <TableCell
            className={styles.empty}
            colSpan={props.colSpan}
            align="center"
          >
            {props.showLoading
              ? 'Loading...'
              : 'No data ' + (props.hasMetrics ? 'selected' : 'available')}
          </TableCell>
        </TableRow>
      </TableBody>
    );
  }

  return (
    <TableBody>
      {props.sections.map((section, i) => (
        <MetricsTableSection
          odd={i % 2 === 1}
          key={section.name}
          rank={section.rank}
          dates={props.dates}
          section={section}
          sectionLinkTemplate={dataSource.sectionLinkTemplate}
        />
      ))}
    </TableBody>
  );
};

export default MetricsTableBody;
