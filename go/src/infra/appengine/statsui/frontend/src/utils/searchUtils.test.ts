// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { search } from './searchUtils';

test.each([
  ['test', 'test', true],
  ['test', 't', true],
  ['test', 'tt', false],

  ['test foo', 'test', true],
  ['test foo', 'foo', true],
  ['test foo', 'test foo', true],
  ['test foo', 'foo test', true],
  ['test foo', 'test foo you', false],
])('.search(%p, %p)', (text, query, expected) => {
  expect(search(text, query)).toBe(expected);
});
