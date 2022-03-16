// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import {makeStyles} from '@material-ui/styles';
import Dialog from '@material-ui/core/Dialog';
import DialogTitle from '@material-ui/core/DialogTitle';
import DialogContent from '@material-ui/core/DialogContent';
import DialogContentText from '@material-ui/core/DialogContentText';
import DialogActions from '@material-ui/core/DialogActions';
import Button from '@material-ui/core/Button';
import Input from '@material-ui/core/Input';

const userStyles = makeStyles({
  title: {
    backgroundColor: 'rgb(84, 110, 122)',
    color: 'white',
    font: '300 20px / 24px Roboto, RobotoDraft, Helvetica, Arial, sans-serif'
  },
  inputArea: {
    padding: '10px',
  },
  content: {
    backgroundColor: 'rgb(250, 250, 250)',
    padding: '12px 16px',
  },
  contentText: {
    fontSize: '12px',
  },
  actionsButton: {
    backgroundColor: 'rgb(250, 250, 250)',
    borderTop: '1px solid rgb(224, 224, 224)',
  }
});

type Props = {
  enable: boolean,
  setEnable: Function,
}

export function IssueWizardFeedback(props: Props): React.ReactElement {
  React.useEffect(() => {
    const script = document.createElement("script");
    script.src = 'https://support.google.com/inapp/api.js';
    script.async = true;
    document.body.appendChild(script);
  }, []);

  const classes = userStyles();
  const {enable, setEnable} = props;
  const [feedback, setFeedback] = React.useState('');

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const textInput = e.target.value;
    setFeedback(textInput);
  };

  const issueWizardFeedbackSend = () => {
    window.userfeedback.api.startFeedback({
      'productId': '5208992',  // Required.
      'bucket': 'IssueWizard',  // Optional.
      'report': {
        'description': feedback
      }
    });
    setEnable(false);
  }

  return (
      <Dialog open={enable}>
        <DialogTitle className={classes.title}>Send Feedback</DialogTitle>
          <Input
            placeholder="Have Feedback? We'd love to hear it, but please don't share sensitive informations. Have questions? Try help or support."
            disableUnderline={true}
            multiline={true}
            rows={3}
            className={classes.inputArea}
            inputProps={{maxLength: 5000}}
            onChange={handleInputChange}
          />
        <DialogContent className={classes.content}>
          <DialogContentText className={classes.contentText}>
          Some account and system information may be sent to Google. We will use it to fix problems and improve our services, subject to our
           <a href="https://myaccount.google.com/privacypolicy?hl=en&amp;authuser=0" target="_blank"> Privacy Policy </a>
           and <a href="https://www.google.com/intl/en/policies/terms?authuser=0" target="_blank"> Terms of Service </a>
           . We may email you for more information or updates.
          Go to <a href="https://support.google.com/legal/answer/3110420?hl=en&amp;authuser=0" target="_blank"> Legal Help </a>
          to ask for content changes for legal reasons.
          </DialogContentText>
        </DialogContent>
        <DialogActions className={classes.actionsButton}>
          <Button onClick={()=>{setEnable(false);}}>Cancel</Button>
          <Button onClick={issueWizardFeedbackSend}>Send</Button>
        </DialogActions>
    </Dialog>
  );
}
