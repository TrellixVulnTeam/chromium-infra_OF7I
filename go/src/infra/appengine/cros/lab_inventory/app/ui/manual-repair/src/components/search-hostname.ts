// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-textfield';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {router} from '../shared/router';
import {SHARED_STYLES} from '../shared/shared-styles';
import {clearAppMessage, receiveAppMessage} from '../state/reducers/message';
import {clearRepairRecord, getRepairRecord} from '../state/reducers/repair-record';
import {store, thunkDispatch} from '../state/store';


@customElement('search-hostname')
export default class SearchHostname extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
      :host {
        width: 100%;
        display: flex;
        justify-content: center;
        margin-bottom: 1.5em;
        padding-top: 10px;
      }

      #search-field {
        width: 80%
      }
    `,
    ];
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
    this.checkHostnameAndQuery(state.queryStore);
  }

  render() {
    return html`
      <mwc-textfield
          id="search-field"
          outlined
          label="Enter a hostname"
          helper="Enter a device hostname to add or update repair records."
          value="${this.input}"
          ?disabled="${this.submitting}"
          @input="${this.handleInput}"
          @keydown="${this.keyboardListener}">
      </mwc-textfield>
    `;
  }

  /**
   * checkHostnameAndQuery executes query based on existing input and/or query
   * string. There are multiple scenarios:
   *
   * 1. No input, query string exists - Update input field to use hostname from
   * query string. Query on hostname. Set path to include query string.
   * 2. Input exists, query string exists, but input not same as query string -
   * Update input field to use hostname from query string. Query on hostname.
   * Set path to include query string.
   * 3. Else - Do nothing.
   */
  checkHostnameAndQuery(queryStore) {
    if (this.user.signedIn &&
        ((this.input === '' && queryStore?.hostname) ||
         (this.input !== '' && queryStore?.hostname &&
          this.input !== queryStore?.hostname))) {
      this.input = queryStore?.hostname;
      this.queryHostname(this.input);
      this.setPath();
    }
  }

  /**
   * setPath pauses the router and changes the pathname to reflect what was
   * inputted in the form of a query string.
   */
  setPath() {
    router.pause();
    if (this.input !== undefined) {
      router.navigate(`/repairs?hostname=${this.input}`);
    }
    router.resume();
  }

  /**
   * Handle input field InputEvents. Changing input field will update the query
   * param 'hostname' as well.
   */
  handleInput(e: InputEvent) {
    this.input = (<HTMLTextAreaElement>e.target!).value;
    this.setPath();
  }

  getResultMessaging() {
    if (isEmpty(this.deviceInfo)) {
      return thunkDispatch(receiveAppMessage(`Device not found for hostname '${
          this.input}'. Please enter a hostname again.`));
    } else if (!isEmpty(this.deviceInfo) && isEmpty(this.recordInfo)) {
      return thunkDispatch(receiveAppMessage(
          `Please fill the form to create a new repair record for host '${
              this.input}'.`));
    } else if (!isEmpty(this.deviceInfo) && !isEmpty(this.recordInfo)) {
      return thunkDispatch(receiveAppMessage(
          `Please continue to update existing repair record for host '${
              this.input}'.`));
    }
    return null;
  }

  /**
   * Dispatch getRepairRecord action with entered hostname. Based on store
   * state, determine application messaging. Input field is disabled while
   * dispatching.
   *
   * @param hostname Hostname of DUT.
   */
  queryHostname(hostname: string) {
    this.submitting = true;
    thunkDispatch(clearRepairRecord())
        .then(
            () =>
                thunkDispatch(getRepairRecord(hostname, this.user.authHeaders)))
        .then(() => this.getResultMessaging())
        .finally(() => {
          this.submitting = false;
        });
  }

  keyboardListener(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      thunkDispatch(clearAppMessage());

      e.preventDefault();
      if (this.input && this.user.signedIn) {
        this.queryHostname(this.input);
        this.setPath();
      } else if (!this.user.signedIn) {
        thunkDispatch(receiveAppMessage('Please sign in to continue!'));
      } else if (!this.input) {
        thunkDispatch(receiveAppMessage('Please enter a hostname!'));
      }
    }
  }
}
