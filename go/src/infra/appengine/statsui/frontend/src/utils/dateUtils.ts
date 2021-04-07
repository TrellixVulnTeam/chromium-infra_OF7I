// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Enum specifying different time periods.
export enum Period {
  Undefined = '',
  Day = 'day',
  Week = 'week',
  Month = 'month',
}

const dayMs = 1000 * 60 * 60 * 24;

// toTzDate returns a date object for a YYYY-MM-DD string.
// Because Javascript interprets YYYY-MM-DD as UTC, simply using the date object
// may give the wrong date once UTC is converted to local TZ.
export function toTzDate(date: string): Date {
  const utcDate = new Date(date);
  return new Date(
    utcDate.getUTCFullYear(),
    utcDate.getUTCMonth(),
    utcDate.getUTCDate()
  );
}

// calculateValidDates generates an array of date strings given a starting date,
// a period, and how many periods before and after.  If the date given does not
// fall on a period boundary, it will use the earliest period boundary before
// the given date as the start point.
export function calculateValidDates(
  date: Date | string,
  period: Period,
  periodsBefore = 0,
  periodsAfter = 0,
  includeDate = true
): string[] {
  const now = new Date().getTime();

  let periodDate = new Date(date);
  if (date instanceof Date) {
    // Change date to UTC so there aren't TZ issues on the date math
    periodDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  }

  switch (period) {
    case Period.Week: {
      let daysAway = periodDate.getUTCDay() - 1; // First day is Monday
      if (daysAway < 0) {
        daysAway = 6;
      }
      periodDate.setTime(periodDate.getTime() - daysAway * dayMs);
      break;
    }
    case Period.Month: {
      periodDate.setUTCDate(1);
      break;
    }
  }

  const dates: string[] = [];
  for (let i = periodsBefore; i > 0; i--) {
    const date = addPeriodsToDate(periodDate, period, -i)
    if (date.getTime() < now) {
      dates.push(date.toISOString().slice(0, 10));
    }
  }
  if (includeDate) {
    dates.push(periodDate.toISOString().slice(0, 10));
  }
  for (let i = 1; i <= periodsAfter; i++) {
    const date = addPeriodsToDate(periodDate, period, i)
    if (date.getTime() < now) {
      dates.push(date.toISOString().slice(0, 10));
    }
  }

  return dates;
}

function addPeriodsToDate(date: Date, period: Period, num: number): Date {
  const ret = new Date();
  switch (period) {
    case Period.Day:
      ret.setTime(date.getTime() + num * dayMs);
      break;
    case Period.Week:
      ret.setTime(date.getTime() + num * dayMs * 7);
      break;
    case Period.Month:
      ret.setTime(date.getTime());
      ret.setUTCMonth(date.getUTCMonth() + num);
      break;
  }
  return ret
}
