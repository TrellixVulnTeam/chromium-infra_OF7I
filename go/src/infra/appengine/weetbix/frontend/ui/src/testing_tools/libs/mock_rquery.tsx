// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {
  QueryClient,
  QueryClientProvider
} from 'react-query';

import { render } from '@testing-library/react';

export const renderWithClient = (ui: React.ReactElement) => {
  const client = new QueryClient();

  return render(
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
  );
};