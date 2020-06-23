// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

const USER_NAME_REGEX = /users\/(\d+)/i;

// See also: https://source.chromium.org/chromium/infra/infra/+/master:appengine/monorail/project/project_constants.py;l=13
const PROJECT_NAME_REGEX = /projects\/([a-z0-9][-a-z0-9]*[a-z0-9])/i;

/**
 * Returns a FieldMask given an array of string paths.
 * https://developers.google.com/protocol-buffers/docs/reference/csharp/class/google/protobuf/well-known-types/field-mask#paths
 * https://source.chromium.org/chromium/chromium/src/+/master:third_party/protobuf/python/google/protobuf/internal/well_known_types.py;l=425;drc=e10d98917fee771b0947a57468d1cadac446bc42
 * @param {Array<string>} paths The given paths to turn into a field mask.
 *   These should be a comma separated list of camel case strings.
 * @return {string}
 */
export function pathsToFieldMask(paths) {
  return paths.join(',');
}

/**
 * Extract a User ID from a User resource name.
 * @param {UserName} user User resource name.
 * @return {string} User ID.
 * @throws {Error} if the User resource name is invalid.
 */
export function extractUserId(user) {
  const matches = user.match(USER_NAME_REGEX);
  if (!matches) {
    throw new Error('Improperly formatted resource name.');
  }
  return matches[1];
}

/**
 * Extract a project's displayName from a Project resource name.
 * @param {ProjectName} project Project resource name.
 * @return {string} The project's displayName.
 * @throws {Error} if the Project resource name is invalid.
 */
export function extractProjectDisplayName(project) {
  const matches = project.match(PROJECT_NAME_REGEX);
  if (!matches) {
    throw new Error('Improperly formatted resource name.');
  }
  return matches[1];
}

/**
 * Creates a ProjectStar resource name based on a UserName nad a ProjectName.
 * @param {ProjectName} project Resource name of the referenced project.
 * @param {UserName} user Resource name of the referenced user.
 * @return {ProjectStarName}
 * @throws {Error} If the project or user resource name is invalid.
 */
export function projectAndUserToStarName(project, user) {
  const userId = extractUserId(user);
  const projectName = extractProjectDisplayName(project);
  return `users/${userId}/projectStars/${projectName}`;
}
