// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography'



export default function Header() {
  return (
    <>
    <AppBar sx={{bgcolor: "white"}}>
      <Toolbar>
        <img src='/static/images/chromium.svg' width='=40' height='40'/>
        <Typography variant="h5" component="div" color="black"> Bugs</Typography>
      </Toolbar>
    </AppBar>
    <Toolbar />
    </>
  );


}