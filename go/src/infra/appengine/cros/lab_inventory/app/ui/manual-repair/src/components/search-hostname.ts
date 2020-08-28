// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-textfield';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {connect} from 'pwa-helpers';

import {getRepairRecord} from '../state/actions';
import {store} from '../state/store';


@customElement('search-hostname')
export default class SearchHostname extends connect
(store)(LitElement) {
  static get styles() {
    return [css`
      :host {
        width: 100%;
        display: flex;
        justify-content: center;
        margin-bottom: 2em;
      }

      #search-field {
        width: 80%
      }
    `];
  }

  @property({type: String}) input = '';
  @property({type: Boolean}) submitDisabled = false;
  @property({type: Object}) deviceInfo;
  @property({type: Object}) recordInfo;
  @property({type: Object}) user;

  stateChanged(state) {
    this.deviceInfo = state.repairRecord.deviceInfo;
    this.recordInfo = state.repairRecord.recordInfo;
    this.user = state.user;
  }

  render() {
    return html`
      <mwc-textfield
          id="search-field"
          outlined
          label="Enter a hostname"
          helper="Enter a device hostname to add or update repair records."
          ?disabled="${this.submitDisabled}"
          @input="${this.handleInput}"
          @keydown="${this.keyboardListener}">
      </mwc-textfield>
    `;
  }

  handleInput(e: InputEvent) {
    this.input = (<HTMLTextAreaElement>e.target!).value;
  }

  keyboardListener(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      // TODO: Disabled search when submitting hostname.
      // TODO: Display errors in UI.
      e.preventDefault();
      if (this.input && this.user.signedIn) {
        store.dispatch(getRepairRecord(this.input, this.user.authHeaders));
      } else if (!this.input) {
        console.error('Please enter a hostname!');
      } else if (!this.user.signedIn) {
        console.error('Please sign in to continue!');
      }
    }
  }
}
