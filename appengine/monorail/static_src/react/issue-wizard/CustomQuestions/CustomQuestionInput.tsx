// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react'
import OutlinedInput from "@material-ui/core/OutlinedInput";
import {makeStyles} from '@material-ui/styles';

const userStyles = makeStyles({
  head: {
    marginTop: '1.5rem',
    fontSize: '1rem'
  },
  inputArea: {
    width: '100%',
  },
});

type Props = {
  question: string,
  updateAnswers: Function,
}

export default function CustomQuestionInput(props: Props): React.ReactElement {

  const classes = userStyles();

  const {question, updateAnswers} = props;
  const [answer, setAnswer] = React.useState('');
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setAnswer(e.target.value);
    updateAnswers(e.target.value);
  };
  const getInnerHtml = ()=> {
    return {__html: question};
  }
  return (
    <>
      <h3 dangerouslySetInnerHTML={getInnerHtml()} className={classes.head}/>
      <OutlinedInput
        value={answer}
        onChange={handleChange}
        className={classes.inputArea}
        inputProps={{ maxLength: 1000 }}
      />
    </>
  );
}
