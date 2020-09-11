// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-textfield';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {clearAppMessage, receiveAppMessage} from '../state/reducers/message';
import {getRepairRecord} from '../state/reducers/repair-record';
import {store, thunkDispatch} from '../state/store';


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
  @property({type: Object}) deviceInfo;
  @property({type: Object}) recordInfo;
  @property({type: Object}) user;
  @property({type: Boolean}) submitting = false;

  stateChanged(state) {
    this.deviceInfo = state.record.info.deviceInfo;
    this.recordInfo = state.record.info.recordInfo;
    this.user = state.user;
  }

  render() {
    return html`
      <mwc-textfield
          id="search-field"
          outlined
          label="Enter a hostname"
          helper="Enter a device hostname to add or update repair records."
          ?disabled="${this.submitting}"
          @input="${this.handleInput}"
          @keydown="${this.keyboardListener}">
      </mwc-textfield>
    `;
  }

  handleInput(e: InputEvent) {
    this.input = (<HTMLTextAreaElement>e.target!).value;
  }

  getResultMessaging() {
    if (isEmpty(this.deviceInfo)) {
      return thunkDispatch(receiveAppMessage(`Device not found for hostname '${
          this.input}'. Please enter a hostname again.`));
    } else if (!isEmpty(this.deviceInfo) && isEmpty(this.recordInfo)) {
      return thunkDispatch(
          receiveAppMessage(`Existing record not found for hostname '${
              this.input}'. Please create a new record.`));
    } else if (!isEmpty(this.deviceInfo) && !isEmpty(this.recordInfo)) {
      return thunkDispatch(receiveAppMessage(
          `Existing repair record found for hostname '${this.input}'.`));
    }
    return null;
  }

  keyboardListener(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      thunkDispatch(clearAppMessage());

      e.preventDefault();
      if (this.input && this.user.signedIn) {
        this.submitting = true;
        thunkDispatch(getRepairRecord(this.input, this.user.authHeaders))
            .then(() => this.getResultMessaging())
            .finally(() => {
              this.submitting = false;
            });
      } else if (!this.user.signedIn) {
        thunkDispatch(receiveAppMessage('Please sign in to continue!'));
      } else if (!this.input) {
        thunkDispatch(receiveAppMessage('Please enter a hostname!'));
      }
    }
  }
}
