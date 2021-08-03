// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { makeStyles, useTheme } from '@material-ui/core/styles';
import MobileStepper from '@material-ui/core/MobileStepper';
import Button from '@material-ui/core/Button';
import KeyboardArrowLeft from '@material-ui/icons/KeyboardArrowLeft';
import KeyboardArrowRight from '@material-ui/icons/KeyboardArrowRight';

/**
 * `<DotMobileStepper />`
 *
 * React component for rendering the linear dot stepper of the issue wizard.
 *
 *  @return ReactElement.
 */
const useStyles = makeStyles({
  root: {
    width: '100%',
    flexGrow: 1,
  },
});

export default function DotsMobileStepper(): React.ReactElement {
  const classes = useStyles();
  const theme = useTheme();
  const [activeStep, setActiveStep] = React.useState(0);

  const handleNext = () => {
    setActiveStep((prevActiveStep: number) => prevActiveStep + 1);
  };

  const handleBack = () => {
    setActiveStep((prevActiveStep: number) => prevActiveStep - 1);
  };
  return (
    <MobileStepper
      variant="dots"
      steps={2}
      position="static"
      activeStep={activeStep}
      className={classes.root}
      nextButton={
        <Button aria-label="nextButton" size="medium" onClick={handleNext} disabled={activeStep === 1}>
          Next
          <KeyboardArrowRight />
        </Button>
      }
      backButton={
        <Button aria-label="backButton" size="medium" onClick={handleBack} disabled={activeStep === 0}>
          <KeyboardArrowLeft />
          Back
        </Button>
      }
    />
  );
}