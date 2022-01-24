// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { makeStyles } from '@material-ui/styles';
import { RoleSelection } from './RoleSelection/RoleSelection.tsx';
import {ISSUE_WIZARD_PERSONAS_DETAIL} from '../IssueWizardTypes.tsx';

const useStyles = makeStyles({
  flex: {
    display: 'flex',
    justifyContent: 'space-between',
  }
});

const getUserGroupSelectors = (
  value: string,
  onSelectorClick:
    (selector: string) =>
      (event: React.MouseEvent<HTMLElement>) => any) => {
  const selectors = new Array();
  Object.values(ISSUE_WIZARD_PERSONAS_DETAIL).forEach((persona) => {
    selectors.push(
        <RoleSelection
          checked={value === persona.name}
          handleOnClick={onSelectorClick(persona.name)}
          value={persona.name}
          description={persona.description}
          inputProps={{ 'aria-label': persona.name }}
        />
      );
  });
  return selectors;
}
/**
 * RadioDescription contains a set of radio buttons and descriptions (RoleSelection)
 * to be chosen from in the landing step of the Issue Wizard.
 *
 * @returns React.ReactElement
 */
export const RadioDescription = ({ value, setValue }: { value: string, setValue: Function }): React.ReactElement => {
  const classes = useStyles();

  const handleRoleSelectionClick = (userGroup: string) =>
     (event: React.MouseEvent<HTMLElement>) => setValue(userGroup);

  const userGroupsSelectors = getUserGroupSelectors(value, handleRoleSelectionClick);

  return (
    <div className={classes.flex}>
      {userGroupsSelectors}
    </div>
  );
}
