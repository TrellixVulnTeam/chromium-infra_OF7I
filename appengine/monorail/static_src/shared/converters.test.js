// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {pathsToFieldMask} from './converters.js';

describe('pathsToFieldMask', () => {
  it('converts an array of strings to a FieldMask', () => {
    assert.equal(pathsToFieldMask(['foo', 'barQux', 'qaz']), 'foo,barQux,qaz');
  });
});
