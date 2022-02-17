// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.


// customized function for add additoinal data base on different categories.
export function expandDescriptions(
  category: string,
  customQuestionsAnswers: Array<string>,
  description: string,
  labels: Array<any>,
  ): {expandDescription:string, expandLabels:Array<any>} {
    let expandDescription = description;
    let expandLabels = labels;

    switch (category) {
      case 'Network / Downloading':
        expandDescription = "Example URL: " + customQuestionsAnswers[0] + "\n\n"
          + expandDescription;
        break;
      default:
        break;

    }
    return {expandDescription, expandLabels};
  }
