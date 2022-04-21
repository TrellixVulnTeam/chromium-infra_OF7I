// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {assert} from 'chai';
import {render, cleanup} from '@testing-library/react';
import CustomQuestionsStep from 'react/issue-wizard/CustomQuestionsStep.tsx';
import {CustomQuestionType} from 'react/issue-wizard/IssueWizardTypes.tsx';

describe('IssueWizard CustomQuestionsStep', () => {
  afterEach(cleanup);
  it('renders', async () => {
    render(<CustomQuestionsStep questions={[]}/>);
    const stepper = document.getElementById("mobile-stepper")

    assert.isNotNull(stepper);
  });

  it('render InputType Question', async () => {
    const questionList = [{
      type: CustomQuestionType.Input,
      question: "this is a test",
    }]
    const {container} = render(<CustomQuestionsStep questions={questionList}/>);
    const input = container.querySelector('input');
    assert.isNotNull(input);
  })

  it('render TextType Question', async () => {
    const questionList = [{
      type: CustomQuestionType.Text,
      question: "this is a test",
    }]
    const {container} = render(<CustomQuestionsStep questions={questionList}/>);
    const input = container.querySelector('textarea');
    assert.isNotNull(input);
  })
});
