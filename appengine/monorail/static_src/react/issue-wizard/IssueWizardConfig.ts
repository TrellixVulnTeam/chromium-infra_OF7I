// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// TODO: create a `monorail/frontend/config/` folder to store all the feature config file
import {IssueCategory, IssueWizardPersona, CustomQuestionType} from "./IssueWizardTypes.tsx";

// Customer Question convert to related labels
export const LABELS_PREFIX = 'LABELS: ';

export const ISSUE_WIZARD_QUESTIONS: IssueCategory[] = [
  {
    name: 'UI',
    description: 'Problems with the user interface (e.g. tabs, context menus, etc...)',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-UI',
    customQuestions: [],
  },
  {
    name: 'Network / Downloading',
    description: 'Problems with accessing remote content',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Internals-Network',
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What specific URL can reproduce the problem?",
        answerPrefix: "Example URL: ",
      },
    ],
  },
  {
    name: 'Audio / Video',
    description: 'Problems playing back sound or movies',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Internals-Media',
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What specific URL can reproduce the problem?",
        answerPrefix: "Example URL: ",
      },
      {
        type: CustomQuestionType.Select,
        question: "Does this feature work correctly in other browsers?",
        answerPrefix: "Does this work in other browsers?\n",
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
        type: CustomQuestionType.Text,
        question: "Please open chrome://gpu in a new Chrome tab and paste the report here.",
        answerPrefix: "Contents of chrome://gpu: \n",
      }
    ],
  },
  {
    name: 'Content',
    description: "Problems with webpages not working correctly",
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: '',
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What specific URL has a problem?",
        answerPrefix: "Example URL: ",
      },
      {
        type: CustomQuestionType.Select,
        question: "Does the problem occur on multiple sites?",
        answerPrefix: LABELS_PREFIX,
        options: ["Not sure - I don't know", "Yes - Please describe below", "No - Just that one URL"],
        subQuestions: [null,null,null],
      },
      {
        type: CustomQuestionType.Select,
        question: "Is it a problem with a plugin?",
        answerPrefix: "Is it a problem with a plugin? ",
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
        answerPrefix: "Does this work in other browsers? ",
        options: ["Not sure - I don't know", "Yes - This is just a Chromium problem", "No - I can reproduce the problem in another browser"],
        subQuestions: [
          null,
          null,
          {
            type:CustomQuestionType.Input,
            question: "Which other browsers (including versions) also have the problem?",
          }],
      },
    ],
  },
  {
    name: 'Apps',
    description: 'Issues with Webstore apps',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Platform-Apps',
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What is the link to that software in <a href='https://chrome.google.com/webstore' target='_blank'>the Chrome Webstore </a>?",
        answerPrefix: "Webstore page: ",
      }
    ],
  },
  {
    name: 'Extensions / Themes',
    description: 'Issues with Webstore extensions and themes',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Platform-Extensions',
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "What kind of software had the problem?",
        answerPrefix: LABELS_PREFIX,
        options: ["Chrome Extension - Adds new browser functionality", "Chrome Theme - Makes Chrome look awesome"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Do you know the latest version where it worked?",
          },
          null],
      },
      {
        type: CustomQuestionType.Input,
        question: "What is the link to that software in <a href='https://chrome.google.com/webstore' target='_blank'>the Chrome Webstore</a>?",
        answerPrefix: "WebStore page: ",
      },
    ],
  },
  {
    name: 'Webstore',
    description: 'Problems with the Chrome Webstore itself',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Webstore',
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "What is the URL of the Chrome WesStore page that had the problem?",
        answerPrefix: "Webstore page: ",
      },
    ],
  },
  {
    name: 'Sync',
    description: 'Problems syncing data to a Google account',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Services-Sync',
    customQuestions: [],
  },
  {
    name: 'Enterprise',
    description: 'Policy configuration and deployment issues',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Enterprise',
    customQuestions: [],
  },
  {
    name: 'Installation',
    description: 'Problem installing Chrome',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: 'Cr-Internals-Installer',
    customQuestions: [],
  },
  {
    name: 'Crashes',
    description: 'The browser closes abruptly or I see "Aw, Snap!" pages',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    tip: 'Please read the instructions on <a href="https://sites.google.com/a/chromium.org/dev/for-testers/bug-reporting-guidelines/reporting-crash-bug" target="_blank">reporting a crash issue</a>',
    component: '',
    customQuestions: [
      {
        type: CustomQuestionType.Input,
        question: "Do you have a Report ID from chrome://crashes?",
        answerPrefix: "Crashed report ID: ",
      },
      {
        type: CustomQuestionType.Select,
        question: "How severe is the crash?",
        options: ["Just one tab", "Just one plugin", "The whole browser"],
        answerPrefix: "How much crashed? ",
        subQuestions: null,
      },
      {
        type: CustomQuestionType.Select,
        question: "Is it a problem with a plugin?",
        answerPrefix: "Is it a problem with a plugin? ",
        options: ["Not sure - I don't know", "Yes - Those darn plugins", "No - It's the browser itself"],
        subQuestions: [
          null,
          {
            type:CustomQuestionType.Input,
            question: "Which plugin?",
          },
          null],
      },
    ],
    labels: ['Stability-Crash'],
  },
  {
    name: 'Security',
    description: 'Problems with the browser security',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    tip: 'Please follow the instructions for <a href="https://www.chromium.org/Home/chromium-security/reporting-security-bugs" target="_blank">reporting security issues</a>.',
    component: '',
    customQuestions: [],
    labels: ['Restrict-View-SecurityTeam'],
  },
  {
    name: 'Other',
    description: 'Something not listed here',
    persona: IssueWizardPersona.EndUser,
    enabled: true,
    component: '',
    customQuestions: [
      {
        type: CustomQuestionType.Select,
        question: "Please select a label to classify your issue:",
        answerPrefix: LABELS_PREFIX,
        options: [
          "Not sure - I don't know",
          "Type-Feature - Request for new or improved features",
          "Type-Bug-Regression - Used to work, now broken",
          "Type-Bug - Software not working correctly",
          "Cr-UI-I18N - Issue in translating UI to other languages"
        ],
        subQuestions: null,
      },
    ],
  },
  {
    name: 'API',
    description: 'Problems with a browser API',
    persona: IssueWizardPersona.Developer,
    enabled: true,
    component: '',
    customQuestions: [
      {
        type:CustomQuestionType.Select,
        question:"Which <a href='https://bugs.chromium.org/p/chromium/adminComponents' target='_blank'>component</a> does this fall under?",
        answerPrefix: LABELS_PREFIX,
        options: [
          "Not sure - I don't know",
          "Blink>Animation",
          "Blink>BackgroundSync",
          "Blink>Bindings",
          "Blink>Bluetooth",
          "Blink>Canvas",
          "Blink>Compositing",
          "Blink>CSS",
          "Blink>DataTransfer",
          "Blink>DOM",
          "Blink>Editing",
          "Blink>FileAPI",
          "Blink>Focus",
          "Blink>Fonts",
          "Blink>Forms",
          "Blink>Fullscreen",
          "Blink>GamepadAPI",
          "Blink>GetUserMedia",
          "Blink>HitTesting",
          "Blink>HTML",
          "Blink>Image",
          "Blink>Input",
          "Blink>Internals",
          "Blink>Javascript",
          "Blink>Layout",
          "Blink>Loader",
          "Blink>Location",
          "Blink>Media",
          "Blink>MediaStream",
          "Blink>MemoryAllocator",
          "Blink>Messaging",
          "Blink>Network",
          "Blink>Paint",
          "Blink>Payments",
          "Blink>PerformanceAPIs",
          "Blink>PermissionsAPI",
          "Blink>PresentationAPI",
          "Blink>PushAPI",
          "Blink>SavePage",
          "Blink>Scheduling",
          "Blink>Scroll",
          "Blink>SecurityFeature",
          "Blink>ServiceWorker",
          "Blink>Speech",
          "Blink>Storage",
          "Blink>SVG",
          "Blink>TextAutosize",
          "Blink>TextEncoding",
          "Blink>TextSelection",
          "Blink>USB",
          "Blink>Vibration",
          "Blink>ViewSource",
          "Blink>WebAudio",
          "Blink>WebComponents",
          "Blink>WebCrypto",
          "Blink>WebFonts",
          "Blink>WebGL",
          "Blink>WebGPU",
          "Blink>WebMIDI",
          "Blink>WebRTC",
          "Blink>WebShare",
          "Blink>WebVR",
          "Blink>WindowDialog",
          "Blink>Workers",
          "Blink>XML",
        ],
        subQuestions: null,
      },
      {
        type: CustomQuestionType.Select,
        question: "Does this feature work correctly in other browsers?",
        answerPrefix: "Does this work in other browsers? ",
        tip: "Tip: Use <a href='https://www.browserstack.com/' target='_blank'>browserstack.com</a> to compare behavior on different browser versions.",
        options: ["Not sure - I don't know", "Yes - This is just a Chrome problem", "No - I can reproduce the problem in another browser"],
        subQuestions: [
          null,
          null,
          {
            type:CustomQuestionType.Text,
            question: "Details of interop issue",
            tip: "Please describe what the behavior is on other browsers and link to any <a href='https://browser-issue-tracker-search.appspot.com/' target='_blank'>existing bugs.</a>",
          }
        ],
      },
    ]
  },
  {
    name: 'JavaScript',
    description: 'Problems with the JavaScript interpreter',
    persona: IssueWizardPersona.Developer,
    enabled: true,
    component: 'Cr-Blink',
    customQuestions: [],
  },
  {
    name: 'Developer Tools',
    description: 'Problems with the Developer tool chain/inspector',
    persona: IssueWizardPersona.Developer,
    enabled: true,
    component: 'Cr-Platform-DevTools',
    customQuestions: [],
  },
];

export const OS_LIST = [
  {
    name: 'Android',
    description: '',
  },
  {
  name: 'Chrome OS',
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

// possible user os channel
export const OS_CHANNEL_LIST = [
  {
    name: 'Not sure',
    description: '',
  },
  {
    name: 'Stable',
    description: '',
  },
  {
    name: 'Beta',
    description: '',
  },
  {
    name: 'Dev',
    description: '',
  },
  {
    name: 'Canary',
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

export const ISSUE_REPRODUCE_PLACEHOLDER = '1.\n2.\n3.';
