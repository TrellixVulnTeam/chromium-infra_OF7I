// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * simplified typescript version of "google/protobuf/field_mask.proto"
 * it helps to serialize/deserialize the field_mask object to json.
 */
export interface FieldMask {
  /** The set of field mask paths. */
  paths: string[];
}

export const FieldMask = {
  fromJSON(object: any): FieldMask {
    return {
      paths:
        typeof object === 'string'
          ? object.split(',').filter(Boolean)
          : Array.isArray(object?.paths)
          ? object.paths.map(String)
          : [],
    };
  },

  toJSON(message: FieldMask): string {
    return message.paths.join(',');
  },

  wrap(paths: string[]): FieldMask {
    return { paths: paths };
  },

  unwrap(message: FieldMask): string[] {
    return message.paths;
  },
};
