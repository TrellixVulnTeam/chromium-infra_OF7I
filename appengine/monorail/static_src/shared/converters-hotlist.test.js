// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import * as example from 'shared/test/constants-hotlist.js';
import * as hotlist from './converters-hotlist.js';

it('hotlistToRef', () => {
  assert.deepEqual(hotlist.hotlistToRef(example.HOTLIST), example.HOTLIST_REF);
});

it('hotlistRefToString', () => {
  const actual = hotlist.hotlistRefToString(example.HOTLIST_REF);
  assert.deepEqual(actual, example.HOTLIST_REF_STRING);
});

it('hotlistToRefString', () => {
  const actual = hotlist.hotlistToRefString(example.HOTLIST);
  assert.deepEqual(actual, example.HOTLIST_REF_STRING);
});
