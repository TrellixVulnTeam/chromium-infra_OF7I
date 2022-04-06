// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LABELS_PREFIX} from "./IssueWizardConfig.ts";

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
    let expandDescription = "";
    let expandLabels = labels;
    let compVal = component || '';
    let typeLabel = isRegression ? 'Type-Bug-Regression' : 'Type-Bug';

    customQuestionsAnswers.forEach((ans) => {
      if (ans.startsWith(LABELS_PREFIX)) {
        const currentAnswer = ans.substring(LABELS_PREFIX.length);
        switch (category) {
          case 'Content':
            if (currentAnswer.split(' - ')[0] === 'Yes') {
              compVal = 'Cr-Blink';
              typeLabel = 'Type-Bug';
            } else {
              compVal = '';
              typeLabel = 'Type-Compat';
            }
            break;
          case 'Extensions / Themes':
            if (currentAnswer.split(' - ')[0] === 'Chrome Extension') {
              compVal = 'Cr-Platform-Extensions';
            } else {
              compVal = 'Cr-UI-Browser-Themes';
            }
            break;
          case 'Security':
            if (typeLabel === '') {
              typeLabel = 'Type-Bug-Security';
            }
          case 'Other':
            typeLabel = "Type-Bug";
            const issueType = currentAnswer.split(' - ')[0];
            if (issueType !== 'Not sure'){
              typeLabel = issueType;
            }
            if (issueType === 'Cr-UI-I18N') {
              compVal = 'Cr-UI-I18N';
            }
            break;
          case 'API':
            compVal = currentAnswer;
            if (compVal === "Not sure - I don't know") {
              compVal = '';
            }
            break;
        }
      } else {
        expandDescription = expandDescription + ans + "\n\n";
      }
    });

    expandDescription = expandDescription + description;

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
