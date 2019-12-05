// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview This file collects helpers for converting between
 * Hotlist-related object formats. It's recommended to use the helpers
 * in this file to ensure consistency across conversions.
 */

import './typedef.js';

/**
 * Converts a full Hotlist Object into only the pieces of its data needed
 * to define an HotlistRef. Useful for cases when we don't want to send excess
 * information to ifentify an Hotlist.
 * @param {Hotlist} hotlist A full Hotlist Object.
 * @return {HotlistRef} Just the ID part of the Hotlist Object.
 */
export function hotlistToRef(hotlist) {
  return {name: hotlist.name, owner: hotlist.ownerRef};
}

/**
 * Converts a HotlistRef into a canonical String format.
 * I.e.: "12345678:Hotlist-Name"
 * @param {HotlistRef} hotlistRef Just the ID part of the Hotlist Object.
 * @return {string} A String with all the data needed to construct a HotlistRef.
 */
export function hotlistRefToString(hotlistRef) {
  return `${hotlistRef.owner.userId}:${hotlistRef.name}`;
}

/**
 * Converts a full Hotlist Object into only the pieces of its data needed
 * to define an HotlistRef. Useful for cases when we don't want to send excess
 * information to ifentify an Hotlist.
 * @param {Hotlist} hotlist A full Hotlist Object.
 * @return {string} A String with all the data needed to construct a HotlistRef.
 */
export function hotlistToRefString(hotlist) {
  const ref = hotlistToRef(hotlist);
  return hotlistRefToString(ref);
}
