// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {ReactElement} from 'react';
import * as React from 'react'
import ReactDOM from 'react-dom';
import styles from './IssueWizard.css';
import DotMobileStepper from './issue-wizard/DotMobileStepper.tsx';
import LandingStep from './issue-wizard/LandingStep.tsx';
import DetailsStep from './issue-wizard/DetailsStep.tsx'

/**
 * Base component for the issue filing wizard, wrapper for other components.
 * @return Issue wizard JSX.
 */
export function IssueWizard(): ReactElement {
  const [checkExisting, setCheckExisting] = React.useState(false);
  const [userType, setUserType] = React.useState('End User');
  const [activeStep, setActiveStep] = React.useState(0);
  const [category, setCategory] = React.useState('');
  const [textValues, setTextValues] = React.useState(
    {
      oneLineSummary: '',
      stepsToReproduce: '',
      describeProblem: '',
      additionalComments: ''
    });

  let nextEnabled;
  let page;
  if (activeStep === 0){
    page = <LandingStep
        checkExisting={checkExisting}
        setCheckExisting={setCheckExisting}
        userType={userType}
        setUserType={setUserType}
        category={category}
        setCategory={setCategory}
        />;
    nextEnabled = checkExisting && userType && (category != '');
  } else if (activeStep === 1){
    page = <DetailsStep textValues={textValues} setTextValues={setTextValues} category={category}/>;
    nextEnabled = (textValues.oneLineSummary.trim() !== '') &&
                  (textValues.stepsToReproduce.trim() !== '') &&
                  (textValues.describeProblem.trim() !== '');
  }

  return (
    <>
      <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Poppins"></link>
      <div className={styles.container}>
        {page}
        <DotMobileStepper nextEnabled={nextEnabled} activeStep={activeStep} setActiveStep={setActiveStep}/>
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
