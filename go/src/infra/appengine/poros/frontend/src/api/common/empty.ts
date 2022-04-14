// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * A generic empty message that you can re-use to avoid defining duplicated
 * empty messages in your APIs. A typical example is to use it as the request
 * or the response type of an API method.
 * The JSON representation for `Empty` is empty JSON object `{}`.
 */
// eslint-disable-next-line
export interface Empty {}

export const Empty = {
  // eslint-disable-next-line
  fromJSON(_: any): Empty {
    return {};
  },
  // eslint-disable-next-line
  toJSON(_: Empty): unknown {
    const obj: any = {};
    return obj;
  },
};
