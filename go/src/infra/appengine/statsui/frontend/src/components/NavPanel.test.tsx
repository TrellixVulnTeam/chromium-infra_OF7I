// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { screen } from '@testing-library/react';
import NavPanel from './NavPanel';
import { renderWithRedux } from '../utils/testUtils';

describe('when rendering NavPanel', () => {
  it('should render available data sources', () => {
    const ds1 = {
      name: 'test-ds1',
      prettyName: 'Test DataSource 1',
    };
    const ds2 = {
      name: 'test-ds2',
      prettyName: 'Test DataSource 2',
    };
    renderWithRedux(<NavPanel open={true} />, {
      dataSources: {
        available: [ds1, ds2],
      },
    });
    expect(screen.getByText(ds1.prettyName)).toBeInTheDocument();
    expect(screen.getByText(ds2.prettyName)).toBeInTheDocument();
  });
});
