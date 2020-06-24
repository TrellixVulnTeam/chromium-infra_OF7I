// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {projectMemberToProjectName} from 'shared/converters.js';

// TODO(crbug.com/monorail/7910): Dedupe this with the similar "projectRoles"
// constant in <mr-header>.
const projectRoles = Object.freeze({
  PROJECT_ROLE_UNSPECIFIED: '',
  OWNER: 'Owner',
  COMMITTER: 'Committer',
  CONTRIBUTOR: 'Contributor',
});

/**
 * Creates a mapping of project names to the user's role in that project.
 * @param {Array<ProjectMember>} projectMembers Project memebrships
 *   for a given user.
 * @return {Object<ProjectName, string>} Mapping of a user's roles,
 *   by project name.
 */
export function computeRoleByProjectName(projectMembers) {
  const mapping = {};
  if (!projectMembers) return mapping;
  projectMembers.forEach(({name, role}) => {
    mapping[projectMemberToProjectName(name)] = projectRoles[role];
  });
  return mapping;
}
