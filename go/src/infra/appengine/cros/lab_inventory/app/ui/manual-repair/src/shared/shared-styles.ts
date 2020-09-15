// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {css} from 'lit-element';

export const SHARED_STYLES = css`
  h1, h2, h3, h4 {
    margin: 0;
    font-family: Google Sans;
    font-weight: 500;
  }

  /**
   * MWC Styling
   */
  mwc-button,
  mwc-checkbox,
  mwc-drawer,
  mwc-fab,
  mwc-formfield,
  mwc-icon-button,
  mwc-list,
  mwc-menu,
  mwc-select,
  mwc-snackbar,
  mwc-textarea,
  mwc-textfield,
  mwc-top-app-bar-fixed {
    --mdc-theme-primary: #1A73E8;
    --mdc-typography-font-family: Roboto;
  }

  mwc-checkbox,
  mwc-fab {
    --mdc-theme-secondary: #1A73E8;
  }
`;
