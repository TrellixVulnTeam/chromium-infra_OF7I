// Copyright 202 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import logo from './logo.svg';
import { Lab } from './features/lab/Lab';
import './App.css';

function App() {
  return (
    <div className="App">
      <header className="App-header">
        <Lab />
      </header>
    </div>
  );
}

export default App;
