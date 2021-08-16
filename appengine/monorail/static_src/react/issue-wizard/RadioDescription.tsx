// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {makeStyles, withStyles} from '@material-ui/styles';
import {blue, grey} from '@material-ui/core/colors';
import Radio, {RadioProps} from '@material-ui/core/Radio';

const userGroups = Object.freeze({
  END_USER: 'End User',
  WEB_DEVELOPER: 'Web Developer',
  CONTRIBUTOR: 'Chromium Contributor',
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

const useStyles = makeStyles({
  flex: {
    display: 'flex',
    justifyContent: 'space-between',
  },
  container: {
    width: '320px',
    height: '150px',
    position: 'relative',
    display: 'inline-block',
  },
  text: {
    position: 'absolute',
    display: 'inline-block',
    left: '55px',
  },
  title: {
    marginTop: '7px',
    fontSize: '20px',
    color: grey[900],
  },
  subheader: {
    fontSize: '16px',
    color: grey[800],
  },
  line: {
    position: 'absolute',
    bottom: 0,
    width: '300px',
    left: '20px',
  }
});

/**
 * `<RadioDescription />`
 *
 * React component for radio buttons and their descriptions
 * on the landing step of the Issue Wizard.
 *
 *  @return ReactElement.
 */
export default function RadioDescription({value, setValue} : {value: string, setValue: Function}): React.ReactElement {
  const classes = useStyles();

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setValue(event.target.value);
  };

  return (
    <div className={classes.flex}>
      <div className={classes.container}>
        <BlueRadio
          checked={value === userGroups.END_USER}
          onChange={handleChange}
          value={userGroups.END_USER}
          inputProps={{ 'aria-label': userGroups.END_USER}}
        />
        <div className={classes.text}>
          <p className={classes.title}>{userGroups.END_USER}</p>
          <p className={classes.subheader}>I am a user trying to do something on a website.</p>
        </div>
        <hr color={grey[200]} className={classes.line}/>
      </div>
      <div className={classes.container}>
        <BlueRadio
          checked={value === userGroups.WEB_DEVELOPER}
          onChange={handleChange}
          value={userGroups.WEB_DEVELOPER}
          inputProps={{ 'aria-label': userGroups.WEB_DEVELOPER }}
        />
        <div className={classes.text}>
          <p className={classes.title}>{userGroups.WEB_DEVELOPER}</p>
          <p className={classes.subheader}>I am a web developer trying to build something.</p>
        </div>
        <hr color={grey[200]} className={classes.line}/>
      </div>
      <div className={classes.container}>
        <BlueRadio
          checked={value === userGroups.CONTRIBUTOR}
          onChange={handleChange}
          value={userGroups.CONTRIBUTOR}
          inputProps={{ 'aria-label': userGroups.CONTRIBUTOR }}
        />
        <div className={classes.text}>
          <p className={classes.title}>{userGroups.CONTRIBUTOR}</p>
          <p className={classes.subheader}>I know about a problem in specific tests or code.</p>
        </div>
        <hr color={grey[200]} className={classes.line}/>
      </div>
    </div>
    );
  }