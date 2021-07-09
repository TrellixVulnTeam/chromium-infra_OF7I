// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { screen } from '@testing-library/react';
import App from './App';
import { renderWithRedux } from './utils/testUtils';

jest.mock('./pages/MetricsPage', () => {
  const MockMetricsPage = () => <div data-testid="metrics-page" />;
  return MockMetricsPage;
});

describe('when rendering the application', () => {
  it('should render the metrics page', () => {
    renderWithRedux(<App />);
    expect(screen.getByTestId('metrics-page')).toBeInTheDocument();
  });
});
