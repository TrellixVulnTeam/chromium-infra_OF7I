// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react'
import {makeStyles} from '@material-ui/styles';
import {grey} from '@material-ui/core/colors';
import DotMobileStepper from './DotMobileStepper.tsx';
import {CustomQuestion, CustomQuestionType} from './IssueWizardTypes.tsx';
import InputTypeCustomQuestion from './CustomQuestions/InputTypeCustomQuestion.tsx';
import TextareaTypeCustomQuestion from './CustomQuestions/TextareaTypeCustomQuestion.tsx';

const userStyles = makeStyles({
  greyText: {
    color: grey[600],
  },
  root: {
    width: '100%',
  },
});

type Props = {
  setActiveStep: Function,
  questionsList: CustomQuestion[],
};

export default function CustomQuestionsStep(props: Props): React.ReactElement {

  const {setActiveStep, questionsList} = props;
  const classes = userStyles();

  const customQuestions = new Array();
  const answers = Array(questionsList.length).fill('');

  const updateAnswer = (answer: string, index: number) => {
    answers[index] = answer;
  }

  questionsList.forEach((q, i) => {
    switch(q.type) {
      case CustomQuestionType.Input:
        customQuestions.push(
          <InputTypeCustomQuestion
            question={q.question}
            updateAnswers={(answer: string) => {updateAnswer(answer, i);}}
          />
        );
        return;
      case CustomQuestionType.Text:
          customQuestions.push(
            <TextareaTypeCustomQuestion
              question={q.question}
              updateAnswers={(answer: string) => {updateAnswer(answer, i);}}
            />
          );
          return;
      default:
        return;
    }
  });
  return (
    <>
      <h2 className={classes.greyText}>Extra Information about the Issue</h2>
      <div className={classes.root}>{customQuestions}</div>
      <DotMobileStepper nextEnabled={false} activeStep={2} setActiveStep={setActiveStep}/>
    </>
  );
}
