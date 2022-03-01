 // Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {makeStyles} from '@material-ui/styles';

const userStyles = makeStyles({
  content: {
    fontSize: '15px',
    marginBottom: '15px',
  }
});

type Props = {
  issueID: string
}
export default function SubmitSuccessStep({issueID} : Props): React.ReactElement {
  const classes = userStyles();
  const issueLink = '/p/chromium/issues/detail?id=' + issueID;
  return (
    <>
      <h1>Well done!</h1>
      <div className={classes.content}>
        <div>Your issue has successfully submitted! Thank you for your contribution to maintaining Chromium.</div>
        <div>Click <a href={issueLink}>here</a> to see your filed bug.</div>
      </div>
      <img src='/static/images/dog.png'/>
    </>
  );
}
