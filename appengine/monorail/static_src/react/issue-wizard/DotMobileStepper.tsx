// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {createTheme} from '@material-ui/core/styles';
import {makeStyles} from '@material-ui/styles';
import MobileStepper from '@material-ui/core/MobileStepper';
import Button from '@material-ui/core/Button';
import KeyboardArrowLeft from '@material-ui/icons/KeyboardArrowLeft';
import KeyboardArrowRight from '@material-ui/icons/KeyboardArrowRight';

const theme: Theme = createTheme();

const useStyles = makeStyles({
  root: {
    width: '100%',
    flexGrow: 1,
  },
}, {defaultTheme: theme});

type Props = {
  nextEnabled: boolean,
  activeStep: number,
  setActiveStep: Function,
  onSubmit?: Function,
}
/**
 * `<DotMobileStepper />`
 *
 * React component for rendering the linear dot stepper of the issue wizard.
 *
 *  @return ReactElement.
 */
export default function DotsMobileStepper(props: Props) : React.ReactElement {

  const {nextEnabled, activeStep, setActiveStep, onSubmit}  = props;
  const classes = useStyles();

  const handleNext = () => {
    setActiveStep((prevActiveStep: number) => prevActiveStep + 1);
  };

  const handleBack = () => {
    setActiveStep((prevActiveStep: number) => prevActiveStep - 1);
  };

  const onSubmitIssue = () => {
    if (onSubmit) {
      onSubmit();
    }
  }

  let nextButton;
  if (activeStep === 2){
    nextButton = (<Button aria-label="nextButton" size="medium" onClick={onSubmitIssue}>{'Submit'}</Button>);
  } else {
    nextButton =
      (<Button aria-label="nextButton" size="medium" onClick={handleNext} disabled={!nextEnabled}>
        {'Next'}
        <KeyboardArrowRight />
      </Button>);
  }
  return (
    <MobileStepper
      id="mobile-stepper"
      variant="dots"
      steps={3}
      position="static"
      activeStep={activeStep}
      className={classes.root}
      nextButton={nextButton}
      backButton={
        <Button aria-label="backButton" size="medium" onClick={handleBack} disabled={activeStep === 0}>
          <KeyboardArrowLeft />
          Back
        </Button>
      }
    />
  );
}
