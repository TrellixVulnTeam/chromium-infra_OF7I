// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { screen } from '@testing-library/react';
import NavBar from './NavBar';
import { renderWithRedux } from '../utils/testUtils';

describe('when rendering NavBar', () => {
  it('should render the current data source', () => {
    const ds = {
      name: 'test-ds',
      prettyName: 'Test DataSource',
    };
    renderWithRedux(<NavBar />, {
      dataSources: {
        current: ds.name,
        available: [ds],
        availableMap: {
          'test-ds': ds,
        },
      },
    });
    expect(screen.getByText('Test DataSource')).toBeInTheDocument();
  });

  it('should render ', () => {
    const ds1 = {
      name: 'ds1',
      prettyName: 'DS 1',
    };
    const ds2 = {
      name: 'ds2',
      prettyName: 'DS 2',
    };
    renderWithRedux(<NavBar />, {
      dataSources: {
        current: '',
        available: [ds1, ds2],
      },
    });
    expect(screen.getByTestId('toggle-nav-panel-button')).toBeInTheDocument();
  });
});
