// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// TODO: create a `monorail/frontend/config/` folder to store all the feature config file
import {IssueCategory, IssueWizardPersona} from "./IssueWizardTypes";

export const ISSUE_WIZARD_QUESTIONS: IssueCategory[] = [
  {
    name: 'UI',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Network / Downloading',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Audio / Video',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Content',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Apps',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Extensions / Themes',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Webstore',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Sync',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Enterprise',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Installation',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Crashes',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Security',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Other',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'API',
    persona: IssueWizardPersona.Developer,
    enabled: true,
  },
  {
    name: 'JavaScript',
    persona: IssueWizardPersona.Developer,
    enabled: true,
  },
  {
    name: 'Developer Tools',
    persona: IssueWizardPersona.Developer,
    enabled: true,
  },
];
