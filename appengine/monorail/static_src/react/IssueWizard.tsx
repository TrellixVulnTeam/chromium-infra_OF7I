// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {ReactElement} from 'react';
import * as React from 'react'
import ReactDOM from 'react-dom';
import styles from './IssueWizard.css';
import LandingStep from './issue-wizard/LandingStep.tsx';
import DetailsStep from './issue-wizard/DetailsStep.tsx'
import {IssueWizardPersona} from './issue-wizard/IssueWizardTypes.tsx';
import CustomQuestionsStep from './issue-wizard/CustomQuestionsStep.tsx';

/**
 * Base component for the issue filing wizard, wrapper for other components.
 * @return Issue wizard JSX.
 */
export function IssueWizard(): ReactElement {
  const [userPersona, setUserPersona] = React.useState(IssueWizardPersona.EndUser);
  const [activeStep, setActiveStep] = React.useState(0);
  const [category, setCategory] = React.useState('');
  const [textValues, setTextValues] = React.useState(
    {
      oneLineSummary: '',
      stepsToReproduce: '',
      describeProblem: '',
      additionalComments: ''
    });

  let page;
  if (activeStep === 0) {
    page = <LandingStep
        userPersona={userPersona}
        setUserPersona={setUserPersona}
        category={category}
        setCategory={setCategory}
        setActiveStep={setActiveStep}
        />;
  } else if (activeStep === 1) {
    page = <DetailsStep textValues={textValues} setTextValues={setTextValues} category={category} setActiveStep={setActiveStep}/>;
  } else if (activeStep === 2) {
    page = <CustomQuestionsStep setActiveStep={setActiveStep}/>;
  }

  return (
    <>
      <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Poppins"></link>
      <div className={styles.container}>
        {page}
      </div>
    </>
  );
}

/**
 * Renders the issue filing wizard page.
 * @param mount HTMLElement that the React component should be
 *   added to.
 */
export function renderWizard(mount: HTMLElement): void {
  ReactDOM.render(<IssueWizard />, mount);
}
