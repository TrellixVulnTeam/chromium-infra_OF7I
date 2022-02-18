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
import {getOs, getBrowser, buildIssueDescription} from './issue-wizard/IssueWizardUtils.tsx'
import Header from './issue-wizard/Header.tsx'

import {GetQuestionsByCategory, buildIssueLabels, getCompValByCategory} from './issue-wizard/IssueWizardUtils.tsx';
import {ISSUE_WIZARD_QUESTIONS} from './issue-wizard/IssueWizardConfig.ts';
import {prpcClient} from 'prpc-client-instance.js';
import {expandDescriptions} from './issue-wizard/IssueWizardDescriptionsUtils.tsx';
/**
 * Base component for the issue filing wizard, wrapper for other components.
 * @return Issue wizard JSX.
 */
 type Props = {
  loginUrl: string,
  userDisplayName: string,
}
export function IssueWizard(props: Props): ReactElement {
  const {loginUrl, userDisplayName} = props;

  React.useEffect(() => {
    if(!userDisplayName) {
      window.location.href = loginUrl;
    }
  },[loginUrl, userDisplayName]);

  const [userPersona, setUserPersona] = React.useState(IssueWizardPersona.EndUser);
  const [activeStep, setActiveStep] = React.useState(0);
  const [category, setCategory] = React.useState('');
  const [isRegression, setIsRegression] = React.useState(false);
  const [textValues, setTextValues] = React.useState(
    {
      oneLineSummary: '',
      stepsToReproduce: '',
      describeProblem: '',
    });
    const [osName, setOsName] = React.useState(getOs())
    const [browserName, setBrowserName] = React.useState(getBrowser())

  const questionByCategory = GetQuestionsByCategory(ISSUE_WIZARD_QUESTIONS);

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
        page = <DetailsStep
          textValues={textValues}
          setTextValues={setTextValues}
          category={category}
          setActiveStep={setActiveStep}
          osName={osName}
          setOsName={setOsName}
          browserName={browserName}
          setBrowserName={setBrowserName}
          setIsRegression={setIsRegression}
    />;
   } else if (activeStep === 2) {
    const compValByCategory = getCompValByCategory(ISSUE_WIZARD_QUESTIONS);

    const onSubmitIssue = (comments: string, customQuestionsAnswers: Array<string>,) => {
      const summary = textValues.oneLineSummary;
      const component =  compValByCategory.get(category);
      const description = buildIssueDescription(textValues.stepsToReproduce, textValues.describeProblem, comments, osName, browserName);
      const labels = buildIssueLabels(category, osName);

      const {expandDescription, expandLabels, compVal} =
        expandDescriptions(category, customQuestionsAnswers, isRegression, description, labels, component);

      const response = prpcClient.call('monorail.v3.Issues', 'MakeIssue', {
        parent: 'projects/chromium',
        issue: {
          summary,
          status: {
            status: 'Untriaged',
          },
          components: [{
            component: 'projects/chromium/componentDefs/' + compVal
          }],
          labels: expandLabels,
        },
        description: expandDescription,
        });
        response.then(() => {
          // TODO: (Issue 10628) handle submit quesiton success
        }, () => {
          // TODO: (Issue 10628) handel submit question failed
        });
    }
    page = <CustomQuestionsStep setActiveStep={setActiveStep} questions={questionByCategory.get(category)} onSubmit={onSubmitIssue}/>;
  }

  return (
    <>
      <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Poppins"></link>
      <div className={styles.container}>
        <Header />
        {page}
      </div>
    </>
  );
}

/**
 * Renders the issue filing wizard page.
 * @param mount HTMLElement that the React component should be added to.
 * @param loginUrl redirect to login page
 * @param userDisplayName login user
 */
export function renderWizard(mount: HTMLElement, loginUrl: string, userDisplayName: string): void {
  ReactDOM.render(<IssueWizard loginUrl={loginUrl} userDisplayName={userDisplayName}/>, mount);
}
