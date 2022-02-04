// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {makeStyles} from '@material-ui/styles';
import SelectMenu from '../SelectMenu.tsx';
import {CustomQuestion, CustomQuestionType} from '../IssueWizardTypes.tsx';
import CustomQuestionInput from './CustomQuestionInput.tsx';
import CustomQuestionTextarea from './CustomQuestionTextarea.tsx';
import {GetSelectMenuOptions} from '../IssueWizardUtils.tsx';

const userStyles = makeStyles({
  inputArea: {
    width: '100%',
  },
});

type Props = {
  question: string,
  options: string[],
  subQuestions: CustomQuestion[],
  updateAnswers: Function,
}

export default function CustomQuestionSelector(props: Props): React.ReactElement {

  const classes = userStyles();

  const {question, updateAnswers, options, subQuestions} = props;
  const [selectedOption, setSelectedOption] = React.useState(options[0]);

  const [subQuestion, setSubQuestion] = React.useState(subQuestions[0]);

  React.useEffect(() => {
    updateAnswers(options[0]);
  });

  const handleOptionChange = (option: string) => {
    setSelectedOption(option);
    updateAnswers(option);
    const index = options.indexOf(option);
    setSubQuestion(subQuestions[index]);
  };

  const updateSubQuestionAnswer = (answer:string) => {
    const updatedAnswer = selectedOption + ' ' + answer;
    updateAnswers(updatedAnswer);
  }
  const optionList = GetSelectMenuOptions(options);

  let renderSubQuestion = null;

  if (subQuestion != null) {
    switch(subQuestion.type) {
      case CustomQuestionType.Input:
        renderSubQuestion =
          <CustomQuestionInput
            question={subQuestion.question}
            updateAnswers={updateSubQuestionAnswer}
          />
        break;
      case CustomQuestionType.Text:
        renderSubQuestion =
            <CustomQuestionTextarea
              question={subQuestion.question}
              updateAnswers={updateSubQuestionAnswer}
            />;
        break;
      default:
        break;
    }
  }
  return (
    <>
      <h3>{question}</h3>
      <SelectMenu
        optionsList={optionList}
        selectedOption={selectedOption}
        setOption={handleOptionChange}
      />
      {renderSubQuestion}
    </>
  );
}
