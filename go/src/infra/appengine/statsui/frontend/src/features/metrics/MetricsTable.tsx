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
  DataSet,
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

/*
  compareSections compares two sections and returns the sort order between the
  two. It will either sort lexically by the section (and subsection) name, or it
  will sort by the value of the first metric for a given section, or if there
  are no metrics in the top-level section, by the first metric in the first
  subsection.
  @param a first section to compare.
  @param b second section to compare.
  @param orderBy the data to sort by. If it's the value 'section' then it
         sorts by section name. Otherwise, it is usually a date in the form of
         YYYY-MM-DD and sorts by value for the date.
  @param order Whether 'asc' for ascending or 'desc' for descending.
*/
function compareSections(
  a: Section,
  b: Section,
  orderBy: string,
  order: Order
): number {
  const invert = order === 'asc' ? 1 : -1;
  switch (orderBy) {
    case OrderBySection:
      if (a.name > b.name) {
        return 1 * invert;
      }
      if (a.name < b.name) {
        return -1 * invert;
      }
      return 0;
    case orderBy?.match(/^\d{4}-\d{2}-\d{2}$/)?.input:
      if (
        a.metrics.length > 0 &&
        orderBy in a.metrics[0].data &&
        b.metrics.length > 0 &&
        orderBy in b.metrics[0].data
      ) {
        // If we have a metric at the top level to show
        return (
          (a.metrics[0].data[orderBy].value -
            b.metrics[0].data[orderBy].value) *
          invert
        );
      } else if (
        a.metrics.length === 0 &&
        a.subSections.length > 0 &&
        b.metrics.length === 0 &&
        b.subSections.length > 0
      ) {
        // If the only metric we have is a subsection metric
        return compareSections(
          a.subSections[0],
          b.subSections[0],
          orderBy,
          order
        );
      }
      if (orderBy in a.metrics[0].data) {
        // Always put missing data at the bottom
        return -1;
      }
      // Always put missing data at the bottom
      return 1;
    default:
      return 0;
  }
  return 0;
}

function sortSections(sections: Section[], orderBy: string, order: Order) {
  // Sort the subsections first since we may use the subsection values to sort
  // the sections.
  sections.forEach((section) => {
    sortSections(section.subSections, orderBy, order);
  });
  sections.sort((a, b) => {
    let res = compareSections(a, b, orderBy, order);
    if (orderBy !== OrderBySection && res === 0) {
      // Make sure we have a stable sort
      res = compareSections(a, b, OrderBySection, order);
    }
    return res;
  });
}

interface Props {
  initialFilter?: string;
  initialOrderCol?: number;
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
  const [orderCol, setOrderCol] = React.useState<number>(
    props.initialOrderCol === undefined ? -1 : props.initialOrderCol
  );
  const [order, setOrder] = React.useState<Order>(
    props.initialOrder === undefined ? 'desc' : (props.initialOrder as Order)
  );

  // If using default orderCol, set it to the most recent date available
  if (orderCol === -1 && dates.length > 0) {
    setOrderCol(dates.length + 1);
  }

  const createSortHandler = (column: number) => {
    return () => {
      setOrderCol(column);
      // Reset the order to default if switching columns
      setOrder(
        orderCol === column ? (order === 'asc' ? 'desc' : 'asc') : 'desc'
      );
      return;
    };
  };

  let sections: Section[] = generateSections(dataSource, data, metrics);
  // orderCol is the column to order by. The UI has the section name as column
  // 0, the metric name as column 1, and the dates as column 2+. Thus
  // to get the date to use for the orderBy, we use orderCol - 2.
  const orderBy = orderCol === 0 ? OrderBySection : dates[orderCol - 2];
  sortSections(sections, orderBy, order);
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
    // Do another sort after filtering. This handles a very specific case
    // where you're only looking at subsection metrics and the filter changes
    // which subsection metric is used in the sorting.
    sortSections(sections, orderBy, order);
  }

  return (
    <TableContainer component={Paper} className={styles.metrics}>
      <MetricsTableParams orderCol={orderCol} order={order} filter={filter} />
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
                active={orderCol === 0}
                direction={orderCol === 0 ? order : 'desc'}
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
                  active={orderCol === i + 2}
                  direction={orderCol === i + 2 ? order : 'desc'}
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
