// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {makeStyles} from '@material-ui/styles';
import {grey} from '@material-ui/core/colors';
import DotMobileStepper from './DotMobileStepper.tsx';
import {CustomQuestion, CustomQuestionType} from './IssueWizardTypes.tsx';
import CustomQuestionInput from './CustomQuestions/CustomQuestionInput.tsx';
import CustomQuestionTextarea from './CustomQuestions/CustomQuestionTextarea.tsx';
import CustomQuestionSelector from './CustomQuestions/CustomQuestionSelector.tsx';

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
  questions: CustomQuestion[],
  onSubmit: Function,
};

export default function CustomQuestionsStep(props: Props): React.ReactElement {

  const {setActiveStep, questions, onSubmit} = props;
  const classes = userStyles();

  const customQuestions = new Array();

  // TODO: (crbug.com/monorail/10581) load all the custom questions and update answers
  const answers = Array(questions.length).fill('');

  const updateAnswer = (answer: string, index: number) => {
    answers[index] = answer;
  }

  questions.forEach((q, i) => {
    switch(q.type) {
      case CustomQuestionType.Input:
        customQuestions.push(
          <CustomQuestionInput
            question={q.question}
            updateAnswers={(answer: string) => updateAnswer(answer, i)}
          />
        );
        return;
      case CustomQuestionType.Text:
          customQuestions.push(
            <CustomQuestionTextarea
              question={q.question}
              tip={q.tip}
              updateAnswers={(answer: string) => updateAnswer(answer, i)}
            />
          );
          return;
      case CustomQuestionType.Select:
        customQuestions.push(
          <CustomQuestionSelector
            question={q.question}
            tip={q.tip}
            options={q.options}
            subQuestions={q.subQuestions}
            updateAnswers={(answer: string) => updateAnswer(answer, i)}
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
      <DotMobileStepper nextEnabled={false} activeStep={2} setActiveStep={setActiveStep} onSubmit={onSubmit}/>
    </>
  );
}
