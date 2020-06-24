// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {ResourceNameError, pathsToFieldMask, extractUserId,
  extractProjectDisplayName, extractProjectFromProjectMember,
  projectAndUserToStarName, projectMemberToProjectName} from './converters.js';

describe('pathsToFieldMask', () => {
  it('converts an array of strings to a FieldMask', () => {
    assert.equal(pathsToFieldMask(['foo', 'barQux', 'qaz']), 'foo,barQux,qaz');
  });
});

describe('extractUserId', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(() => extractUserId('projects/1234'),
        ResourceNameError);
    assert.throws(() => extractUserId('users/notAnId'),
        ResourceNameError);
    assert.throws(() => extractUserId('user/1234'),
        ResourceNameError);
  });

  it('extracts user ID', () => {
    assert.equal(extractUserId('users/1234'), '1234');
  });
});

describe('extractProjectDisplayName', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(() => extractProjectDisplayName('users/1234'),
        ResourceNameError);
    assert.throws(() => extractProjectDisplayName('projects/(what)'),
        ResourceNameError);
    assert.throws(() => extractProjectDisplayName('project/test'),
        ResourceNameError);
    assert.throws(() => extractProjectDisplayName('projects/-test-'),
        ResourceNameError);
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

describe('extractProjectFromProjectMember', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(
        () => extractProjectFromProjectMember(
            'projects/monorail/members/fakeName'),
        ResourceNameError);
    assert.throws(
        () => extractProjectFromProjectMember(
            'projects/-invalid-project-/members/1234'),
        ResourceNameError);
    assert.throws(
        () => extractProjectFromProjectMember(
            'projects/monorail/member/1234'),
        ResourceNameError);
  });

  it('extracts project display name', () => {
    assert.equal(extractProjectFromProjectMember(
        'projects/1234/members/1234'), '1234');
    assert.equal(extractProjectFromProjectMember(
        'projects/monorail/members/1234'), 'monorail');
    assert.equal(extractProjectFromProjectMember(
        'projects/test-project/members/1234'), 'test-project');
    assert.equal(extractProjectFromProjectMember(
        'projects/what-is-love2/members/1234'), 'what-is-love2');
  });
});

describe('projectAndUserToStarName', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(
        () => projectAndUserToStarName('users/1234', 'projects/monorail'),
        ResourceNameError);
  });

  it('generates project star resource name', () => {
    assert.equal(projectAndUserToStarName('projects/monorail', 'users/1234'),
        'users/1234/projectStars/monorail');
  });
});

describe('projectMemberToProjectName', () => {
  it('throws error on improperly formatted resource name', () => {
    assert.throws(
        () => projectMemberToProjectName(
            'projects/monorail/members/fakeName'),
        ResourceNameError);
  });

  it('creates project resource name', () => {
    assert.equal(projectMemberToProjectName(
        'projects/1234/members/1234'), 'projects/1234');
    assert.equal(projectMemberToProjectName(
        'projects/monorail/members/1234'), 'projects/monorail');
    assert.equal(projectMemberToProjectName(
        'projects/test-project/members/1234'), 'projects/test-project');
    assert.equal(projectMemberToProjectName(
        'projects/what-is-love2/members/1234'), 'projects/what-is-love2');
  });
});
