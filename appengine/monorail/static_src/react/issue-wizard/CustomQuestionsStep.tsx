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
import Modal from '@material-ui/core/Modal';
import Box from '@material-ui/core/Box';

const userStyles = makeStyles({
  greyText: {
    color: grey[600],
  },
  root: {
    width: '100%',
  },
  modalBox: {
    position: 'absolute',
    top: '40%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    width: 400,
    backgroundColor: 'white',
    borderRadius: '10px',
    padding: '10px',
  },
  modalTitle: {
    fontSize: '20px',
    margin: '5px 0px',
  },
  modalContext: {
    fontSize: '15px',
  },
});

type Props = {
  setActiveStep: Function,
  questions: CustomQuestion[],
  onSubmit: Function,
  setNewIssueLink: Function,
};

export default function CustomQuestionsStep(props: Props): React.ReactElement {

  const {setActiveStep, questions, onSubmit, setNewIssueLink} = props;
  const classes = userStyles();

  const customQuestions = new Array();

  const [additionalComments, setAdditionalComments] = React.useState('');
  const [attachments, setAttachments] = React.useState([]);
  const [answers, setAnswers] = React.useState(Array(questions.length).fill(''));
  const [hasError, setHasError] = React.useState(false);
  const [submitEnable, setSubmitEnable] = React.useState(true);
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const updateAnswer = (answer: string, index: number) => {
    const updatedAnswers = answers;
    updatedAnswers[index] = questions[index].answerPrefix + answer;
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

  const onSuccess = (response: Issue) => {
    //redirect to issue
    setIsSubmitting(false);
    const issueId = response.name.split('/')[3];
    const issueLink = '/p/chromium/issues/detail?id=' + issueId;
    setNewIssueLink(issueLink);
    setActiveStep(3);
  };

  const onFailure = () => {
    setIsSubmitting(false);
    setHasError(true);
  }

  const onMakeIssue = () => {
    setHasError(false);
    setIsSubmitting(true);
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
        <AttachmentUploader files={attachments} setFiles={setAttachments} setSubmitEnable={setSubmitEnable}/>

      </div>
      <DotMobileStepper nextEnabled={submitEnable} activeStep={2} setActiveStep={setActiveStep} onSubmit={onMakeIssue}/>
      <Modal open={isSubmitting} >
        <Box className={classes.modalBox}>
          <p className={classes.modalTitle}>Thanks for your support!</p>
          <p>Bug Submitting...</p>
        </Box>
      </Modal>
    </>
  );
}
