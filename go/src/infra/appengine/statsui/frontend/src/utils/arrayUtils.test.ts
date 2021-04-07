// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { merge, removeFrom } from './arrayUtils';

test.each([
  [
    [
      [1, 2],
      [2, 3],
    ],
    [1, 2, 3],
  ],
])('.merge(%p, %p)', (a, expected) => {
  expect(merge(...a)).toStrictEqual(expected);
});

test.each([
  [[1, 2], [2], [1]],
  [
    [1, 2, 3, 4],
    [2, 4],
    [1, 3],
  ],
])('.removeFrom(%p, %p)', (a, b, expected) => {
  expect(removeFrom(a, b)).toStrictEqual(expected);
});
