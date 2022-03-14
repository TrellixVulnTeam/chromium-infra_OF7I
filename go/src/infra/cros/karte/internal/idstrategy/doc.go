// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// idstrategy contains strategies for converting concrete objects to store in Karte
// (actions and observations) into identities. There are two strategies in this package.
//
// ProdStrategy  -- Uses the current time and an entropy source.
// NaiveStrategy -- Has a global in-memory counter and produces entity1, entity2, ...
//
package idstrategy
