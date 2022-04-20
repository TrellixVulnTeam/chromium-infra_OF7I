// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {createStyles, createTheme} from '@material-ui/core/styles';
import {makeStyles} from '@material-ui/styles';
import { TextareaAutosize } from '@material-ui/core';
import TextField from '@material-ui/core/TextField';
import {red, grey} from '@material-ui/core/colors';
import DotMobileStepper from './DotMobileStepper.tsx';
import SelectMenu from './SelectMenu.tsx';
import {OS_LIST, ISSUE_WIZARD_QUESTIONS, ISSUE_REPRODUCE_PLACEHOLDER, OS_CHANNEL_LIST} from './IssueWizardConfig.ts'
import {getTipByCategory} from './IssueWizardUtils.tsx';
import CustomQuestionSelector from './CustomQuestions/CustomQuestionSelector.tsx';

/**
 * The detail step is the second step on the dot
 * stepper. This react component provides the users with
 * specific questions about their bug to be filled out.
 */
const theme: Theme = createTheme();

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    root: {
      '& > *': {
        width: '100%',
      },
    },
    head: {
      marginTop: '1.5rem',
      fontSize: '1rem'
    },
    red: {
        color: red[600],
    },
    pageHeader: {
      color: grey[600],
      fontSize: '1.5rem',
      margin: '1rem 0',
    },
    inlineStyle: {
      display: 'inline-flex',
      alignItems: 'center',
      marginTop: '1.5rem',
    },
    inlineTitle: {
      marginRight: '10px',
      fontSize: '1rem',
    }
  }), {defaultTheme: theme}
);

type Props = {
  textValues: Object,
  setTextValues: Function,
  category: string,
  setActiveStep: Function,
  osName: string,
  setOsName: Function,
  setIsRegression: Function,
};

export default function DetailsStep(props: Props): React.ReactElement {
  const classes = useStyles();

  const {
    textValues,
    setTextValues,
    category,
    setActiveStep,
    setIsRegression
  } = props;

  const handleChange = (valueName: string) => (e: React.ChangeEvent<HTMLInputElement>) => {
    const textInput = e.target.value;
    setTextValues({...textValues, [valueName]: textInput});
  };

  const selectOs = (os: string) => {
    setTextValues({...textValues, 'osName': os});
  }

  const selectChannel = (channel: string) => {
    setTextValues({...textValues, 'channel': channel});
  }

  const tipByCategory = getTipByCategory(ISSUE_WIZARD_QUESTIONS);

  const nextEnabled =
    (textValues.oneLineSummary.trim() !== '') &&
    (textValues.stepsToReproduce.trim() !== ISSUE_REPRODUCE_PLACEHOLDER) &&
    (textValues.stepsToReproduce.trim() !== '') &&
    (textValues.describeProblem.trim() !== '');

  const getTipInnerHtml = () => {
    return {__html: tipByCategory.get(category)};
  }
  return (
    <>
        <h2 className={classes.pageHeader}>Details for problems with {category}</h2>

        <form className={classes.root} noValidate autoComplete="off">
          <div dangerouslySetInnerHTML={getTipInnerHtml()}/>

          <h3 className={classes.head}>Please confirm that the following version information is correct. <span className={classes.red}>*</span></h3>
          <div className={classes.inlineStyle}>
            <h3 className={classes.inlineTitle}>Operating System:</h3>
            <SelectMenu optionsList={OS_LIST} selectedOption={textValues.osName} setOption={selectOs} />
            <h3 className={classes.inlineTitle}>Channel:</h3>
            <SelectMenu optionsList={OS_CHANNEL_LIST} selectedOption={textValues.channel} setOption={selectChannel} />
          </div>
          <div className={classes.inlineStyle}>
            <h3 className={classes.inlineTitle}>Chrome version: </h3>
            <TextField variant="outlined" onChange={handleChange('chromeVersion')} value={textValues.chromeVersion}/>
          </div>

          <h3 className={classes.head}>Please enter a one line summary (100 character limit) <span className={classes.red}>*</span></h3>
          <TextField id="outlined-basic-1" variant="outlined" inputProps={{maxLength: 100}} onChange={handleChange('oneLineSummary')} value={textValues.oneLineSummary}/>

          <h3 className={classes.head}>Steps to reproduce problem (5000 character limit) <span className={classes.red}>*</span></h3>
          <TextareaAutosize minRows={4} id="outlined-basic-2" maxLength={5000} onChange={handleChange('stepsToReproduce')} value={textValues.stepsToReproduce}/>

          <h3 className={classes.head}>Please describe the problem (5000 character limit)<span className={classes.red}>*</span></h3>
          <TextareaAutosize minRows={3} id="outlined-basic-3" maxLength={5000} onChange={handleChange('describeProblem')} value={textValues.describeProblem}/>

          <CustomQuestionSelector
            question="Did this work before?"
            options={["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"]}
            subQuestions={null}
            updateAnswers={(answer: string) => {
              if (answer === "Yes - This is a regression") {
                setIsRegression(true);
              } else {
                setIsRegression(false);
              }
            }}
          />
        </form>
        <DotMobileStepper nextEnabled={nextEnabled} activeStep={1} setActiveStep={setActiveStep}/>
    </>
  );
}
