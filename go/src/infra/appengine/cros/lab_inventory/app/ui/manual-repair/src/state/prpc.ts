// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {PrpcClient} from '@chopsui/prpc-client';

export const prpcClient = new PrpcClient({
  host: process.env.NODE_ENV === 'development' ?
      'cros-lab-inventory-dev.appspot.com' :
      'cros-lab-inventory.appspot.com',
  insecure: false,
});
