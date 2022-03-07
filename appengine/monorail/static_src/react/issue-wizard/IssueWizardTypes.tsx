// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// this const is used on issue wizard lading page for render user role  options
export enum IssueWizardPersona {
  EndUser = "EndUser",
  Developer = "Developer",
  Contributor = "Contributor",
};


export const ISSUE_WIZARD_PERSONAS_DETAIL  = Object.freeze({
  [IssueWizardPersona.EndUser]: {
    name: 'End User',
    description: 'I am trying to use a website.',
  },
  [IssueWizardPersona.Developer]: {
    name: 'Web Developer',
    description: 'I am trying to build something on a website.',
  },
  [IssueWizardPersona.Contributor]: {
    name: 'Chromium Contributor',
    description: 'I know about a problem in specific tests or code.',
  }
});

export enum CustomQuestionType {
  EMPTY, // this is used to define there is no subquestions
  Text,
  Input,
  Select,
}
export type CustomQuestion = {
  type: CustomQuestionType,
  question: string,
  answerPrefix?: string,
  tip?: string,
  options?: string[],
  subQuestions?: CustomQuestion[] | null,
};

export type IssueCategory = {
  name: string,
  description: string,
  persona: IssueWizardPersona,
  enabled: boolean,
  tip?: string,
  component?: string,
  customQuestions?: CustomQuestion[],
};

export type SelectMenuOption = {
  name: string,
  description?: string,
};
