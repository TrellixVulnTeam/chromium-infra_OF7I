// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-fab';
import '@material/mwc-textarea';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {SHARED_STYLES} from '../shared/shared-styles';
import {prpcClient} from '../state/prpc';
import {clearAppMessage, receiveAppMessage} from '../state/reducers/message';
import {store, thunkDispatch} from '../state/store';


@customElement('batch-repairs-view') export class BatchRepairs extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        h1 {
          margin-bottom: 0.5em;
        }

        :host {
          width: 100%;
          display: flex;
          flex-direction: column;
          overflow: hidden;
        }

        #batch-repair-form {
          display: flex;
          flex-direction: row;
          overflow: hidden;
        }

        #batch-repair-form-left {
          width: 50%;
          flex-shrink: 0;
          flex-grow: 1;
          overflow-y: scroll;
        }

        #batch-repair-form-right {
          width: 50%;
          flex-shrink: 0;
          flex-grow: 1;
          overflow-y: scroll;
        }

        .form-column {
          padding: 0 0.8em 1.8em;
        }

        .form-slot {
          display: flex;
          flex-direction: column;
        }

        .form-title {
          margin: 0 0 1em 0;
          text-align: center;
        }

        .form-subtitle {
          padding: 0.8em 8px 0.3em;
          margin-bottom: 0.5em;

          position: -webkit-sticky;
          position: sticky;
          top: 0px;
          z-index: 1;

          text-align: left;
          background-color: #fff;
        }

        #batch-search mwc-textarea{
          width: 100%;
          margin-bottom: 1em;
        }

        #form-btn-group {
          position: fixed;
          right: 2em;
          bottom: 2em;
          z-index: 5;

          display: flex;
          flex-direction: column;
          align-items: flex-end;
        }

        #form-btn-group mwc-fab {
          margin-top: 0.8em;
        }

        mwc-fab.complete-btn {
          --mdc-theme-secondary: #1E8E3E;
        }
      `,
    ]
  }

  @property({type: Object}) user;
  @property({type: Object}) devices;
  @property({type: Object}) repairRecords;
  @property({type: Object}) hostsStatus;
  @property({type: Array}) hostnames;
  @property({type: String}) hostnamesInput;
  @property({type: Boolean}) submitting = false;

  stateChanged(state) {
    this.user = state.user;
  }

  /**
   * Splits the input string into an array of hostnames with whitespaces
   * trimmed.
   */
  splitHostnames(): Array<string> {
    const hostnames: string[] =
        this.hostnamesInput.split(',').map((item: string) => item.trim());
    return hostnames;
  }

  /**
   * getDevices makes an RPC call with the entered hostnames to get
   * existing datastore device info if any.
   */
  async getDevices(hostnames: Array<string>) {
    const ids: Array<{[key: string]: string}> = hostnames.map((h: string) => {
      return {
        'hostname': h,
      };
    });
    const deviceMsg: {[key: string]: Array<{[key: string]: string}>} = {
      'ids': ids,
    };

    let response = await prpcClient
                       .call(
                           'inventory.Inventory', 'GetCrosDevices', deviceMsg,
                           this.user.authHeaders)
                       .then(
                           res => {
                             this.devices = res;
                           },
                           err => {
                             throw Error(err.description);
                           },
                       );
    return response;
  }

  /**
   * getRecords makes an RPC call with the entered hostnames to get
   * existing datastore repair records if any.
   */
  async getRecords(hostnames: Array<string>) {
    const recordMsg: {[key: string]: any} = {
      'hostnames': hostnames,
    };

    let response = await prpcClient
                       .call(
                           'inventory.Inventory', 'BatchGetManualRepairRecords',
                           recordMsg, this.user.authHeaders)
                       .then(
                           res => {
                             this.repairRecords = res;
                           },
                           err => {
                             throw Error(err.description);
                           },
                       );
    return response;
  }

  /**
   * For each hostname entered, set statuses for whether a device and/or a
   * record exists. A readiness message will be included for UI display.
   */
  processHostsStatus() {
    let hostsStatus: {[key: string]: object} = {};
    this.hostnames.forEach((host) => {
      hostsStatus[host] = {
        'deviceExists': true,
        'recordExists': false,
        'readyMsg': '',
      };
    });

    if ('failedDevices' in this.devices) {
      this.devices.failedDevices.forEach((failed) => {
        hostsStatus[failed.hostname]['deviceExists'] = false;
        hostsStatus[failed.hostname]['readyMsg'] +=
            'NO: Device does not exist. Device could not be found in Inventory v2.\n'
      });
    }

    this.repairRecords.repairRecords.forEach((record) => {
      if ('repairRecord' in record) {
        hostsStatus[record.hostname]['recordExists'] = true;
        hostsStatus[record.hostname]['readyMsg'] +=
            'NO: Open record exists. Please close the current record before creating a new one.'
      }
    });

    let hostsStatusArray: Array<{[key: string]: string}> = [];
    for (const host in hostsStatus) {
      hostsStatusArray.push({
        'hostname': host,
        'deviceExists': hostsStatus[host]['deviceExists'],
        'recordExists': hostsStatus[host]['recordExists'],
        'readyMsg': hostsStatus[host]['readyMsg'] || 'YES',
      });
    }

    this.hostsStatus = hostsStatusArray;
  }

  /**
   * Queries and checks if devices exist in Inventory v2. Also checks if any
   * host has an open repair record.
   */
  validateHosts() {
    const devicesPromise = this.getDevices(this.hostnames);
    const recordsPromise = this.getRecords(this.hostnames);

    Promise
        .all([
          devicesPromise,
          recordsPromise,
        ])
        .then(() => this.processHostsStatus());
  }

  handleSearchBar(e: InputEvent) {
    this.hostnamesInput = (<HTMLTextAreaElement>e.target!).value;
  };

  handleFormSubmission() {
    // TODO: disable button when submitting using this.submitting.
    console.log('Submit form!');
  }

  keyboardListener(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      thunkDispatch(clearAppMessage());

      e.preventDefault();
      if (this.hostnamesInput && this.user.signedIn &&
          !isEmpty(this.user.profile)) {
        this.hostnames = this.splitHostnames();
        this.validateHosts();
      } else if (!this.user.signedIn) {
        thunkDispatch(receiveAppMessage('Please sign in to continue!'));
      } else if (!this.hostnamesInput) {
        thunkDispatch(receiveAppMessage('Please enter a hostname!'));
      }
    }
  }

  /**
   * Method for displaying the repair form. Left side will show the ready status
   * of each entered hostname. Right side will show the available actions to be
   * recorded for the hosts.
   */
  displayRepairForm() {
    return html`
      <div id='batch-repair-form'>
        <div id='batch-repair-form-left' class='form-column'>
          <div class="form-slot">
            <h3 class="form-subtitle">Devices</h3>
            <vaadin-grid
              .items="${this.hostsStatus}"
              .heightByRows="${true}"
              theme="compact no-border row-stripes wrap-cell-content">
              <vaadin-grid-column width="300px" flex-grow="0" path="hostname"></vaadin-grid-column>
              <vaadin-grid-column path="readyMsg" header="Ready for Batch Process"></vaadin-grid-column>
            </vaadin-grid>
          </div>
        </div>

        <div id='batch-repair-form-right' class='form-column'>
          <div class="form-slot">
            <h3 class="form-subtitle">Repair Actions</h3>
          </div>
        </div>
        ${this.displayFormBtnGroup()}
      </div>
    `;
  }

  /**
   * Button group indicating available actions. Only action available will be to
   * open and close the records for the current batch. This will set issueFixed
   * to true and update the record, which will set the RepairState to complete
   * in the datastore.
   */
  displayFormBtnGroup() {
    return html`
      <div id="form-btn-group">
        <mwc-fab
          class="complete-btn"
          extended
          ?disabled="${this.submitting}"
          label="Create and Complete Records"
          @click="${this.handleFormSubmission}"
          @keydown="${this.keyboardListener}">
        </mwc-fab>
      </div>
    `;
  }

  render() {
    return html`
      <h1>Batch Repairs</h1>
      <div class="form-slot">
        <h3 class="form-subtitle">Search Bar</h3>
        <div id='batch-search'>
          <mwc-textarea
            outlined
            label="Enter hostnames"
            helper="Enter multiple hostnames separated by commas"
            rows=4
            ?disabled="${this.submitting}"
            value=""
            @input="${this.handleSearchBar}"
            @keydown="${this.keyboardListener}"
          ></mwc-textarea>
        </div>
      </div>
      ${this.hostsStatus ? this.displayRepairForm() : null}
    `;
  }
}
