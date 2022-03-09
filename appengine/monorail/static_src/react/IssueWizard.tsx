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
import {getOs, getChromeVersion, buildIssueDescription} from './issue-wizard/IssueWizardUtils.tsx'
import Header from './issue-wizard/Header.tsx'

import {GetQuestionsByCategory, buildIssueLabels, getCompValByCategory, getLabelsByCategory} from './issue-wizard/IssueWizardUtils.tsx';
import {ISSUE_WIZARD_QUESTIONS, ISSUE_REPRODUCE_PLACEHOLDER, OS_CHANNEL_LIST} from './issue-wizard/IssueWizardConfig.ts';
import {prpcClient} from 'prpc-client-instance.js';
import {expandDescriptions} from './issue-wizard/IssueWizardDescriptionsUtils.tsx';
import SubmitSuccessStep from './issue-wizard/SubmitSuccessStep.tsx';

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
  const [newIssueID, setnewIssueID] = React.useState('');
  const [isRegression, setIsRegression] = React.useState(false);
  const [textValues, setTextValues] = React.useState(
    {
      oneLineSummary: '',
      stepsToReproduce: ISSUE_REPRODUCE_PLACEHOLDER,
      describeProblem: '',
      chromeVersion: getChromeVersion(),
      osName: getOs(),
      channel: OS_CHANNEL_LIST[0].name,
    });

  const questionByCategory = GetQuestionsByCategory(ISSUE_WIZARD_QUESTIONS);

  const reset = () => {
    setTextValues({
      oneLineSummary: '',
      stepsToReproduce: ISSUE_REPRODUCE_PLACEHOLDER,
      describeProblem: '',
      chromeVersion: getChromeVersion(),
      osName: getOs(),
      channel: OS_CHANNEL_LIST[0].name,
    });
    setIsRegression(false);
  }

  const updateCategory = (category: string) => {
    setCategory(category);
    reset();
  }

  let page;
  if (activeStep === 0) {
    page = <LandingStep
        userPersona={userPersona}
        setUserPersona={setUserPersona}
        category={category}
        setCategory={updateCategory}
        setActiveStep={setActiveStep}
        />;
      } else if (activeStep === 1) {
        page = <DetailsStep
          textValues={textValues}
          setTextValues={setTextValues}
          category={category}
          setActiveStep={setActiveStep}
          setIsRegression={setIsRegression}
    />;
   } else if (activeStep === 2) {
    const compValByCategory = getCompValByCategory(ISSUE_WIZARD_QUESTIONS);
    const labelsByCategory = getLabelsByCategory(ISSUE_WIZARD_QUESTIONS);

    const onSubmitIssue = (comments: string, customQuestionsAnswers: Array<string>, attachments: Array<any>,onSuccess: Function, onFailure: Function) => {
      const summary = textValues.oneLineSummary;
      const component =  compValByCategory.get(category);
      const description = buildIssueDescription(
        textValues.stepsToReproduce,
        textValues.describeProblem,
        comments, textValues.osName,
        textValues.chromeVersion,
        textValues.channel);
      const labels = buildIssueLabels(category, textValues.osName, textValues.chromeVersion, labelsByCategory.get(category));

      const {expandDescription, expandLabels, compVal} =
        expandDescriptions(category, customQuestionsAnswers, isRegression, description, labels, component);

      const componentsArray = [];
      if (compVal.length > 0) {
        componentsArray.push({
          component: 'projects/chromium/componentDefs/' + compVal
        })
      }

      const response = prpcClient.call('monorail.v3.Issues', 'MakeIssue', {
        parent: 'projects/chromium',
        issue: {
          summary,
          status: {
            status: 'Untriaged',
          },
          components: componentsArray,
          labels: expandLabels,
        },
        description: expandDescription,
        uploads: attachments,
        });
        response.then(onSuccess, onFailure);
    }
    page =
      <CustomQuestionsStep
        setActiveStep={setActiveStep}
        questions={questionByCategory.get(category)}
        onSubmit={onSubmitIssue}
        setnewIssueID={setnewIssueID}
      />;
  } else if (activeStep === 3) {
    page = <SubmitSuccessStep issueID={newIssueID}/>;
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
