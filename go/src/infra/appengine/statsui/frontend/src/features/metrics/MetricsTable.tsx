// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import classnames from 'classnames';
import Table from '@material-ui/core/Table';
import TableCell from '@material-ui/core/TableCell';
import TableContainer from '@material-ui/core/TableContainer';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Paper from '@material-ui/core/Paper';
import { useDispatch, useSelector } from 'react-redux';
import { IconButton, LinearProgress, TableSortLabel } from '@material-ui/core';
import ChevronLeftIcon from '@material-ui/icons/ChevronLeft';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';

import {
  decrementDates,
  incrementDates,
  MetricsData,
  selectMaxDate,
  selectShowLoading,
  selectVisibleData,
  selectVisibleDates,
  selectVisibleMetrics,
} from './metricsSlice';
import MetricsTableParams from './MetricsTableParams';
import MetricsTableToolbar from './MetricsTableToolbar';
import {
  DataSource,
  selectCurrentSource,
} from '../dataSources/dataSourcesSlice';
import { format, Unit } from '../../utils/formatUtils';
import { DataSet } from '../../api/metrics';

import styles from './MetricsTable.module.css';
import { search } from '../../utils/searchUtils';
import MetricsTableBody from './MetricsTableBody';

type Order = 'asc' | 'desc';
const OrderBySection = 'section';

export interface Section {
  name: string;
  metrics: SectionMetric[];
  subSections: Section[];
  rank?: number;
}

export interface SectionMetric {
  name: string;
  data: DataSet;
}

/*
  generateSections converts MetricsData into an array of Section objects, which
  is easier to render. It basically does hoists subsections, turning a data
  structure that looks like this:

  Section A:
    Metric 1: <data>
    Metric 2:
      Subsection A: <data>

  into:

  Section A:
    Metric 1: <data>
    Subsection A:
      Metric2: <data>

  The former is closer to the API. The latter is closer to how the UI renders
  it.
*/
function generateSections(
  source: DataSource,
  data: MetricsData,
  dates: string[],
  metrics: string[]
): Section[] {
  const ret: Section[] = [];
  Object.keys(data).forEach((sectionName) => {
    const section: Section = {
      name: sectionName,
      metrics: [],
      subSections: [],
    };
    const subSections: { [name: string]: SectionMetric[] } = {};
    metrics.forEach((metricName) => {
      if (metricName in data[sectionName]) {
        const metric = data[sectionName][metricName];
        if (metric.data !== undefined) {
          section.metrics.push({
            name: metric.name,
            data: metric.data,
          });
        } else if (metric.sections !== undefined) {
          Object.keys(metric.sections).forEach((subSectionName) => {
            if (metric.sections === undefined) return; // Needed for linter
            if (!(subSectionName in subSections)) {
              subSections[subSectionName] = [];
            }
            subSections[subSectionName].push({
              name: metric.name,
              data: metric.sections[subSectionName],
            });
          });
        }
      } else {
        // If there is no data, display a blank row
        const metric = { name: metricName, data: {} };
        if (source.metricMap[metricName].hasSubsections) {
          subSections[''] = [metric];
        } else {
          section.metrics.push({ name: metricName, data: {} });
        }
      }
    });
    Object.keys(subSections).forEach((subSectionName) => {
      section.subSections.push({
        name: subSectionName,
        metrics: subSections[subSectionName],
        subSections: [],
      });
    });
    ret.push(section);
  });
  return ret;
}

function compareSections(
  a: Section,
  b: Section,
  orderBy: string,
  order: Order
): number {
  const invert = order === 'asc' ? 1 : -1;
  switch (orderBy) {
    case '':
      return 0;
    case OrderBySection:
      if (a.name > b.name) {
        return 1 * invert;
      }
      if (a.name < b.name) {
        return -1 * invert;
      }
      return 0;
    default:
      if (a.metrics.length === 0 || b.metrics.length === 0) {
        // This should never happen
        return 0;
      }
      if (orderBy in a.metrics[0].data && orderBy in b.metrics[0].data) {
        return (
          (a.metrics[0].data[orderBy] - b.metrics[0].data[orderBy]) * invert
        );
      }
      if (orderBy in a.metrics[0].data) {
        // Always put missing data at the bottom
        return -1;
      }
      // Always put missing data at the bottom
      return 1;
  }
  return 0;
}

function sortSections(sections: Section[], orderBy: string, order: Order) {
  sections.sort((a, b) => {
    let res = compareSections(a, b, orderBy, order);
    if (orderBy !== OrderBySection && res === 0) {
      // Make sure we have a stable sort
      res = compareSections(a, b, OrderBySection, order);
    }
    return res;
  });
  sections.forEach((section) => {
    sortSections(section.subSections, orderBy, order);
  });
}

