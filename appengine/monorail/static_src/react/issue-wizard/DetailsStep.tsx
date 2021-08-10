// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {createStyles, createTheme} from '@material-ui/core/styles';
import {makeStyles} from '@material-ui/styles';
import TextField from '@material-ui/core/TextField';
import {red, grey} from '@material-ui/core/colors';

/**
 * The detail step is the second step on the dot
 * stepper. This react component provides the users with
 * specific questions about their bug to be filled out.
 */
const theme: Theme = createTheme();

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    root: {
      '& > *': {
        margin: theme.spacing(1),
        width: '100%',
      },
    },
    head: {
        marginTop: '25px',
    },
    red: {
        color: red[600],
    },
    grey: {
        color: grey[600],
    },
  }), {defaultTheme: theme}
);

export default function DetailsStep(): React.ReactElement {
  const classes = useStyles();

  return (
    <>
        <h2 className={classes.grey}>Details for problems with</h2>
        <form className={classes.root} noValidate autoComplete="off">
            <h3 className={classes.head}>Please enter a one line summary <span className={classes.red}>*</span></h3>
            <TextField id="outlined-basic-1" variant="outlined" />

            <h3 className={classes.head}>Steps to reproduce problem <span className={classes.red}>*</span></h3>
            <TextField multiline rows={4} id="outlined-basic-2" variant="outlined" />

            <h3 className={classes.head}>Please describe the problem <span className={classes.red}>*</span></h3>
            <TextField multiline rows={3} id="outlined-basic-3" variant="outlined" />

            <h3 className={classes.head}>Additional Comments</h3>
            <TextField multiline rows={3} id="outlined-basic-3" variant="outlined" />
        </form>
    </>
  );
}
