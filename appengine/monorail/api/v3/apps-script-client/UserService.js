// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/* eslint-disable no-unused-vars */

/**
 * Fetches the user from Monorai.
 * @param {string} userName The resource name of the user.
 * @return {User}
 */
function getUser(userName) {
  const message = {'name': userName};
  const url = URL + 'monorail.v3.Users/GetUser';
  return run_(url, message);
}

/**
 * Fetches the users from Monorail.
 * @param {Array<string>} userNames The resource names of the users.
 * @return {Array<User>}
 */
function batchGetUsers(userNames) {
  const message = {'names': userNames};
  const url = URL + 'monorail.v3.Users/BatchGetUsers';
  return run_(url, message);
}
