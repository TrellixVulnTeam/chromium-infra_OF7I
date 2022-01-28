// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react'
import {makeStyles} from '@material-ui/styles';
import {grey} from '@material-ui/core/colors';
import DotMobileStepper from './DotMobileStepper.tsx';

const userStyles = makeStyles({
  greyText: {
    color: grey[600],
  },
});

type Props = {
  setActiveStep: Function,
};

export default function CustomQuestionsStep(props: Props): React.ReactElement {
  const {setActiveStep} = props;
  const classes = userStyles();
  return (
    <>
      <h2 className={classes.greyText}>Extra Information about the Issue</h2>
      <DotMobileStepper nextEnabled={false} activeStep={2} setActiveStep={setActiveStep} />
    </>
  );
}
