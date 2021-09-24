// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { makeStyles } from '@material-ui/styles';
import { RoleSelection } from './RoleSelection/RoleSelection.tsx';

const userGroups = Object.freeze({
  END_USER: 'End User',
  WEB_DEVELOPER: 'Web Developer',
  CONTRIBUTOR: 'Chromium Contributor',
});

const useStyles = makeStyles({
  flex: {
    display: 'flex',
    justifyContent: 'space-between',
  }
});

/**
 * RadioDescription contains a set of radio buttons and descriptions (RoleSelection)
 * to be chosen from in the landing step of the Issue Wizard.
 *
 * @returns React.ReactElement
 */
export const RadioDescription = ({ value, setValue }: { value: string, setValue: Function }): React.ReactElement => {
  const classes = useStyles();

  const handleRoleSelectionClick = (userGroup: string) =>
    (event: React.MouseEvent<HTMLElement>) => setValue(userGroup)

  return (
    <div className={classes.flex}>
      <RoleSelection
        checked={value === userGroups.END_USER}
        handleOnClick={handleRoleSelectionClick(userGroups.END_USER)}
        value={userGroups.END_USER}
        description="I am a user trying to do something on a website."
        inputProps={{ 'aria-label': userGroups.END_USER }}
      />
      <RoleSelection
        checked={value === userGroups.WEB_DEVELOPER}
        handleOnClick={handleRoleSelectionClick(userGroups.WEB_DEVELOPER)}
        value={userGroups.WEB_DEVELOPER}
        description="I am a web developer trying to build something."
        inputProps={{ 'aria-label': userGroups.WEB_DEVELOPER }}
      />
      <RoleSelection
        checked={value === userGroups.CONTRIBUTOR}
        handleOnClick={handleRoleSelectionClick(userGroups.CONTRIBUTOR)}
        value={userGroups.CONTRIBUTOR}
        description="I know about a problem in specific tests or code."
        inputProps={{ 'aria-label': userGroups.CONTRIBUTOR }}
      />
    </div>
  );
}