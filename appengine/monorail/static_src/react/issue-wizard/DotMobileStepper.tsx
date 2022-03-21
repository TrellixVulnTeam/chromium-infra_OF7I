// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, {useEffect} from 'react';
import {createTheme} from '@material-ui/core/styles';
import {makeStyles} from '@material-ui/styles';
import MobileStepper from '@material-ui/core/MobileStepper';
import Button from '@material-ui/core/Button';
import Box from '@material-ui/core/Box';
import KeyboardArrowLeft from '@material-ui/icons/KeyboardArrowLeft';
import KeyboardArrowRight from '@material-ui/icons/KeyboardArrowRight';
import {ConfirmBackModal} from './ConfirmBackModal.tsx';

const theme: Theme = createTheme();

const useStyles = makeStyles({
  root: {
    width: '100%',
    flexGrow: 1,
    padding: '8px 0px',
  },
  back: {
    padding: '6px 0px',
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

  const [showConfirmModal, setShowConfirmModal] = React.useState(false);

  const handleNext = () => {
    setActiveStep(activeStep + 1);
  };

  const handleBack = () => {
    if (activeStep === 2) {
      setShowConfirmModal(true);
    } else {
      setActiveStep(activeStep - 1);
    }
  };

  const onSubmitIssue = () => {
    if (onSubmit) {
      onSubmit();
    }
  }

  const onBrowserBackButtonEvent = (e: Event) => {
    e.preventDefault();
    if (activeStep === 0) {
      window.history.back();
    } else {
      setActiveStep(activeStep-1);
    }
  }

  useEffect(() => {
    window.history.pushState(null, '', window.location.pathname);
    window.addEventListener('popstate', onBrowserBackButtonEvent);
    return () => {
      window.removeEventListener('popstate', onBrowserBackButtonEvent);
    };
  }, [activeStep]);

  let nextButton;
  if (activeStep === 2){
    nextButton = (<Button aria-label="nextButton" size="medium" onClick={onSubmitIssue} disabled={!nextEnabled}>{'Submit'}</Button>);
  } else {
    nextButton =
      (<Button aria-label="nextButton" size="medium" onClick={handleNext} disabled={!nextEnabled}>
        {'Next'}
        <KeyboardArrowRight />
      </Button>);
  }

  const backButton = activeStep === 0 ? <Box></Box> :
    (<Button aria-label="backButton" size="medium" onClick={handleBack} disabled={activeStep === 0} className={classes.back}>
      <KeyboardArrowLeft />
      Back
    </Button>);

  return (
    <>
      <MobileStepper
        id="mobile-stepper"
        variant="dots"
        steps={3}
        position="static"
        activeStep={activeStep}
        className={classes.root}
        nextButton={nextButton}
        backButton={backButton}
      />
      <ConfirmBackModal
        enable={showConfirmModal}
        setEnable={setShowConfirmModal}
        confirmBack={()=>{setActiveStep(activeStep-1);}}
      />
    </>
  );
}
