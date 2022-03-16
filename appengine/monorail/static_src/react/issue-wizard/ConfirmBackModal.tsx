// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import {makeStyles} from '@material-ui/styles';
import Dialog from '@material-ui/core/Dialog';
import DialogTitle from '@material-ui/core/DialogTitle';
import DialogContent from '@material-ui/core/DialogContent';
import DialogContentText from '@material-ui/core/DialogContentText';
import DialogActions from '@material-ui/core/DialogActions';
import Button from '@material-ui/core/Button';

const userStyles = makeStyles({
  actionsButtons: {
    paddingTop: '0',
  },
  primaryButton: {
    backgroundColor: 'rgb(25, 118, 210)',
    color: 'white',
  }
});

type Props = {
  enable: boolean,
  setEnable: Function,
  confirmBack: Function,
}

export function ConfirmBackModal(props: Props): React.ReactElement {
  const {enable, setEnable, confirmBack} = props;
  const classes = userStyles();

  return (
    <Dialog open={enable}>
        <DialogTitle>Warning!</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Changes you made on this page won't be saved.
          </DialogContentText>
        </DialogContent>
        <DialogActions className={classes.actionsButtons}>
          <Button onClick={() => setEnable(false)}>Cancel</Button>
          <Button onClick={() => {
            confirmBack();
            setEnable(false);
          }} className={classes.primaryButton}>Ok</Button>
        </DialogActions>
    </Dialog>
  )
}
