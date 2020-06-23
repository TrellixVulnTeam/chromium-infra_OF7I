// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {pathsToFieldMask, extractUserId, extractProjectDisplayName,
  projectAndUserToStarName} from './converters.js';

describe('pathsToFieldMask', () => {
  it('converts an array of strings to a FieldMask', () => {
    assert.equal(pathsToFieldMask(['foo', 'barQux', 'qaz']), 'foo,barQux,qaz');
  });
});

describe('extractUserId', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(() => extractUserId('projects/1234'),
        'Improperly formatted resource name.');
    assert.throws(() => extractUserId('users/notAnId'),
        'Improperly formatted resource name.');
    assert.throws(() => extractUserId('user/1234'),
        'Improperly formatted resource name.');
  });

  it('extracts user ID', () => {
    assert.equal(extractUserId('users/1234'), '1234');
  });
});

describe('extractProjectDisplayName', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(() => extractProjectDisplayName('users/1234'),
        'Improperly formatted resource name.');
    assert.throws(() => extractProjectDisplayName('projects/(what)'),
        'Improperly formatted resource name.');
    assert.throws(() => extractProjectDisplayName('project/test'),
        'Improperly formatted resource name.');
    assert.throws(() => extractProjectDisplayName('projects/-test-'),
        'Improperly formatted resource name.');
  });

  it('extracts project display name', () => {
    assert.equal(extractProjectDisplayName('projects/1234'), '1234');
    assert.equal(extractProjectDisplayName('projects/monorail'), 'monorail');
    assert.equal(extractProjectDisplayName('projects/test-project'),
        'test-project');
    assert.equal(extractProjectDisplayName('projects/what-is-love2'),
        'what-is-love2');
  });
});

describe('projectAndUserToStarName', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(
        () => projectAndUserToStarName('users/1234', 'projects/monorail'),
        'Improperly formatted resource name.');
  });

  it('generates project star resource name', () => {
    assert.equal(projectAndUserToStarName('projects/monorail', 'users/1234'),
        'users/1234/projectStars/monorail');
  });
});
