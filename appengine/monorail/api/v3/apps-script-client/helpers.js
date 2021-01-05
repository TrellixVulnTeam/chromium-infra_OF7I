// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/* eslint-disable no-unused-vars */

/**
 * Returns the user's resource name.
 * @param {string|number} user The user's email or user_id
 * @return {string}
 */
function computeUserName(user) {
  return `users/${user}`;
}

/**
 * Returns the users' resource names.
 * @param {Array<string|number>} users Array of user emails/user_ids.
 * @return {Array<string>}
 */
function computeUserNames(users) {
  const userNames = [];
  users.forEach((user) => {
    userNames.push(computeUserName(user));
  });
  return userNames;
}


/**
 * Returns the issue's resource name.
 * @param {string} project The name of the project the issue belongs to,
 *     e.g. 'chromium'.
 * @param {number} id The issue's id.
 * @return {string}
 */
function computeIssueName(project, id) {
  return `projects/${project}/issues/${id}`;
}

/**
 * Returns the project's resource name.
 * @param {string} project The display name of the project, e.g. 'chromium'.
 * @return {string}
 */
function computeProjectName(project) {
  return `projects/${project}`;
}

/**
 * Returns the projects' resource names in the same order.
 * @param {Array<string>} projects The display names of the projects,
 *     e.g. 'chromium'.
 * @return {Array<string>}
 */
function computeProjectNames(projects) {
  const projectNames = [];
  projects.forEach((project) => {
    projectNames.push(computeProjectName(project));
  });
  return projectNames;
}

/**
 * Returns the FieldDef's resource name.
 * @param {string} project The display name of the project, e.g. 'chromium'.
 * @param {number} fieldId ID of the FieldDef.
 * @return {string}
 */
function computeFieldDefName(project, fieldId) {
  return `projects/${project}/fieldDefs/${fieldId}`;
}

/**
 * Returns the ComponentDef's resource name.
 * @param {string} project The display name of the project, e.g. 'chromium'.
 * @param {number|string} componentIdOrPath ID or value of the ComponentDef.
 * @return {string}
*/
function computeComponentDefName(project, componentIdOrPath) {
  return `projects/${project}/componentDefs/${componentIdOrPath}`;
}
