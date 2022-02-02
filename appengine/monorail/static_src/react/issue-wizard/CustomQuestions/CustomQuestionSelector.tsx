// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react'
import {makeStyles} from '@material-ui/styles';
import SelectMenu from '../SelectMenu.tsx';
import {GetSelectMenuOptions} from '../IssueWizardUtils.tsx';

const userStyles = makeStyles({
  inputArea: {
    width: '100%',
  },
});

type Props = {
  question: string,
  options: string[],
  updateAnswers: Function,
}

export default function CustomQuestionSelector(props: Props): React.ReactElement {

  const classes = userStyles();

  const {question, updateAnswers, options} = props;
  const [selectedOption, setSelectedOption] = React.useState(options[0]);
  const handleChange = (option: string) => {
    setSelectedOption(option);
    updateAnswers(option);
  };

  const optionList = GetSelectMenuOptions(options);
  return (
    <>
      <h3>{question}</h3>
      <SelectMenu
        optionsList={optionList}
        selectedOption={selectedOption}
        setOption={handleChange}
      />
    </>
  );
}
