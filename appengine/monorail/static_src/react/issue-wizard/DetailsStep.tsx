// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {createStyles, createTheme} from '@material-ui/core/styles';
import {makeStyles} from '@material-ui/styles';
import TextField from '@material-ui/core/TextField';
import {red, grey} from '@material-ui/core/colors';
import DotMobileStepper from './DotMobileStepper.tsx';
import SelectMenu from './SelectMenu.tsx';
import {OS_LIST, BROWSER_LIST, ISSUE_WIZARD_QUESTIONS} from './IssueWizardConfig.ts'
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
        margin: theme.spacing(1),
        width: '100%',
      },
    },
    head: {
        marginTop: '25px',
    },
    red: {
        color: red[600],
    },
    grey: {
        color: grey[600],
    },
  }), {defaultTheme: theme}
);

type Props = {
  textValues: Object,
  setTextValues: Function,
  category: string,
  setActiveStep: Function,
  osName: string,
  setOsName: Function,
  browserName: string
  setBrowserName: Function,
  setIsRegression: Function,
};

export default function DetailsStep(props: Props): React.ReactElement {
  const classes = useStyles();

  const {
    textValues,
    setTextValues,
    category,
    setActiveStep,
    osName,
    setOsName,
    browserName,
    setBrowserName,
    setIsRegression
  } = props;

  const handleChange = (valueName: string) => (e: React.ChangeEvent<HTMLInputElement>) => {
    const textInput = e.target.value;
    setTextValues({...textValues, [valueName]: textInput});
  };
  const tipByCategory = getTipByCategory(ISSUE_WIZARD_QUESTIONS);

  const nextEnabled =
    (textValues.oneLineSummary.trim() !== '') &&
    (textValues.stepsToReproduce.trim() !== '') &&
    (textValues.describeProblem.trim() !== '');

  const getTipInnerHtml = () => {
    return {__html: tipByCategory.get(category)};
  }

  return (
    <>
        <h2 className={classes.grey}>Details for problems with {category}</h2>

        <form className={classes.root} noValidate autoComplete="off">
          <div dangerouslySetInnerHTML={getTipInnerHtml()}/>

          <h3 className={classes.head}>Please confirm that the following version information is correct. <span className={classes.red}>*</span></h3>
          <h3>Operating System:</h3>
          <SelectMenu optionsList={OS_LIST} selectedOption={osName} setOption={setOsName} />
          <h3>Browser:</h3>
          <SelectMenu optionsList={BROWSER_LIST} selectedOption={browserName} setOption={setBrowserName} />

            <h3 className={classes.head}>Please enter a one line summary <span className={classes.red}>*</span></h3>
            <TextField id="outlined-basic-1" variant="outlined" inputProps={{maxLength: 100}} onChange={handleChange('oneLineSummary')}/>

          <h3 className={classes.head}>Steps to reproduce problem <span className={classes.red}>*</span></h3>
          <TextField multiline rows={4} id="outlined-basic-2" variant="outlined" onChange={handleChange('stepsToReproduce')}/>

          <h3 className={classes.head}>Please describe the problem <span className={classes.red}>*</span></h3>
          <TextField multiline rows={3} id="outlined-basic-3" variant="outlined" onChange={handleChange('describeProblem')}/>

          <h3 className={classes.head}>Upload any relevant screenshots</h3>
          <input type="file" accept="image/*" multiple />

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
