// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * filterUndefinedKeys removes all undefined fields from an object.
 */
export function filterUndefinedKeys(obj: object) {
  const ret = {};
  Object.keys(obj)
      .filter((key) => obj[key] !== undefined)
      .forEach((key) => ret[key] = obj[key]);
  return ret;
}

/**
 * filterZeroFromSet takes in a set of numbers. If set contains 0, remove the 0
 * and return the rest. Otherwise, return just a set with the 0.
 */
export function filterZeroFromSet(actionsSet: Set<number>): Set<number> {
  let resSet = new Set(actionsSet);

  if (resSet.has(0) && resSet.size > 1) {
    resSet.delete(0);
  } else if (resSet.size === 0) {
    resSet = new Set([0]);
  }

  return resSet;
}
