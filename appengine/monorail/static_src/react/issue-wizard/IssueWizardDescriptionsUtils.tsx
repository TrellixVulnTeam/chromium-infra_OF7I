// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.


// customized function for add additoinal data base on different categories.
export function expandDescriptions(
  category: string,
  customQuestionsAnswers: Array<string>,
  isRegression: boolean,
  description: string,
  labels: Array<any>,
  ): {expandDescription:string, expandLabels:Array<any>} {
    let expandDescription = description;
    let expandLabels = labels;
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
          // TODO: get components value
          typeLabel = 'Type-Bug';
        } else {
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
        // TODO: get components value
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
        // TODO: get components value
        break;
      case 'API':
        expandDescription = "Does this work in other browsers? " + customQuestionsAnswers[2] + "\n\n"
          + expandDescription;
        // TODO: get components value
        break;
      default:
        break;
    }

    if (typeLabel.length > 0) {
      expandLabels.push({
        label: typeLabel
      });
    }

    return {expandDescription, expandLabels};
  }
