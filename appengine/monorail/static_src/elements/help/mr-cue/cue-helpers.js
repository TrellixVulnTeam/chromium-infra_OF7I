// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared helpers for dealing with how <mr-cue> element instances
 * are used.
 */

export const cueNames = Object.freeze({
  CODE_OF_CONDUCT: 'code_of_conduct',
  AVAILABILITY_MSGS: 'availability_msgs',
  SWITCH_TO_PARENT_ACCOUNT: 'switch_to_parent_account',
  SEARCH_FOR_NUMBERS: 'search_for_numbers',
});

export const AVAILABLE_CUES = Object.freeze(new Set(Object.values(cueNames)));

export const CUE_DISPLAY_PREFIX = 'cue.';

/**
 * Converts a cue name to the format expected by components like <mr-metadata>
 * for the purpose of ordering fields.
 *
 * @param {String} cueName The name of the cue.
 * @return {String} A "cue.cue_name" formatted String used in ordering cues
 *   alongside field types (ie: Owner) in various field specs.
 */
export const cueNameToSpec = (cueName) => {
  return CUE_DISPLAY_PREFIX + cueName;
};

/**
 * Converts an issue field specifier to the name of the cue it references if
 * it references a cue. ie: "cue.cue_name" would reference "cue_name".
 *
 * @param {String} spec A "cue.cue_name" format String specifying that a
 *   specific cue should be mixed alongside issue fields in a component like
 *   <mr-metadata>.
 * @return {String} Name of the cue customized in the spec or an empty
 *   String if the spec does not reference a cue.
 */
export const specToCueName = (spec) => {
  spec = spec.toLowerCase();
  if (spec.startsWith(CUE_DISPLAY_PREFIX)) {
    return spec.substring(CUE_DISPLAY_PREFIX.length);
  }
  return '';
};
