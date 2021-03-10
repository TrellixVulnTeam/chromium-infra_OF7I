// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {PrpcClient} from '@chopsui/prpc-client';

export const inventoryApiVersion = 'inventory.Inventory'

export const inventoryClient = new PrpcClient({
  host: process.env.NODE_ENV === 'development' ?
      'cros-lab-inventory-dev.appspot.com' :
      'cros-lab-inventory.appspot.com',
  insecure: false,
});

export const ufsApiVersion = 'unifiedfleet.api.v1.rpc.Fleet'

export const ufsClient = new PrpcClient({
  host: process.env.NODE_ENV === 'development' ? 'staging.ufs.api.cr.dev' :
                                                 'ufs.api.cr.dev',
  insecure: false,
});

// TODO: need to add 'Grpc-Metadata-namespace': 'os' header in the future as
// UFS will block calls without the header.
export const defaultUfsHeaders: {[key: string]: any} = {
  'User-Agent': 'manual-repair/6.0.0',
}
