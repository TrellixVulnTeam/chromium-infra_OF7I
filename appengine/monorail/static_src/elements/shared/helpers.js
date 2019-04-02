// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// With lists a and b, get the elements that are in a but not in b.
// result = a - b
export function arrayDifference(listA, listB, equals) {
  if (!equals) {
    equals = (a, b) => (a === b);
  }
  listA = listA || [];
  listB = listB || [];
  return listA.filter((a) => {
    return !listB.find((b) => (equals(a, b)));
  });
}

export function removePrefix(str, prefix) {
  return str.substr(prefix.length);
}

// TODO(zhangtiff): Make this more grammatically correct for
// more than two items.
export function arrayToEnglish(arr) {
  if (!arr) return '';
  return arr.join(' and ');
}

export function isEmptyObject(obj) {
  return Object.keys(obj).length === 0;
}

export function equalsIgnoreCase(a, b) {
  return a.toLowerCase() === b.toLowerCase();
}
