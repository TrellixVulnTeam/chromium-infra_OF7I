// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {CustomQuestion, IssueCategory, SelectMenuOption, IssueWizardPersona} from "./IssueWizardTypes";


const CHROME_VERSION_REX = /chrome\/(\d|\.)+/i;
// this function is used to get the issue list belong to different persona
// when a user group is selected a list of related issue categories will show up
export function GetCategoriesByPersona (categories: IssueCategory[]): Map<IssueWizardPersona, SelectMenuOption[]> {
  const categoriesByPersona = new Map<IssueWizardPersona, SelectMenuOption[]>();

  categories.forEach((category) => {
    if (category.enabled) {
      const currentIssuePersona = category.persona;
      const currentCategories = categoriesByPersona.get(currentIssuePersona) ?? [];
      currentCategories.push({
        name: category.name,
        description: category.description,
      });
      categoriesByPersona.set(currentIssuePersona, currentCategories);
    }
  });

  return categoriesByPersona;
}

// this function is used to get the customer questions belong to different issue category
// the customer question page will render base on these data
export function GetQuestionsByCategory(categories: IssueCategory[]): Map<string, CustomQuestion[] | null> {
  const questionsByCategory = new Map<string, CustomQuestion[] | null>();
  categories.forEach((category) => {
    questionsByCategory.set(category.name, category.customQuestions ?? null);
  })
  return questionsByCategory;
}

// this function is used to convert the options list fit for render use SelectMenu
export function GetSelectMenuOptions(optionsList: string[]): SelectMenuOption[] {
  const selectMenuOptionList = new Array<SelectMenuOption>();
  optionsList.forEach((option) => {
    selectMenuOptionList.push({name: option});
  });
  return selectMenuOptionList;
}

/**
 * Detects the user's operating system.
 */
 export function getOs() {
  const userAgent = window.navigator.userAgent,
    platform = window.navigator.platform,
    macosPlatforms = ['Macintosh', 'MacIntel', 'MacPPC', 'Mac68K'],
    windowsPlatforms = ['Win32', 'Win64', 'Windows', 'WinCE'],
    iosPlatforms = ['iPhone', 'iPad', 'iPod'];

    if (macosPlatforms.indexOf(platform) !== -1) {
      return'Mac OS';
    } else if (iosPlatforms.indexOf(platform) !== -1) {
      return 'iOS';
    } else if (windowsPlatforms.indexOf(platform) !== -1) {
      return 'Windows';
    } else if (/Android/.test(userAgent)) {
      return 'Android';
    } else if (/Linux/.test(platform)) {
      return 'Linux';
    } else if (/\bCrOS\b/.test(userAgent)) {
      return 'Chrome OS';
    }

    return 'Unknown / Other';

}

// this function is used to get the tip belong to different issue category
// used for render detail page
export function getTipByCategory(categories: IssueCategory[]): Map<string, string> {
  const tipByCategory = new Map<string, string>();
  categories.forEach((category) => {
    if (category.tip) {
      tipByCategory.set(category.name, category.tip);
    }
  })
  return tipByCategory;
}

// this function is used to get the component value for each issue category used for make issue
export function getCompValByCategory(categories: IssueCategory[]): Map<string, string> {
  const compValByCategory = new Map<string, string>();
  categories.forEach((category) => {
    if (category.component) {
      compValByCategory.set(category.name, category.component);
    }
  })
  return compValByCategory;
}

export function getLabelsByCategory(categories: IssueCategory[]): Map<string, Array<string>> {
  const labelsByCategory = new Map<string, Array<string>>();
  categories.forEach((category) => {
    if (category.labels) {
      labelsByCategory.set(category.name, category.labels);
    }
  })
  return labelsByCategory;
}


export function buildIssueDescription(
  reproduceStep: string,
  description: string,
  comments: string,
  os: string,
  chromeVersion: string,
  channel: string,
  ): string {
  const issueDescription =
    "<b>Steps to reproduce the problem:</b>\n" + reproduceStep.trim() + "\n\n"
    + "<b>Problem Description:</b>\n" + description.trim() + "\n\n"
    + "<b>Additional Comments:</b>\n" + comments.trim() + "\n\n"
    + "<b>Chrome version: </b>" + chromeVersion.trim() + " <b>Channel: </b>" + channel + "\n\n"
    + "<b>OS:</b>" + os.trim();
  return issueDescription;
}

export function buildIssueLabels(category: string, osName: string, chromeVersion: string, configLabels: Array<string> | null | undefined): Array<any> {
  const labels = [
    {label:'via-wizard-'+category},
    {label:'Pri-2'},
  ];

  const os = osName.split(' ')[0];
  if (os !== 'Unknown/Other') {
    labels.push({
      label: 'OS-'+os
    })
  }
  const mainChromeVersion = chromeVersion.split('.').length > 0 ? chromeVersion.split('.')[0] : null;
  if (mainChromeVersion !== null) {
    labels.push({
      label:'Needs-Triage-M'+mainChromeVersion
    });
  }

  if (configLabels) {
    configLabels.forEach((v) => {
      labels.push({label: v});
    })
  }
  return labels;
}


export function getChromeVersion() {
  const userAgent = window.navigator.userAgent;
  var browser= userAgent.match(CHROME_VERSION_REX) || [];
  if (browser.length > 0) {
    return browser[0].split('/')[1];
  }
  return "<Copy from:'about:version'>";
}
