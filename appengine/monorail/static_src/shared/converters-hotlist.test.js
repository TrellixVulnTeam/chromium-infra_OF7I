// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {hotlistExample, hotlistRefExample,
  hotlistRefStringExample} from 'shared/test/hotlist-constants.js';
import * as hotlist from './converters-hotlist.js';

it('hotlistToRef', () => {
  assert.deepEqual(hotlist.hotlistToRef(hotlistExample), hotlistRefExample);
});

it('hotlistRefToString', () => {
  const actual = hotlist.hotlistRefToString(hotlistRefExample);
  assert.deepEqual(actual, hotlistRefStringExample);
});

it('hotlistToRefString', () => {
  const actual = hotlist.hotlistToRefString(hotlistExample);
  assert.deepEqual(actual, hotlistRefStringExample);
});
