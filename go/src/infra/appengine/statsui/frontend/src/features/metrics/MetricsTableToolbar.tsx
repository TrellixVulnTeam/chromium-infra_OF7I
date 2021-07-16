// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useEffect } from 'react';
import TableCell from '@material-ui/core/TableCell';
import TableRow from '@material-ui/core/TableRow';
import {
  Checkbox,
  FormControl,
  InputLabel,
  ListItemText,
  MenuItem,
  Select,
  TextField,
  Tooltip,
} from '@material-ui/core';
import {
  KeyboardDatePicker,
  MuiPickersUtilsProvider,
} from '@material-ui/pickers';
import DateFnsUtils from '@date-io/date-fns';
import { useDispatch, useSelector } from 'react-redux';

import {
  selectCurrentPeriod,
  selectNumPeriods,
  selectVisibleDates,
  selectVisibleMetrics,
  setDates,
  setMetrics,
  setNumPeriods,
  setPeriod,
} from './metricsSlice';
import { selectCurrentSource } from '../dataSources/dataSourcesSlice';
import {
  updatePreferencesDataSourceMetrics,
  updatePreferencesNumPeriods,
} from '../preferences/preferencesSlice';
import { DateFormat } from '../../utils/formatUtils';
import { Period, toTzDate } from '../../utils/dateUtils';

import styles from './MetricsTableToolbar.module.css';

interface Props {
  initialFilter?: string;
  onFilterChange: (filter: string) => void;
}

/*
  Top toolbar in the MetricsTable, holding a variety of controls.
*/
const MetricsTableToolbar: React.FunctionComponent<Props> = (props: Props) => {
  const dataSource = useSelector(selectCurrentSource);
  const dates = useSelector(selectVisibleDates);
  const metrics = useSelector(selectVisibleMetrics);
  const numPeriods = useSelector(selectNumPeriods);
  const period = useSelector(selectCurrentPeriod);

  const dispatch = useDispatch();

  // The toolbar has it's own filter state to store the current value of the
  // text field. Any changes are debounced before being sent to the filter
  // change handler.
  const [filter, setFilter] = React.useState<string>(props.initialFilter || '');
  // Implements a basic 500ms debounce. This useEffect is scoped to changes to
  // the filter state only, and returns a clean function to clear the timer.
  useEffect(() => {
    const timer = setTimeout(() => {
      props.onFilterChange(filter);
    }, 500);
    return () => clearTimeout(timer);
  }, [filter]);

  // Choose a date to use for the picker.  Need to use tzDate as just passing
  // 2000-01-01 to Date create a date in GMT, which comes out as 1999-12-31
  // in US timezones.
  const pickerDate =
    dates.length > 0 ? toTzDate(dates[dates.length - 1]) : new Date();

  const handleMetricsChange = (e: React.ChangeEvent<{ value: unknown }>) => {
    const metrics = e.target.value as string[];
    dispatch(setMetrics(metrics));
    dispatch(
      updatePreferencesDataSourceMetrics({
        dataSource: dataSource.name,
        metrics: metrics,
      })
    );
  };

  const handleDateChange = (date: Date | null) => {
    // Because we're using keyboard input, dates returned can be invalid dates
    if (date !== null && !Number.isNaN(date.getTime())) {
      dispatch(setDates(date));
    }
  };

  const handlePeriodChange = (e: React.ChangeEvent<{ value: unknown }>) => {
    const period = e.target.value as Period;
    dispatch(setPeriod(period));
  };

  const handleNumPeriodsChange = (e: React.ChangeEvent<{ value: unknown }>) => {
    const numPeriods = e.target.value as number;
    dispatch(setNumPeriods(numPeriods));
    dispatch(updatePreferencesNumPeriods(numPeriods));
  };

  return (
    <MuiPickersUtilsProvider utils={DateFnsUtils}>
      <TableRow className={styles.toolbar}>
        <TableCell></TableCell>
        <TableCell>
          <TextField
            label="Filter"
            size="small"
            className={styles.section}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            InputProps={{ inputProps: { 'data-testid': 'mt-toolbar-filter' } }}
          />
        </TableCell>
        <TableCell colSpan={2}>
          <FormControl size="small" className={styles.metrics}>
            <InputLabel>Metrics</InputLabel>
            <Select
              multiple
              value={metrics}
              onChange={handleMetricsChange}
              renderValue={(selected) => (selected as string[]).join(', ')}
            >
              {dataSource.metrics.map((metric) => (
                <MenuItem key={metric.name} value={metric.name}>
                  <Checkbox checked={metrics.indexOf(metric.name) > -1} />
                  <Tooltip
                    title={metric.description}
                    placement="right"
                    arrow
                    enterDelay={500}
                  >
                    <ListItemText primary={metric.name} />
                  </Tooltip>
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </TableCell>
        <TableCell colSpan={dates.length - 1} align="right">
          <KeyboardDatePicker
            disableToolbar
            variant="inline"
            size="small"
            format={DateFormat}
            label="Date"
            autoOk={true}
            disableFuture={true}
            value={pickerDate}
            onChange={handleDateChange}
            className={styles.date}
          />
          <FormControl size="small" className={styles.period}>
            <InputLabel>Cols</InputLabel>
            <Select value={numPeriods} onChange={handleNumPeriodsChange}>
              {Array(5)
                .fill(0)
                .map((_, i) => (
                  <MenuItem key={i} value={4 + i}>
                    {4 + i}
                  </MenuItem>
                ))}
            </Select>
          </FormControl>
          <FormControl size="small" className={styles.period}>
            <InputLabel>Period</InputLabel>
            <Select value={period} onChange={handlePeriodChange}>
              {dataSource.periods.map((period) => (
                <MenuItem key={period.period} value={period.period}>
                  {period.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </TableCell>
      </TableRow>
    </MuiPickersUtilsProvider>
  );
};

export default MetricsTableToolbar;
