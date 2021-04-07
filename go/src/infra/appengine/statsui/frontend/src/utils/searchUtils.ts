// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export function search(text: string, query: string): boolean {
  if (query === '') {
    return true;
  }
  text = text.toLowerCase();
  return query
    .toLowerCase()
    .split(' ')
    .every((term) => text.includes(term));
}
