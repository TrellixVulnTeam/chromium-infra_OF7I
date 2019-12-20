// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

const DEFAULT_DATE_LOCALE = 'en-US';

// Creating the datetime formatter costs ~1.5 ms, so when formatting
// multiple timestamps, it's more performant to reuse the formatter object.
// Export FORMATTER and SHORT_FORMATTER for testing. The return value differs
// based on time zone and browser, so we can't use static strings for testing.
// We can't stub out the method because it's native code and can't be modified.
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/DateTimeFormat/format#Avoid_comparing_formatted_date_values_to_static_values
export const FORMATTER = new Intl.DateTimeFormat(DEFAULT_DATE_LOCALE, {
  weekday: 'short',
  year: 'numeric',
  month: 'short',
  day: 'numeric',
  hour: 'numeric',
  minute: '2-digit',
  timeZoneName: 'short',
});

export const SHORT_FORMATTER = new Intl.DateTimeFormat(DEFAULT_DATE_LOCALE, {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
});

export const MS_PER_MINUTE = 60 * 1000;
export const MS_PER_HOUR = MS_PER_MINUTE * 60;
export const MS_PER_DAY = MS_PER_HOUR * 24;
export const MS_PER_MONTH = MS_PER_DAY * 30;

/**
 * Helper to determine if a Date was less than a month ago.
 * @param {Date} date The date to check.
 * @return {boolean} Whether the date was less than a
 *   month ago.
 */
function isLessThanAMonthAgo(date) {
  const now = new Date();
  const msDiff = Math.abs(Math.floor((now.getTime() - date.getTime())));
  return msDiff < MS_PER_MONTH;
}

/**
 * Displays timestamp in a standardized format to be re-used.
 * @param {Date} date
 * @return {string}
 */
export function standardTime(date) {
  if (!date) return;
  const absoluteTime = FORMATTER.format(date);

  let timeAgoBit = '';
  if (isLessThanAMonthAgo(date)) {
    // Only show relative time if the time is less than a
    // month ago because otherwise, it's not as useful.
    timeAgoBit = ` (${relativeTime(date)})`;
  }
  return `${absoluteTime}${timeAgoBit}`;
}

/**
 * Displays a timestamp in a format that's easy for a human to immediately
 * reason about, based on long ago the time was.
 * @param {Date} date native JavaScript Data Object.
 * @return {string} Human-readable string of the date.
 */
export function relativeTime(date) {
  if (!date) return;

  const now = new Date();
  let msDiff = now.getTime() - date.getTime();

  // Use different wording depending on whether the time is in the
  // future or past.
  const pastOrPresentSuffix = msDiff < 0 ? 'from now' : 'ago';
  msDiff = Math.abs(msDiff);

  if (msDiff < MS_PER_MINUTE) {
    // Less than a minute.
    return 'just now';
  } else if (msDiff < MS_PER_HOUR) {
    // Less than an hour.
    const minutes = Math.floor(msDiff / MS_PER_MINUTE);
    if (minutes === 1) {
      return `a minute ${pastOrPresentSuffix}`;
    }
    return `${minutes} minutes ${pastOrPresentSuffix}`;
  } else if (msDiff < MS_PER_DAY) {
    // Less than an day.
    const hours = Math.floor(msDiff / MS_PER_HOUR);
    if (hours === 1) {
      return `an hour ${pastOrPresentSuffix}`;
    }
    return `${hours} hours ${pastOrPresentSuffix}`;
  } else if (msDiff < MS_PER_MONTH) {
    // Less than a month.
    const days = Math.floor(msDiff / MS_PER_DAY);
    if (days === 1) {
      return `a day ${pastOrPresentSuffix}`;
    }
    return `${days} days ${pastOrPresentSuffix}`;
  }

  // A month or more ago. Better to show an exact date at this point.
  return SHORT_FORMATTER.format(date);
}
