// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import Navigo from 'navigo';

/**
 * Universal router used by other views. Routes are defined in
 * manual-repair/src/components/manual-repair.ts. All routes will be using
 * hash-based routing.
 */
export const router = new Navigo('/', true, '#');
