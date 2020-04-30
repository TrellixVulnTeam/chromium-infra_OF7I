// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

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
