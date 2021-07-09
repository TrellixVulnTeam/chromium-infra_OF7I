// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useSelector } from 'react-redux';
import { Drawer, List, ListItem, ListItemText } from '@material-ui/core';

import { selectAvailable } from '../features/dataSources/dataSourcesSlice';

import styles from './NavPanel.module.css';

interface Props {
  open: boolean;
}

const NavPanel: React.FunctionComponent<Props> = ({ open }: Props) => {
  const available = useSelector(selectAvailable);

  return (
    <Drawer
      variant="persistent"
      anchor="left"
      open={open}
      className={styles.nav}
    >
      <List className={styles.list}>
        {available.map((datasource) => (
          <ListItem key={datasource.name} button>
            <ListItemText primary={datasource.prettyName} />
          </ListItem>
        ))}
      </List>
    </Drawer>
  );
};

export default NavPanel;
