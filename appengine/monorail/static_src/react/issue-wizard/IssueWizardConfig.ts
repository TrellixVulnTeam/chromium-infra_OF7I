// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// TODO: create a `monorail/frontend/config/` folder to store all the feature config file
import {CustomQuestionType} from "./IssueWizardTypes.tsx";
import {IssueCategory, IssueWizardPersona} from "./IssueWizardTypes.tsx";

export const ISSUE_WIZARD_QUESTIONS: IssueCategory[] = [
  {
    name: 'UI',
    description: 'Something is wrong with the user interface (e.g. tabs, context menus, etc...)',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What part of the browser is affected?",
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Network / Downloading',
    description: 'Problems with accessing remote content',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What specific URL can reproduce the problem?",
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Audio / Video',
    description: 'Problems playing back sound or movies',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What specific URL can reproduce the problem?",
      },
      {
        type: CustomQuestionType.Select,
        question: "Does this feature work correctly in other browsers?",
        options: ["Not sure - I don't know", "Yes - This is just a Chromium problem", "No - I can reproduce the problem in another browser"],
        subQuestions: [
          null,
          null,
          {
            type:CustomQuestionType.Input,
            question: "Which other browsers (including versions) also have the problem?",
          }],
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Content',
    description: "Web pages aren't displaying or working properly",
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What part of the browser is affected?",
      },
      {
        type: CustomQuestionType.Select,
        question: "Does the problem occur on multiple sites?",
        options: ["Not sure - I don't know", "Yes - Please describe below", "No - Just that one URL"],
        subQuestions: [null,null,null],
      },
      {
        type: CustomQuestionType.Select,
        question: "Is it a problem with a plugin?",
        options: ["Not sure - I don't know", "Yes - Those darn plugins", "No - It's the browser itself"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Which plugin?",
          },
          null],
      },
      {
        type: CustomQuestionType.Select,
        question: "Does this feature work correctly in other browsers?",
        options: ["Not sure - I don't know", "Yes - This is just a Chromium problem", "No - I can reproduce the problem in another browser"],
        subQuestions: [
          null,
          null,
          {
            type:CustomQuestionType.Input,
            question: "Which other browsers (including versions) also have the problem?",
          }],
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Apps',
    description: 'Problems with how the browser deals with apps from the webstore',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What is the name or URL of that software at https://chrome.google.com/webstore?",
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Extensions / Themes',
    description: 'Issues related to extensions and themes from the webstore',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "What kind of software had the problem?",
        options: ["Chrome Extension - Adds new browser functionality", "Chrome Theme - Makes Chrome look awesome"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      },
      {
        type: CustomQuestionType.Input,
        question: "What is the name or URL of that software at https://chrome.google.com/webstore?",
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Webstore',
    description: 'Problems with the Chrome WebStore itself',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What is the URL of the Chrome WebStore page that had the problem?",
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Sync',
    description: 'Problems syncing data',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Enterprise',
    description: 'Policy configuration and deployment issues',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Installation',
    description: 'Problem installing Chrome',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
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
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "Please select a label to classify your issue:",
        options: [
          "Not sure - I don't know",
          "Type-Feature- Request for new or improved features",
          "Type-Bug-Regression - Used to work, now broken",
          "Type-Bug - Software not working correctly",
          "Cr-UI-I18N - Issue in translating UI to other languages"
        ],
        subQuestions: [null, null, null, null, null]
      },
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
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
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
  {
    name: 'Developer Tools',
    description: 'Problems with the Developer tool chain/inspector',
    persona: IssueWizardPersona.Developer,
    enabled: true,
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "Did this work before?",
        options: ["Not applicable or don't know", "Yes - This is a regression", "No - I think it never worked"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Latest version when it worked?",
          },
          null],
      }
    ],
  },
];

export const OS_LIST = [
  {
    name: 'Android',
    description: '',
  },
  {
    name: 'iOS',
    description: '',
  },
  {
    name: 'Linux',
    description: '',
  },
  {
    name: 'Mac OS',
    description: '',
  },
  {
    name: 'Windows',
    description: '',
  },

  {
    name: 'Unknown/Other',
    description: '',
  },

]

export const BROWSER_LIST = [
  {
    name: 'Apple Safari',
    description: '',
  },
  {
    name: 'Google Chrome or Chromium',
    description: '',
  },
  {
    name: 'Mozilla Firefox',
    description: '',
  },
  {
    name: 'Microsoft Edge (Chromium)',
    description: '',
  },
  {
    name: 'Microsoft Edge (Legacy)',
    description: '',
  },
  {
    name: 'Microsoft Internet Explorer',
    description: '',
  },
  {
    name: 'Opera',
    description: '',
  },
  {
    name: 'Samsung Internet',
    description: '',
  },
  {
    name: 'Unknown / Other',
    description: '',
  },
]