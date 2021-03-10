// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-fab';
import '@material/mwc-textarea';

import {Checkbox} from '@material/mwc-checkbox';
import {Fab} from '@material/mwc-fab';
import {css, customElement, html, LitElement, property, TemplateResult} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {filterUndefinedKeys, filterZeroFromSet} from '../shared/helpers/common-helpers';
import {formatRecordTimestamp, getAssetTag, getHostname, getRepairTargetType} from '../shared/helpers/repair-record-helpers';
import {SHARED_STYLES} from '../shared/shared-styles';
import {defaultUfsHeaders, inventoryApiVersion, inventoryClient, ufsApiVersion, ufsClient} from '../state/prpc';
import {clearAppMessage, receiveAppMessage} from '../state/reducers/message';
import {store, thunkDispatch} from '../state/store';

import * as repairConst from './repair-form/repair-form-constants';

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

        #repair-info, #additional-info, #repair-actions, .repair-checkboxes {
          margin-bottom: 1.5em;
          flex: 1;
          display: flex;
          flex-direction: row;
          flex-wrap: wrap;
        }

        .repair-dropdown {
          width: 90%;
          margin: 0 0.8em 0.5em 0;
        }

        #repair-info mwc-textfield {
          width: 90%;
          margin-bottom: 0.5em;
        }

        #repair-info mwc-formfield {
          width: 45%;
        }

        .repair-checkboxes mwc-formfield {
          width: 30%;
        }

        .repair-dropdown mwc-select {
          width: 100%;
        }

        #additional-info mwc-textarea{
          width: 90%;
          margin-top: 1em;
        }

        #additional-info mwc-textfield {
          width: 90%;
          margin: 0 0.8em 0.5em 0;
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

        mwc-fab.warning-btn {
          --mdc-theme-secondary: #D93025;
        }
      `,
    ]
  }

  @property({type: Object}) user;
  // baseRecord is the object used to create record payload for submission.
  @property({type: Object}) baseRecord;
  @property({type: Object}) devices;
  @property({type: Object}) repairRecords;
  // hostsStatus is used to store the ready status of each hostname. If a host
  // does not match with any device in Inv v2 or has an open repair record
  // already, then the host is not ready.
  @property({type: Array}) hostnames;
  @property({type: String}) hostnamesInput;
  @property({type: Object}) hostsInfo;
  @property({type: Object}) hostsStatusArray;
  @property({type: Object}) submitResults;
  @property({type: Boolean}) submitting = false;
  @property({type: Boolean}) disableSubmit = false;

  stateChanged(state) {
    this.user = state.user;
    if (!isEmpty(this.user.profile)) {
      this.baseRecord = this.getBaseRecordObj();
    }
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
  getDevices(hostnames: Array<string>) {
    let deviceRequests: Array<any> = [];
    this.devices = [];

    hostnames.forEach(h => {
      deviceRequests.push(
          ufsClient
              .call(
                  ufsApiVersion, 'GetChromeOSDeviceData', {
                    'hostname': h,
                  },
                  Object.assign({}, defaultUfsHeaders, this.user.authHeaders))
              .then(
                  res => {
                    this.devices.push({
                      hostname: h,
                      deviceData: res,
                    });
                  },
                  err => {
                    this.devices.push({
                      hostname: h,
                      deviceData: null,
                    });
                    console.error(err.description);
                  },
                  ));
    });

    return deviceRequests;
  }

  /**
   * getRecords makes an RPC call with the entered hostnames to get
   * existing datastore repair records if any.
   */
  getRecords(hostnames: Array<string>) {
    const recordMsg: {[key: string]: any} = {
      'hostnames': hostnames,
    };

    return inventoryClient
        .call(
            inventoryApiVersion, 'BatchGetManualRepairRecords', recordMsg,
            this.user.authHeaders)
        .then(
            res => {
              this.repairRecords = res;
            },
            err => {
              console.error(err.description);
            },
        );
  }

  /**
   * batchCreateRecords makes an RPC call with the processed records to submit
   * and create repair records in the datastore. A response of results is
   * returned with error messages, if any.
   */
  async batchCreateRecords(repairRecords: Array<{[key: string]: string}>) {
    const recordsMsg: {[key: string]: Array<{[key: string]: string}>} = {
      'repair_records': repairRecords,
    };

    let response =
        await inventoryClient
            .call(
                inventoryApiVersion, 'BatchCreateManualRepairRecords',
                recordsMsg, this.user.authHeaders)
            .then(
                res => {
                  this.submitResults = this.processBatchResults(res);
                },
                err => {
                  throw Error(err.description);
                },
            );
    return response;
  }

  /**
   * For each hostname entered, set statuses for whether a device and/or a
   * record exists. A readiness message will be included for UI display. The
   * hostsInfo object contains relevant information for each host and acts as
   * the UI-side source of truth.
   */
  processHostsInfo() {
    let hostsInfo: {[key: string]: object} = {};
    this.hostnames.forEach((host) => {
      hostsInfo[host] = {
        'deviceExists': true,
        'recordExists': false,
        'readyMsg': '',
        'deviceInfo': {},
      };
    });
    this.disableSubmit = false;

    this.devices.forEach((device) => {
      hostsInfo[device.hostname]['deviceInfo'] = device.deviceData;

      if (isEmpty(device.deviceData)) {
        hostsInfo[device.hostname]['deviceExists'] = false;
        hostsInfo[device.hostname]['readyMsg'] +=
            'NO: Device does not exist. Device could not be found in UFS.\n';
      }
    });

    this.repairRecords.repairRecords.forEach((record) => {
      if ('repairRecord' in record) {
        hostsInfo[record.hostname]['recordExists'] = true;
        hostsInfo[record.hostname]['readyMsg'] +=
            'NO: Open record exists. Please close the current record before creating a new one.';
      }
    });

    let hostsStatusArray: Array<{[key: string]: string}> = [];
    for (const host in hostsInfo) {
      let status = hostsInfo[host];
      hostsStatusArray.push({
        'hostname': host,
        'readyMsg': status['readyMsg'] || 'YES',
      });

      if (!status['deviceExists'] || status['recordExists']) {
        this.disableSubmit = true;
      }
    }

    this.hostsInfo = hostsInfo;
    this.hostsStatusArray = hostsStatusArray;
  }

  /**
   * Queries and checks if devices exist in Inventory v2. Also checks if any
   * host has an open repair record.
   */
  validateHosts() {
    const devicesPromises = this.getDevices(this.hostnames);
    const recordsPromise = this.getRecords(this.hostnames);

    return Promise
        .all([
          ...devicesPromises,
          recordsPromise,
        ])
        .then(() => {
          this.processHostsInfo();
        });
  }

  /**
   * Return a base record object. Note that this object does not have the
   * timestamps as they will be created in the backend.
   */
  getBaseRecordObj() {
    return {
      hostname: getHostname({}),
      assetTag: getAssetTag({}),
      repairTargetType: getRepairTargetType({}),
      repairState: repairConst.RepairState.STATE_COMPLETED,
      buganizerBugUrl: '',
      chromiumBugUrl: '',
      diagnosis: '',
      repairProcedure: '',
      labstationRepairActions: new Set([0]),
      servoRepairActions: new Set([0]),
      yoshiRepairActions: new Set([0]),
      chargerRepairActions: new Set([0]),
      usbStickRepairActions: new Set([0]),
      cableRepairActions: new Set([0]),
      rpmRepairActions: new Set([0]),
      dutRepairActions: new Set([0]),
      issueFixed: false,
      replacementRequested: false,
      userLdap: this.user.profile.getEmail(),
      timeTaken: 0,
      additionalComments: '',
    };
  }

  /**
   * getProcessedRecord returned a sanitized base record for record submission.
   */
  getProcessedRecord() {
    let processedRecord = Object.assign({}, this.baseRecord);

    // Filter 0 values from checkboxes sets and convert to arrays.
    processedRecord.cableRepairActions =
        Array.from(filterZeroFromSet(this.baseRecord.cableRepairActions));
    processedRecord.dutRepairActions =
        Array.from(filterZeroFromSet(this.baseRecord.dutRepairActions));

    // Convert dropdown sets to arrays.
    processedRecord.labstationRepairActions =
        Array.from(this.baseRecord.labstationRepairActions);
    processedRecord.servoRepairActions =
        Array.from(this.baseRecord.servoRepairActions);
    processedRecord.yoshiRepairActions =
        Array.from(this.baseRecord.yoshiRepairActions);
    processedRecord.chargerRepairActions =
        Array.from(this.baseRecord.chargerRepairActions);
    processedRecord.usbStickRepairActions =
        Array.from(this.baseRecord.usbStickRepairActions);
    processedRecord.rpmRepairActions =
        Array.from(this.baseRecord.rpmRepairActions);

    processedRecord = filterUndefinedKeys(processedRecord);

    return processedRecord;
  }

  /**
   * createPayloadRecords uses this.baseRecord as a template for all the repair
   * records to be created. Each record will have a unique hostname and asset
   * tag attached to it.
   */
  createPayloadRecords() {
    let payloadRecords: Array<{[key: string]: string}> = [];
    const processedRecord = this.getProcessedRecord();

    for (let h in this.hostsInfo) {
      let device = this.hostsInfo[h].deviceInfo;
      let record = Object.assign({}, processedRecord);

      record.hostname = getHostname(device);
      record.assetTag = getAssetTag(device);
      record.repairTargetType = getRepairTargetType(device);

      payloadRecords.push(record);
    }

    return payloadRecords;
  }

  /**
   * processBatchResults applies a timestamp formatter to the completedTime
   * column and adds a status to the creation result.
   */
  processBatchResults(rsp) {
    const repairRecords: Array<{[key: string]: any}> = rsp.repairRecords || [];

    if (repairRecords.length > 0) {
      repairRecords.forEach(el => {
        el.completedTime = formatRecordTimestamp(el.repairRecord.completedTime);
        if (el.errorMsg) {
          el.status = 'Failed';
        } else {
          el.status = 'Success';
        }
      });
    }

    return repairRecords;
  }

  /**
   * Returns a dropdown menu of repair actions for a given component.
   *
   * @param configObj Configuration of dropdown actions to be created.
   *     See {@link repairConst.DropdownActionsConfig} for details of interface.
   * @returns         Lit HTML for the dropdown.
   */
  buildRepairDropdown(configObj: repairConst.DropdownActionsConfig) {
    const componentName: string = configObj.componentName;
    const stateName: string = configObj.stateName;
    const actionsList: Map<string, {[key: string]: number}> =
        configObj.actionList;
    const actionsListHtml: Array<TemplateResult> = [];
    const helperText: string = configObj.helperText || '';

    for (const [key, obj] of actionsList.entries()) {
      actionsListHtml.push(html`
        <mwc-list-item
          .name="${stateName}"
          timeValue="${obj.timeVal}"
          value="${obj.enumVal}"
          ?selected="${this.baseRecord[stateName].has(obj.enumVal)}"
          ?activated="${this.baseRecord[stateName].has(obj.enumVal)}"
          @click="${this.handleRepairDropdown}">
          ${key}
        </mwc-list-item>
      `)
    }

    return html`
      <div id="${stateName}" class="repair-dropdown">
        <mwc-select
          label="${componentName}"
          ?disabled="${this.submitting}"
          helper="${helperText}">
          ${actionsListHtml}
        </mwc-select>
      </div>
    `
  }

  /**
   * Returns a checkbox of repair actions for a given component.
   *
   * @param configObj Configuration of checkbox actions to be created.
   *     See {@link repairConst.CheckboxActionsConfig} for details of interface.
   * @returns         Lit HTML for the dropdown.
   */
  buildRepairCheckboxes(configObj: repairConst.CheckboxActionsConfig) {
    const stateName: string = configObj.stateName;
    const actionsList: Map<string, {[key: string]: number}> =
        configObj.actionList;
    const actionsListHtml: Array<TemplateResult> = [];

    for (const [key, obj] of actionsList.entries()) {
      actionsListHtml.push(html`
        <mwc-formfield label="${key}">
          <mwc-checkbox
            .name="${stateName}"
            timeValue="${obj.timeVal}"
            value="${obj.enumVal}"
            ?disabled="${this.submitting}"
            ?checked="${this.baseRecord[stateName].has(obj.enumVal)}"
            @change="${this.handleRepairCheckboxes}">
          </mwc-checkbox>
        </mwc-formfield>
      `)
    }

    return html`
      <div id="${stateName}" class="repair-checkboxes">
        ${actionsListHtml}
      </div>
    `
  }

  /** Form handlers */

  handleSearchBar(e: InputEvent) {
    this.hostnamesInput = (<HTMLTextAreaElement>e.target!).value;
  };

  handleRepairDropdown(e: InputEvent) {
    const el = (<HTMLSelectElement>e.target!);
    this.baseRecord[el.name] = new Set([parseInt(el.value)]);

    // Subtract old value from timeTaken.
    const prevTimeVal: string =
        this.shadowRoot!
            .querySelector(`#${el.name} > mwc-select > mwc-list-item[selected]`)
            ?.getAttribute('timeValue') ||
        '0';
    this.baseRecord.timeTaken -= parseInt(prevTimeVal);

    // Add newly selected value to timeTaken.
    this.baseRecord.timeTaken += parseInt(el.getAttribute('timeValue') || '0');
  };

  handleRepairCheckboxes(e: InputEvent) {
    const t: any = e.target;
    const v: number = parseInt(t.value);
    const timeVal: number = parseInt(t.getAttribute('timeValue'));
    const n: string = t.name;

    if (t.checked) {
      this.baseRecord[n].add(v);
      this.baseRecord.timeTaken += timeVal;
    } else {
      this.baseRecord[n].delete(v);
      this.baseRecord.timeTaken -= timeVal;
    };
  };

  handleCheckboxChange(field: string, e: InputEvent) {
    this.baseRecord[field] = (<Checkbox>e.target!).checked;
  };

  handleReplacementRequestedChange(e: InputEvent) {
    this.handleCheckboxChange('replacementRequested', e);
  };

  handleFieldChange(field: string, e: InputEvent) {
    this.baseRecord[field] = (<HTMLTextAreaElement>e.target!).value;
  };

  handleBuganizerChange(e: InputEvent) {
    this.handleFieldChange('buganizerBugUrl', e);
  };

  handleDiagnosisChange(e: InputEvent) {
    this.handleFieldChange('diagnosis', e);
  };

  handleProcedureChange(e: InputEvent) {
    this.handleFieldChange('repairProcedure', e);
  };

  handleCommentsChange(e: InputEvent) {
    this.handleFieldChange('additionalComments', e);
  };

  handleFormSubmission(e: MouseEvent) {
    if (!(<Fab>e.target!).disabled) {
      this.submitting = true;
      const toSubmit = this.createPayloadRecords();
      const recordsPromise = this.batchCreateRecords(toSubmit);
      recordsPromise.then(() => {
        this.hostsStatusArray = [];
        this.submitting = false;
      });
    } else {
      thunkDispatch(receiveAppMessage(
          'Please remove or fix the hosts that are not ready.'));
    }
  }

  searchKeyboardListener(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      thunkDispatch(clearAppMessage());
      this.submitting = true;

      e.preventDefault();
      if (this.hostnamesInput && this.user.signedIn &&
          !isEmpty(this.user.profile)) {
        this.hostnames = this.splitHostnames();
        this.validateHosts().then(() => this.submitting = false);
        this.submitResults = [];
      } else if (!this.user.signedIn) {
        thunkDispatch(receiveAppMessage('Please sign in to continue!'));
      } else if (!this.hostnamesInput) {
        thunkDispatch(receiveAppMessage('Please enter a hostname!'));
      }
    }
  }

  /** End form handlers */

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
              .items="${this.hostsStatusArray}"
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
            <div class="form-slot">
              <div id="repair-actions">
                ${
        this.buildRepairDropdown(
            repairConst.DROPDOWN_ACTIONS.labstationRepairActions)}
                ${
        this.buildRepairDropdown(
            repairConst.DROPDOWN_ACTIONS.servoRepairActions)}
                ${
        this.buildRepairDropdown(
            repairConst.DROPDOWN_ACTIONS.yoshiRepairActions)}
                ${
        this.buildRepairDropdown(
            repairConst.DROPDOWN_ACTIONS.chargerRepairActions)}
                ${
        this.buildRepairDropdown(
            repairConst.DROPDOWN_ACTIONS.usbStickRepairActions)}
                ${
        this.buildRepairDropdown(repairConst.DROPDOWN_ACTIONS.rpmRepairActions)}
              </div>
            </div>
            <div class="form-slot">
              <h3 class="form-subtitle">Other Cables Repair Actions</h4>
              ${
        this.buildRepairCheckboxes(
            repairConst.CHECKBOX_ACTIONS.cableRepairActions)}
            </div>
            <div class="form-slot">
              <h3 class="form-subtitle">DUT Repair Actions</h4>
              ${
        this.buildRepairCheckboxes(
            repairConst.CHECKBOX_ACTIONS.dutRepairActions)}
            </div>
            <div class="form-slot">
              <h3 class="form-subtitle">Repair Info</h4>
              <div id="repair-info">
                <mwc-textfield
                  label="Buganizer Bug"
                  ?disabled="${this.submitting}"
                  value="${this.baseRecord.buganizerBugUrl}"
                  @input="${this.handleBuganizerChange}"
                ></mwc-textfield>
                <mwc-textfield
                  label="Diagnosis"
                  ?disabled="${this.submitting}"
                  value="${this.baseRecord.diagnosis}"
                  @input="${this.handleDiagnosisChange}"
                ></mwc-textfield>
                <mwc-textfield
                  label="Fix Procedure"
                  ?disabled="${this.submitting}"
                  value="${this.baseRecord.repairProcedure}"
                  @input="${this.handleProcedureChange}"
                ></mwc-textfield>
                <mwc-formfield label="Replacement Requested">
                  <mwc-checkbox
                    ?disabled="${this.submitting}"
                    ?checked="${this.baseRecord.replacementRequested}"
                    @change="${this.handleReplacementRequestedChange}">
                  </mwc-checkbox>
                </mwc-formfield>
              </div>
            </div>
            <div class="form-slot">
              <h3 class="form-subtitle">Additional Info</h4>
              <div id="additional-info">
                <mwc-textfield
                  disabled
                  label="Technician LDAP"
                  value="${this.baseRecord.userLdap}"
                ></mwc-textfield>
                <mwc-textarea
                  label="Additional Comments"
                  rows=6
                  ?disabled="${this.submitting}"
                  value="${this.baseRecord.additionalComments}"
                  @input="${this.handleCommentsChange}"
                ></mwc-textarea>
              </div>
            </div>
          </div>
        </div>
        ${this.displayFormBtnGroup()}
      </div>
    `;
  }

  /**
   * Display the results of the batch creation submission.
   */
  displayResults() {
    return html`
      <div class="form-slot">
        <h3 class="form-subtitle">Submission Results</h3>
        <vaadin-grid
          .items="${this.submitResults}"
          .heightByRows="${true}"
          theme="row-stripes wrap-cell-content column-borders">
          <vaadin-grid-column width="100px" flex-grow="0" path="status" header="Status"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="hostname" header="Hostname"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="repairRecord.assetTag" header="Asset Tag"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="repairRecord.userLdap" header="User LDAP"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="completedTime" header="Completed Time"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="errorMsg" header="Error Message"></vaadin-grid-column>
        </vaadin-grid>
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
    const className: string =
        this.disableSubmit ? 'warning-btn' : 'complete-btn';
    const icon: string = this.disableSubmit ? 'error_outline' : '';
    const label: string = this.disableSubmit ? 'Not All Hosts are Valid' :
                                               'Create and Complete Records';

    return html`
      <div id="form-btn-group">
        <mwc-fab
          class="${className}"
          extended
          icon="${icon}"
          ?disabled="${this.submitting || this.disableSubmit}"
          label="${label}"
          @click="${this.handleFormSubmission}">
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
            @keydown="${this.searchKeyboardListener}"
          ></mwc-textarea>
        </div>
      </div>
      ${!isEmpty(this.hostsStatusArray) ? this.displayRepairForm() : null}
      ${!isEmpty(this.submitResults) ? this.displayResults() : null}
    `;
  }
}
