// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useSelector } from 'react-redux';
import AppBar from '@material-ui/core/AppBar';
import { IconButton } from '@material-ui/core';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import MenuIcon from '@material-ui/icons/Menu';

import {
  selectAvailable,
  selectCurrentSource,
} from '../features/dataSources/dataSourcesSlice';
import { makeStyles } from '@material-ui/core/styles';

const useStyles = makeStyles((theme) => ({
  navBar: {
    zIndex: theme.zIndex.drawer + 1,
  },
}));

interface Props {
  toggleNavPanel?: () => void;
}

const NavBar: React.FunctionComponent<Props> = ({ toggleNavPanel }: Props) => {
  const classes = useStyles();
  const dataSource = useSelector(selectCurrentSource);
  const available = useSelector(selectAvailable);

  return (
    <AppBar position="relative" className={classes.navBar}>
      <Toolbar>
        {available.length > 1 && (
          <IconButton
            edge="start"
            color="inherit"
            onClick={toggleNavPanel}
            data-testid="nav-bar-toggle-panel"
          >
            <MenuIcon />
          </IconButton>
        )}
        <Typography variant="h6">
          {dataSource && dataSource.prettyName}
        </Typography>
      </Toolbar>
    </AppBar>
  );
};

export default NavBar;
