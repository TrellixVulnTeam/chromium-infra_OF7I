// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Chromium project component prefix
const CR_PREFIX = 'Cr-';

// customized function for add additoinal data base on different categories.
export function expandDescriptions(
  category: string,
  customQuestionsAnswers: Array<string>,
  isRegression: boolean,
  description: string,
  labels: Array<any>,
  component?: string,
  ): {expandDescription:string, expandLabels:Array<any>, compVal:string} {
    let expandDescription = description;
    let expandLabels = labels;
    let compVal = component || '';
    let typeLabel = isRegression ? 'Type-Bug-Regression' : '';
    switch (category) {
      case 'Network / Downloading':
        expandDescription = "Example URL: " + customQuestionsAnswers[0] + "\n\n"
          + expandDescription;
        break;
      case 'Audio / Video':
        expandDescription = "Example URL: " + customQuestionsAnswers[0] + "\n\n"
          + "Does this work in other browsers? \n" + customQuestionsAnswers[1] + "\n\n"
          + "Contents of chrome://gpu: \n" + customQuestionsAnswers[2] + "\n\n"
          + expandDescription;
        break;
      case 'Content':
        expandDescription = "Example URL: " + customQuestionsAnswers[0] + "\n\n"
          + "Is it a problem with a plugin? " + customQuestionsAnswers[2] + "\n\n"
          + "Does this work in other browsers? " + customQuestionsAnswers[3] + "\n\n"
          + expandDescription;
        if (customQuestionsAnswers[1].split(' - ')[0] === 'Yes') {
          compVal = 'Cr-Blink';
          typeLabel = 'Type-Bug';
        } else {
          compVal = '';
          typeLabel = 'Type-Compat';
        }
        break;
      case 'App':
        expandDescription = "WebStore page: " + customQuestionsAnswers[0] + "\n\n"
          +expandDescription;
        break;
      case 'Extensions / Themes':
        expandDescription = "WebStore page: " + customQuestionsAnswers[1] + "\n\n"
          + expandDescription;
        if (customQuestionsAnswers[0].split(' - ')[0] === 'Chrome Extension') {
          compVal = 'Cr-Platform-Extensions';
        } else {
          compVal = 'Cr-UI-Browser-Themes';
        }
        break;
      case 'Webstore':
        expandDescription = "WebStore page: " + customQuestionsAnswers[0] + "\n\n"
          + expandDescription;
        break;
      case 'Crashes':
        expandDescription = "Crashed report ID: " + customQuestionsAnswers[0] + "\n\n"
          + "How much crashed? " + customQuestionsAnswers[1] + "\n\n"
          + "Is it a problem with a plugin? " + customQuestionsAnswers[2] + "\n\n"
          + expandDescription;
        break;
      case 'Other':
        typeLabel = "Type-Bug";
        const issueType = customQuestionsAnswers[0].split(' - ')[0];
        if (issueType !== 'Not sure'){
          typeLabel = issueType;
        }
        if (issueType === 'Cr-UI-I18N') {
          compVal = 'Cr-UI-I18N';
        }
        break;
      case 'API':
        expandDescription = "Does this work in other browsers? " + customQuestionsAnswers[2] + "\n\n"
          + expandDescription;
        compVal = customQuestionsAnswers[0];
        if (compVal === "Not sure - I don't know") {
          compVal = '';
        }
        break;
      default:
        break;
    }

    if (typeLabel.length > 0) {
      expandLabels.push({
        label: typeLabel
      });
    }

    if (compVal.length > 0) {
      if (compVal.startsWith(CR_PREFIX)) {
        compVal = compVal.substring(CR_PREFIX.length);
        compVal = compVal.replace(/-/g, '>');
      }
    }
    return {expandDescription, expandLabels, compVal};
  }
