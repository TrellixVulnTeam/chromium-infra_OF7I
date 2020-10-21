// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * Parse a query string into a key-value store and return.
 *
 * @param queryString Query string separated by '&'.
 * @returns           A key-value store of the query parameters parsed.
 */
export function parseQueryStringToDict(queryString: string):
    {[key: string]: string} {
  const queryStore = {};
  const pairs = queryString.split('&');
  for (let i = 0; i < pairs.length; i++) {
    let pair = pairs[i].split('=');
    if (pair[0] !== '') {
      queryStore[decodeURIComponent(pair[0])] =
          decodeURIComponent(pair[1] || '');
    }
  }
  return queryStore;
}
