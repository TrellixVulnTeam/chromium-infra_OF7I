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
import Alert from '@material-ui/core/Alert';
import AttachmentUploader from './AttachmentUploader.tsx';

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

  const [additionalComments, setAdditionalComments] = React.useState('');
  const [attachments, setAttachments] = React.useState([]);
  const [answers, setAnswers] = React.useState(Array(questions.length).fill(''));
  const [hasError, setHasError] = React.useState(false);

  const updateAnswer = (answer: string, index: number) => {
    const updatedAnswers = answers;
    updatedAnswers[index] = answer;
    setAnswers(updatedAnswers);
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

  const loadFiles = () => {
    if (!attachments || attachments.length === 0) {
      return Promise.resolve([]);
    }
    const loads = attachments.map(loadLocalFile);
    return Promise.all(loads);
  }

  const loadLocalFile = (f: File) => {
    return new Promise((resolve, reject) => {
      const r = new FileReader();
      r.onloadend = () => {
        resolve({filename: f.name, content: btoa(r.result)});
      };
      r.onerror = () => {
        reject(r.error);
      };

      r.readAsBinaryString(f);
    });
  }

  const onSuccess = () => {
    //redirect to issue list
    window.location.href='/p/chromium/issues/list?q=reporter%3Ame&can=2';
  };

  const onFailure = () => {
    setHasError(true);
  }

  const onMakeIssue = () => {
    setHasError(false);
    try {
      const uploads = loadFiles();
      uploads.then((files) => {
        // TODO: add attachments to request
        onSubmit(additionalComments, answers, onSuccess, onFailure);
      }, onFailure)
    } catch (e) {
      onFailure();
    }
  }

  return (
    <>
      <h2 className={classes.greyText}>Extra Information about the Issue</h2>
      {hasError
        ? <Alert severity="error" onClose={() => {setHasError(false)}}>Something went wrong, please try again later.</Alert>
        : null
      }
      <div className={classes.root}>
        {customQuestions}

        <CustomQuestionTextarea
          question="Additional comments"
          updateAnswers={(answer: string) => setAdditionalComments(answer)}
        />

        <h3>Upload any relevant screenshots</h3>
        <AttachmentUploader files={attachments} setFiles={setAttachments}/>

      </div>
      <DotMobileStepper nextEnabled={false} activeStep={2} setActiveStep={setActiveStep} onSubmit={onMakeIssue}/>
    </>
  );
}
