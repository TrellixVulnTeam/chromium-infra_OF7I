// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {
  format as dateFormat,
  formatDuration,
  intervalToDuration,
} from 'date-fns';
import { toTzDate } from './dateUtils';

export enum Unit {
  Date = 'date',
  Duration = 'duration',
  Number = 'number',
}

export const DateFormat = 'MM/dd/yy';

const min = 60;
const hour = 60 * min;

const numberFormat = new Intl.NumberFormat();

export function format(value: string | number, unit: Unit): string {
  switch (unit) {
    case Unit.Date: {
      if (typeof value === 'number') {
        return '';
      }
      return dateFormat(toTzDate(value), DateFormat);
    }
    case Unit.Duration: {
      if (typeof value === 'string') {
        return '';
      }
      const duration = intervalToDuration({
        start: 0,
        end: value * 1000,
      });
      if (value > 10 * min) {
        duration.seconds = 0;
      }
      if (value > 10 * hour) {
        duration.minutes = 0;
      }
      // 2 years 9 months 1 week 7 days 5 hours 9 minutes 30 seconds
      let ret = formatDuration(duration);
      ret = ret.replace(/ seconds?/, 's');
      ret = ret.replace(/ minutes?/, 'm');
      ret = ret.replace(/ hours?/, 'h');
      ret = ret.replace(/ weeks?/, 'w');
      ret = ret.replace(/ months?/, 'm');
      ret = ret.replace(/ years?/, 'y');
      return ret;
    }
    case Unit.Number: {
      if (typeof value === 'string') {
        return '';
      }
      return numberFormat.format(value);
    }
  }
}
