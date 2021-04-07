// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { format, Unit } from './formatUtils';

test.each([
  // Date
  ['2020-01-02', Unit.Date, '01/02/20'],

  // Duration
  [1, Unit.Duration, '1s'],
  [30, Unit.Duration, '30s'],
  [60, Unit.Duration, '1m'],
  [90, Unit.Duration, '1m 30s'],
  [330, Unit.Duration, '5m 30s'],
  [600, Unit.Duration, '10m'],
  // Truncate the seconds if we're beyond 10m
  [630, Unit.Duration, '10m'],
  [3600, Unit.Duration, '1h'],
  [4200, Unit.Duration, '1h 10m'],
  [4230, Unit.Duration, '1h 10m'],

  // Number
  [1, Unit.Number, '1'],
  [1.5, Unit.Number, '1.5'],
  [3.14159, Unit.Number, '3.142'],
  [150, Unit.Number, '150'],
  [1050, Unit.Number, '1,050'],
])('.format(%p, %p)', (value, unit, expected) => {
  expect(format(value, unit)).toBe(expected);
});
