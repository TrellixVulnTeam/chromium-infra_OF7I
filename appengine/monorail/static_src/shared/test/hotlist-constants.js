// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export const hotlistExample = {
  ownerRef: {
    userId: 12345678,
    displayName: 'example@example.com',
  },
  name: 'Hotlist-Name',
  summary: 'Summary',
  description: 'Description',
  defaultColSpec: 'Rank ID Summary',
  isPrivate: false,
};

export const hotlistRefExample = {

  owner: {
    userId: 12345678,
    displayName: 'example@example.com',
  },
  name: 'Hotlist-Name',
};

export const hotlistRefStringExample = '12345678:Hotlist-Name';

export const hotlistsExample = {[hotlistRefStringExample]: hotlistExample};
