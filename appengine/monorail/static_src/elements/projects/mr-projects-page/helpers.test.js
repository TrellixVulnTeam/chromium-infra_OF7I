// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {computeRoleByProjectName} from './helpers.js';

describe('computeRoleByProjectName', () => {
  it('handles empty project memberships', () => {
    assert.deepEqual(computeRoleByProjectName(undefined), {});
    assert.deepEqual(computeRoleByProjectName([]), {});
  });

  it('creates mapping', () => {
    const projectMembers = [
      {role: 'OWNER', name: 'projects/project-name/members/1234'},
      {role: 'COMMITTER', name: 'projects/test/members/1234'},
    ];
    assert.deepEqual(computeRoleByProjectName(projectMembers), {
      'projects/project-name': 'Owner',
      'projects/test': 'Committer',
    });
  });
});
