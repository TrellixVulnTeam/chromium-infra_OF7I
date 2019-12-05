// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import * as hotlist from './converters-hotlist.js';

export const hotlistExample = {
  ownerRef: {
    userId: 12345678,
    displayName: 'example@example.com',
  },
  name: 'Hotlist-Name',
};
export const hotlistRefExample = {
  owner: {
    userId: 12345678,
    displayName: 'example@example.com',
  },
  name: 'Hotlist-Name',
};
export const hotlistRefStringExample = '12345678:Hotlist-Name';

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