interface Props {
  initialFilter?: string;
  initialOrderBy?: number;
  initialOrder?: string;
}

/*
  MetricsTable component, which displays the metrics that are currently
  visible. Data to be rendered is read from the MetricState redux store.
*/
const MetricsTable: React.FunctionComponent<Props> = (props: Props) => {
  const dataSource = useSelector(selectCurrentSource);
  const data = useSelector(selectVisibleData);
  const dates = useSelector(selectVisibleDates);
  const metrics = useSelector(selectVisibleMetrics);
  const maxDate = useSelector(selectMaxDate);
  const showLoading = useSelector(selectShowLoading);

  const dispatch = useDispatch();

  const [filter, setFilter] = React.useState<string>(
    props.initialFilter === undefined ? '' : props.initialFilter
  );
  const [orderBy, setOrderBy] = React.useState<number>(
    props.initialOrderBy === undefined ? -1 : props.initialOrderBy
  );
  const [order, setOrder] = React.useState<Order>(
    props.initialOrder === undefined ? 'desc' : (props.initialOrder as Order)
  );

  // If using default orderBy, set it to the most recent date available
  if (orderBy === -1 && dates.length > 0) {
    setOrderBy(dates.length + 1);
  }

  const createSortHandler = (column: number) => {
    return () => {
      setOrderBy(column);
      // Reset the order to default if switching columns
      setOrder(
        orderBy === column ? (order === 'asc' ? 'desc' : 'asc') : 'desc'
      );
      return;
    };
  };

  let sections: Section[] = generateSections(dataSource, data, dates, metrics);
  sortSections(
    sections,
    orderBy === 0 ? OrderBySection : dates[orderBy - 2],
    order
  );
  // Set a rank number because filtering makes row number unreliable
  sections.forEach((section, i) => (section.rank = i + 1));

  if (filter !== '') {
    sections.forEach((section) => {
      // If the section name includes the filter string, show all subsections
      section.subSections = section.subSections.filter(
        (subSection) =>
          search(section.name, filter) ||
          // Hack to add a few more options when searching. Should probably
          // remove later and replace with a real filtering language.
          search(`sec:${section.name} sub:${subSection.name}`, filter)
      );
    });
    // Show the section if there are any subsections
    sections = sections.filter(
      (section) =>
        section.subSections.length > 0 || search(section.name, filter)
    );
  }

  return (
    <TableContainer component={Paper} className={styles.metrics}>
      <MetricsTableParams orderBy={orderBy} order={order} filter={filter} />
      <LinearProgress
        className={showLoading ? '' : styles.hidden}
        data-testid="metrics-table-loading"
      />
      <Table size="small" className={styles.table}>
        <TableHead>
          <MetricsTableToolbar
            initialFilter={props.initialFilter}
            onFilterChange={setFilter}
          />
          <TableRow>
            <TableCell className={styles.rank}>#</TableCell>
            <TableCell className={styles.section}>
              <TableSortLabel
                active={orderBy === 0}
                direction={orderBy === 0 ? order : 'desc'}
                onClick={createSortHandler(0)}
                data-testid="mt-sortby-section"
              >
                {dataSource.sectionName}
              </TableSortLabel>
            </TableCell>
            <TableCell className={styles.metric}>Metric</TableCell>
            {dates.map((date, i) => (
              <TableCell key={i} align="right" className={styles.data}>
                {/* Button to scroll dates to the right */}
                {i === 0 && (
                  <IconButton
                    size="small"
                    className={styles.dateNav}
                    onClick={() => dispatch(decrementDates())}
                  >
                    <ChevronLeftIcon />
                  </IconButton>
                )}
                {/* Label to sort by date */}
                <TableSortLabel
                  active={orderBy === i + 2}
                  direction={orderBy === i + 2 ? order : 'desc'}
                  onClick={createSortHandler(i + 2)}
                  data-testid="mt-sortby-date"
                >
                  {format(date, Unit.Date)}
                </TableSortLabel>
                {/* Button to scroll dates to the left */}
                {i === dates.length - 1 && (
                  <IconButton
                    size="small"
                    className={classnames(
                      styles.dateNav,
                      date < maxDate ? '' : styles.hidden
                    )}
                    onClick={() => dispatch(incrementDates())}
                  >
                    <ChevronRightIcon />
                  </IconButton>
                )}
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <MetricsTableBody
          dates={dates}
          sections={sections}
          colSpan={dates.length + 2}
          showLoading={showLoading}
          hasMetrics={metrics.length === 0}
        />
      </Table>
    </TableContainer>
  );
};

export default MetricsTable;
