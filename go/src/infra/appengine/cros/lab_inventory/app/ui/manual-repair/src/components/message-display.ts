// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-snackbar';

import {Snackbar} from '@material/mwc-snackbar';
import {customElement, html, LitElement, property} from 'lit-element';
import {connect} from 'pwa-helpers';

import {SHARED_STYLES} from '../shared/shared-styles';
import {store} from '../state/store';

const TIMEOUT_MS = 10000;

@customElement('message-display')
export default class MessageDisplay extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
    ];
  }

  @property({type: String}) applicationMessage;

  stateChanged(state) {
    this.applicationMessage = state.message.applicationMessage;

    const snackbarEl = <Snackbar>this.shadowRoot!.querySelector('#msgSnackbar');
    if (state.message.applicationMessage) {
      // Close any previous message first and display new one.
      snackbarEl?.close();
      snackbarEl?.show();
    } else {
      snackbarEl?.close();
    };
  }

  render() {
    return html`
      <mwc-snackbar
        id="msgSnackbar"
        labelText="${this.applicationMessage}"
        timeoutMs="${TIMEOUT_MS}">
        <mwc-icon-button icon="close" slot="dismiss"></mwc-icon-button>
      </mwc-snackbar>
    `;
  }
}
