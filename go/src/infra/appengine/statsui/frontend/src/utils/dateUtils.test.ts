// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { calculateValidDates, Period } from './dateUtils';

test.each([
  // Day period
  ['2020-01-01', Period.Day, 0, 0, ['2020-01-01']],
  [new Date(2020, 0, 1), Period.Day, 0, 0, ['2020-01-01']],
  [
    new Date('2020-01-01T20:00:00.0000-09:00'),
    Period.Day,
    0,
    0,
    ['2020-01-01'],
  ],
  ['2020-01-01', Period.Day, 1, 0, ['2019-12-31', '2020-01-01']],
  ['2020-01-01', Period.Day, 0, 1, ['2020-01-01', '2020-01-02']],
  ['2019-12-31', Period.Day, 0, 1, ['2019-12-31', '2020-01-01']],

  // Week period
  ['2020-01-01', Period.Week, 0, 0, ['2019-12-30']],
  [new Date('2020-01-01'), Period.Week, 0, 0, ['2019-12-30']],
  ['2020-01-01', Period.Week, 1, 0, ['2019-12-23', '2019-12-30']],
  ['2020-01-01', Period.Week, 0, 1, ['2019-12-30', '2020-01-06']],
  ['2020-02-02', Period.Week, 0, 0, ['2020-01-27']],

  // Month period
  ['2020-01-15', Period.Month, 0, 0, ['2020-01-01']],
  [new Date('2020-01-15'), Period.Month, 0, 0, ['2020-01-01']],
  ['2020-01-15', Period.Month, 1, 0, ['2019-12-01', '2020-01-01']],
  ['2020-01-15', Period.Month, 0, 1, ['2020-01-01', '2020-02-01']],
])(
  '.calculateValidDates(%p, %p, %p, %p)',
  (date, period, before, after, expected) => {
    expect(calculateValidDates(date, period, before, after)).toStrictEqual(
      expected
    );
  }
);

test('do not include date', () => {
  expect(calculateValidDates(new Date(), Period.Day, 0, 0, false)).toHaveLength(
    0
  );
});

test('no future dates', () => {
  expect(calculateValidDates(new Date(), Period.Day, 0, 5)).toHaveLength(1);
});
