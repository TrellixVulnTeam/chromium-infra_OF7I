// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {CustomQuestion, IssueCategory, SelectMenuOption, IssueWizardPersona} from "./IssueWizardTypes";

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
    }

    return 'Unknown / Other';

}

/**
 * Detects the user's browser.
 */
 export function getBrowser() {
  const userAgent = window.navigator.userAgent;
  if (userAgent.indexOf("Firefox") > -1) {
    return "Mozilla Firefox";
    // "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0"
  } else if (userAgent.indexOf("SamsungBrowser") > -1) {
    return "Samsung Internet";
    // "Mozilla/5.0 (Linux; Android 9; SAMSUNG SM-G955F Build/PPR1.180610.011) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/9.4 Chrome/67.0.3396.87 Mobile Safari/537.36
  } else if (userAgent.indexOf("Opera") > -1 || userAgent.indexOf("OPR") > -1) {
    return "Opera";
    // "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.102 Safari/537.36 OPR/57.0.3098.106"
  } else if (userAgent.indexOf("Trident") > -1) {
    return "Microsoft Internet Explorer";
    // "Mozilla/5.0 (Windows NT 10.0; WOW64; Trident/7.0; .NET4.0C; .NET4.0E; Zoom 3.6.0; wbx 1.0.0; rv:11.0) like Gecko"
  } else if (userAgent.indexOf("Edge") > -1) {
    return "Microsoft Edge (Legacy)";
    // "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36 Edge/16.16299"
  } else if (userAgent.indexOf("Edg") > -1) {
    return "Microsoft Edge (Chromium)";
    // Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.64
  } else if (userAgent.indexOf("Chrome") > -1) {
    return "Google Chrome or Chromium";
    // "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/66.0.3359.181 Chrome/66.0.3359.181 Safari/537.36"
  } else if (userAgent.indexOf("Safari") > -1) {
    return "Apple Safari";
    // "Mozilla/5.0 (iPhone; CPU iPhone OS 11_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/11.0 Mobile/15E148 Safari/604.1 980x1306"
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

export function buildIssueDescription(reproduceStep: string, description: string, comments: string, os: string, browser: string): string {
  const issueDescription =
    "Steps to reproduce the problem:\n" + reproduceStep.trim() + "\n"
    + "Problem Description:\n" + description.trim() + "\n"
    + "Additional Comments:\n" + comments.trim() + "\n"
    + "Browser:" + browser.trim() + "\n"
    + "OS:" + os.trim();
  return issueDescription;
}
