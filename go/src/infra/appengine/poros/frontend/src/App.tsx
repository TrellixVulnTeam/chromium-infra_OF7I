// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import './App.css';
import { BrowserRouter } from 'react-router-dom';

import SideDrawerWithAppBar from './features/utility/App.Bar';

function App() {
  return (
    <BrowserRouter>
      <div className="App">
        <SideDrawerWithAppBar />
      </div>
    </BrowserRouter>
  );
}

export default App;
