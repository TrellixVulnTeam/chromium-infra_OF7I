// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-textarea';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {connect} from 'pwa-helpers';

import {getRecords} from '../state/actions';
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
      }

      #search-field {
        width: 80%
      }
    `];
  }

  @property({type: String}) input = '';
  @property({type: Boolean}) submitDisabled = false;
  @property({type: Array}) records = [];
  @property({type: Object}) user;

  stateChanged(state) {
    this.records = state.records;
    this.user = state.user;
  }

  render() {
    return html`
      <mwc-textarea
          id = "search-field"
          outlined
          rows=1
          label="Enter a hostname"
          helper="Enter a device hostname to add or update repair records."
          ?disabled="${this.submitDisabled}"
          @input="${this.handleInput}"
          @keydown="${this.keyboardListener}">
      </mwc-textarea>
    `;
  }

  handleInput(e: InputEvent) {
    this.input = (<HTMLTextAreaElement>e.target!).value;
  }

  keyboardListener(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      // TODO: Disabled search when submitting hostname.
      e.preventDefault();
      if (this.input) {
        store.dispatch(getRecords(this.input));
      } else {
        // TODO: Replace console log with error in UI.
        console.log('Please enter a hostname!');
      }
    }
  }
}
