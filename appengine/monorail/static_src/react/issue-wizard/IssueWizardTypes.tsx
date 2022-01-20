// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export enum IssueWizardPersona {
  EndUser = 'End User',
  Developer = 'Web Developer'
};

export enum CustomQuestionType {
  EMPTY, // this is used to define there is no subquestions
  Text,
  Input,
  Select,
}
export type CustomQuestion = {
  type: CustomQuestionType,
  question: string,
  options?: string[],
  subQuestions?: CustomQuestion[],
};

export type IssueCategory = {
  name: string,
  persona: IssueWizardPersona,
  enabled: boolean,
  customQuestions?: CustomQuestion[],
};
