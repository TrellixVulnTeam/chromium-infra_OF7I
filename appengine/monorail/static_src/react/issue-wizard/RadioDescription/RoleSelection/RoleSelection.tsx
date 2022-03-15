// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { makeStyles, withStyles } from '@material-ui/styles';
import { blue, grey } from '@material-ui/core/colors';
import Radio, { RadioProps } from '@material-ui/core/Radio';

const useStyles = makeStyles({
  container: {
    width: '320px',
    minWidth: '140px',
    height: '160px',
    position: 'relative',
    display: 'inline-block',
    cursor: 'pointer',
  },
  text: {
    position: 'absolute',
    display: 'inline-block',
  },
  title: {
    margin: '0.5rem 0',
    fontSize: '1.125rem',
    color: grey[900],
  },
  subheader: {
    fontSize: '0.875rem',
    color: grey[800],
  },
  line: {
    position: 'absolute',
    bottom: 0,
    width: '300px',
    left: '20px',
  }
});

const BlueRadio = withStyles({
  root: {
    color: blue[400],
    '&$checked': {
      color: blue[600],
    },
  },
  checked: {},
})((props: RadioProps) => <Radio color="default" {...props} />);

interface RoleSelectionProps {
  /* Whether or not the radio button should be checked */
  checked: boolean
  /* onClick callback defined in parent component */
  handleOnClick: (event: React.MouseEvent<HTMLElement>) => void
  /*
      A string representing the type of user; this is the value of the input
      see `userGroups`, which is defined in RadioDescription
  */
  value: string
  /* Descriptive text to be displayed along with the radio button */
  description: string
  /* Additional props for the radio button component */
  inputProps: { [key: string]: string }
}

/**
 * RoleSelection encapsulates the radio button and details
 * for selecting a role as an issue reporter in the issue wizard
 * @see RadioDescription
 */

export const RoleSelection = ({
  checked,
  handleOnClick,
  value,
  description,
  inputProps
}: RoleSelectionProps): React.ReactElement => {
  const classes = useStyles();
  return (
    <div className={classes.container} onClick={handleOnClick}>
      <BlueRadio
        checked={checked}
        value={value}
        inputProps={inputProps}
      />
      <div className={classes.text}>
        <p className={classes.title}>{value}</p>
        <p className={classes.subheader}>{description}</p>
      </div>
      <hr color={grey[200]} className={classes.line} />
    </div>
  )
}

