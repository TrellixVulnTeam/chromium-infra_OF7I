// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { makeStyles } from '@material-ui/styles';
import { RoleSelection } from './RoleSelection/RoleSelection.tsx';
import {ISSUE_WIZARD_PERSONAS_DETAIL, IssueWizardPersona} from '../IssueWizardTypes.tsx';

const useStyles = makeStyles({
  flex: {
    display: 'flex',
    justifyContent: 'space-between',
  }
});

const getUserGroupSelectors = (
  value: IssueWizardPersona,
  onSelectorClick:
    (selector: string) =>
      (event: React.MouseEvent<HTMLElement>) => any) => {
  const selectors = new Array();
  Object.entries(ISSUE_WIZARD_PERSONAS_DETAIL).forEach(([key, persona]) => {
    selectors.push(
        <RoleSelection
          checked={IssueWizardPersona[value] === key}
          handleOnClick={onSelectorClick(key)}
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
type Props = {
  selectedRadio: IssueWizardPersona,
  onClickRadio: Function,
}

export const RadioDescription = (props: Props): React.ReactElement => {
  const { selectedRadio, onClickRadio } = props;
  const classes = useStyles();

  const handleRoleSelectionClick = (userGroup: string) =>
     (event: React.MouseEvent<HTMLElement>) => onClickRadio(userGroup);

  const userGroupsSelectors = getUserGroupSelectors(selectedRadio, handleRoleSelectionClick);

  return (
    <div className={classes.flex}>
      {userGroupsSelectors}
    </div>
  );
}
