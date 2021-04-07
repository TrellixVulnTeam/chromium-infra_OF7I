// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// merge merges multiple arrays and removes duplicate values.
export function merge<T>(...arrays: T[][]): T[] {
  const ret: T[] = [];
  arrays.forEach((array) => ret.push(...array.filter((v) => !ret.includes(v))));
  return ret;
}

// removeFrom removes values from a given array.
export function removeFrom<T>(from: T[], ...sets: T[][]): T[] {
  const set = merge(...sets);
  return from.filter((i) => !set.includes(i));
}
