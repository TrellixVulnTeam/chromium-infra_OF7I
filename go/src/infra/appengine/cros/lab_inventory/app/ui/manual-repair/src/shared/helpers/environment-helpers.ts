// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * getManualRepairBaseUrl returns the base url of the application depending on
 * the environment it is deployed in.
 */
export function getManualRepairBaseUrl(): string {
  if (process.env.NODE_ENV === 'staging') {
    return 'https://manual-repair-dot-cros-lab-inventory-dev.appspot.com/'
  } else if (process.env.NODE_ENV === 'production') {
    return 'https://manual-repair-dot-cros-lab-inventory.appspot.com/'
  } else {
    return 'http://localhost:8080/'
  }
}
