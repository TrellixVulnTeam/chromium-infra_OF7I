// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// TODO: create a `monorail/frontend/config/` folder to store all the feature config file
import {IssueCategory, IssueWizardPersona} from "./IssueWizardTypes.tsx";

export const ISSUE_WIZARD_QUESTIONS: IssueCategory[] = [
  {
    name: 'UI',
    description: 'Something is wrong with the user interface (e.g. tabs, context menus, etc...)',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Network / Downloading',
    description: 'Problems with accessing remote content',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Audio / Video',
    description: 'Problems playing back sound or movies',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Content',
    description: "Web pages aren't displaying or working properly",
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Apps',
    description: 'Problems with how the browser deals with apps from the webstore',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Extensions / Themes',
    description: 'Issues related to extensions and themes from the webstore',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Webstore',
    description: 'Problems with the Chrome WebStore itself',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Sync',
    description: 'Problems syncing data',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Enterprise',
    description: 'Policy configuration and deployment issues',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Installation',
    description: 'Problem installing Chrome',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Crashes',
    description: 'The browser closes abruptly or I see "Aw, Snap!" pages',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Security',
    description: 'Issues related to the security of the browser',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'Other',
    description: 'Something not listed here',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
  },
  {
    name: 'API',
    description: 'Problems with a browser API',
    persona: IssueWizardPersona.Developer,
    enabled: true,
  },
  {
    name: 'JavaScript',
    description: 'Problems with the JavaScript interpreter',
    persona: IssueWizardPersona.Developer,
    enabled: true,
  },
  {
    name: 'Developer Tools',
    description: 'Problems with the Developer tool chain/inspector',
    persona: IssueWizardPersona.Developer,
    enabled: true,
  },
];
