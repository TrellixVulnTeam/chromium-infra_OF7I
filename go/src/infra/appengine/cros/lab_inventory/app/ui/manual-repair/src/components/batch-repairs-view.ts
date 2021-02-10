// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {css, customElement, html, LitElement, property} from 'lit-element';
import {connect} from 'pwa-helpers';

import {SHARED_STYLES} from '../shared/shared-styles';
import {store} from '../state/store';


@customElement('batch-repairs-view') export class BatchRepairs extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        h1 {
          margin-bottom: 0.5em;
        }
      `,
    ]
  }

  @property({type: Object}) user;

  constructor() {
    super();
  }

  stateChanged(state) {
    this.user = state.user;
  }

  render() {
    return html`
      <h1>Batch Repairs</h1>
    `;
  }
}
