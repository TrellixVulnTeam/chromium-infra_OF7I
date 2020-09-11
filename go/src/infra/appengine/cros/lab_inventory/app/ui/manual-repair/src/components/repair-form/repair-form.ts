// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-checkbox';
import '@material/mwc-fab';
import '@material/mwc-formfield';
import '@material/mwc-select';
import '@material/mwc-textarea';
import '@material/mwc-textfield';

import {Checkbox} from '@material/mwc-checkbox';
import {css, customElement, html, LitElement, property, TemplateResult} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {createRepairRecord, getRepairRecord, updateRepairRecord} from '../../state/reducers/repair-record';
import {store, thunkDispatch} from '../../state/store';
import {TYPE_DUT, TYPE_LABSTATION, TYPE_UNKNOWN} from '../constants';

import * as repairConst from './repair-form-constants';

enum FormAction {
  CREATE,
  UPDATE,
}

@customElement('repair-form') export default class RepairForm extends connect
(store)(LitElement) {
  static get styles() {
    return [css`
      :host {
        width: 100%;
        display: flex;
        justify-content: center;
      }

      h1, h2, h3, h4 {
        margin: 0 0 1em 0;
        font-family: Roboto;
        font-weight: 500;
      }

      .form-slot {
        display: flex;
        flex-direction: row;
      }

      .form-title {
        text-align: center;
      }

      .form-subtitle {
        padding: 0.8em 0.5em 0 0;
        width: 20%;
        text-align: right;
      }

      #device-info, #repair-info, #additional-info, #repair-actions, .repair-checkboxes {
        margin-bottom: 1.5em;
        flex: 1;
        display: flex;
        flex-direction: row;
        flex-wrap: wrap;
      }

      #device-info mwc-textfield, .repair-dropdown {
        width: 45%;
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
        width: 30%;
        margin: 0 0.8em 0.5em 0;
      }

      mwc-fab {
        position: fixed;
        right: 2em;
        bottom: 2em;
      }
    `];
  }

  @property({type: Boolean}) submitting = false;
  @property({type: Object}) user;
  // Device information received from RPC.
  @property({type: Object}) deviceInfo;
  // Repair Record received from RPC.
  @property({type: Object}) recordInfo;
  // Repair Record ID as stored in datastore.
  @property({type: String}) recordId;
  // State of the form record object.
  @property({type: Object}) recordObj;
  // Action to be executed on form submission.
  @property({type: Number}) formAction;

  stateChanged(state) {
    this.deviceInfo = state.record.info.deviceInfo;
    this.recordInfo = state.record.info.recordInfo;
    this.recordId = state.record.info.recordId;
    this.user = state.user;
    this.formAction =
        isEmpty(this.recordInfo) ? FormAction.CREATE : FormAction.UPDATE;
    this.initRecordObject();
  }

  /**
   * Checks the type of the device that is managed in state by this form. It
   * returns a type constant defined in ../constants.
   */
  checkDeviceType(): string {
    if (isEmpty(this.deviceInfo)) return TYPE_UNKNOWN;

    if (TYPE_DUT in this.deviceInfo.labConfig) {
      return TYPE_DUT;
    } else if (TYPE_LABSTATION in this.deviceInfo.labConfig) {
      return TYPE_LABSTATION;
    }
    return TYPE_UNKNOWN;
  }

  /**
   * Based on device type, return hostname from deviceInfo. Returns empty string
   * if hostname is not found.
   */
  getHostname(): string {
    if (isEmpty(this.deviceInfo)) return '';

    if (this.checkDeviceType() === TYPE_DUT) {
      return this.deviceInfo.labConfig.dut.hostname;
    } else if (this.checkDeviceType() === TYPE_LABSTATION) {
      return this.deviceInfo.labConfig.labstation.hostname;
    }
    return '';
  }

  /**
   * Based on device type, return target type. Returns
   * repairConst.RepairTargetType.TYPE_DUT as default.
   */
  getRepairTargetType(): number {
    if (this.checkDeviceType() === TYPE_LABSTATION) {
      return repairConst.RepairTargetType.TYPE_LABSTATION;
    }
    return repairConst.RepairTargetType.TYPE_DUT;
  }

  /**
   * Based on the enumType, match a list of action strings to enum field.
   */
  convertActionToEnum = (actionsList: Array<string>, enumType: object) =>
      actionsList.map((action) => enumType[action]);

  /**
   * Remove all undefined fields from an object.
   */
  filterUndefined(obj: object) {
    const ret = {};
    Object.keys(obj)
        .filter((key) => obj[key] !== undefined)
        .forEach((key) => ret[key] = obj[key]);
    return ret;
  }

  /**
   * Based on whether there is an existing record or not, construct and return
   * the appropriate record object.
   */
  initRecordObject(): void {
    if (!isEmpty(this.deviceInfo) && isEmpty(this.recordInfo)) {
      this.recordObj = this.getBaseRecordObj();
    } else if (!isEmpty(this.deviceInfo) && !isEmpty(this.recordInfo)) {
      this.recordObj = this.getExistingRecordObj();
    }
  }

  /**
   * Return a base record object. Note that this object does not have the
   * timestamps as they will be created in the backend.
   */
  getBaseRecordObj() {
    return {
      hostname: this.getHostname(),
      assetTag: this.deviceInfo.labConfig.id.value,
      repairTargetType: this.getRepairTargetType(),
      repairState: repairConst.RepairState.STATE_IN_PROGRESS,
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
      userLdap: this.user.profile.$t,
      timeTaken: 0,
      additionalComments: '',
    };
  }

  /**
   * Return a record object with existing information filled in as gathered from
   * the backend datastore. If a field does not exist, replace with default
   * field.
   */
  getExistingRecordObj() {
    const baseObj = this.getBaseRecordObj();
    const existingObj = this.filterUndefined({
      hostname: this.recordInfo.hostname,
      assetTag: this.recordInfo.assetTag,
      repairTargetType: this.recordInfo.repairTargetType,
      repairState: repairConst.RepairState[this.recordInfo.repairState],
      buganizerBugUrl: this.recordInfo.buganizerBugUrl,
      chromiumBugUrl: this.recordInfo.chromiumBugUrl,
      diagnosis: this.recordInfo.diagnosis,
      repairProcedure: this.recordInfo.repairProcedure,
      labstationRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.labstationRepairActions,
          repairConst.LabstationRepairAction)),
      servoRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.servoRepairActions, repairConst.ServoRepairAction)),
      yoshiRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.yoshiRepairActions, repairConst.YoshiRepairAction)),
      chargerRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.chargerRepairActions,
          repairConst.ChargerRepairAction)),
      usbStickRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.usbStickRepairActions,
          repairConst.UsbStickRepairAction)),
      cableRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.cableRepairActions, repairConst.CableRepairAction)),
      rpmRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.rpmRepairActions, repairConst.RpmRepairAction)),
      dutRepairActions: new Set(this.convertActionToEnum(
          this.recordInfo.dutRepairActions, repairConst.DutRepairAction)),
      issueFixed: this.recordInfo.issueFixed,
      replacementRequested: this.recordInfo.replacementRequested,
      userLdap: this.recordInfo.userLdap,
      timeTaken: this.recordInfo.timeTaken,
      createdTime: this.recordInfo.createdTime,
      updatedTime: this.recordInfo.updatedTime,
      completedTime: this.recordInfo.completedTime,
      additionalComments: this.recordInfo.additionalComments || '',
    });

    return {
      ...baseObj,
      ...existingObj,
    };
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
          ?selected="${this.recordObj[stateName].has(obj.enumVal)}"
          ?activated="${this.recordObj[stateName].has(obj.enumVal)}"
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
            ?checked="${this.recordObj[stateName].has(obj.enumVal)}"
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

  /** Form input handlers */

  handleRepairDropdown(e: InputEvent) {
    const el = (<HTMLSelectElement>e.target!);
    this.recordObj[el.name] = [parseInt(el.value)];

    // Subtract old value from timeTaken.
    const prevTimeVal: string =
        this.shadowRoot!
            .querySelector(`#${el.name} > mwc-select > mwc-list-item[selected]`)
            ?.getAttribute('timeValue') ||
        '0';
    this.recordObj.timeTaken -= parseInt(prevTimeVal);

    // Add newly selected value to timeTaken.
    this.recordObj.timeTaken += parseInt(el.getAttribute('timeValue') || '0');
  };

  handleRepairCheckboxes(e: InputEvent) {
    const t: any = e.target;
    const v: number = parseInt(t.value);
    const timeVal: number = parseInt(t.getAttribute('timeValue'));
    const n: string = t.name;

    if (t.checked) {
      this.recordObj[n].add(v);
      this.recordObj.timeTaken += timeVal;
    } else {
      this.recordObj[n].delete(v);
      this.recordObj.timeTaken -= timeVal;
    };
  };

  handleCheckboxChange(field: string, e: InputEvent) {
    this.recordObj[field] = (<Checkbox>e.target!).checked;
  };

  handleIssueFixedChange(e: InputEvent) {
    this.handleCheckboxChange('issueFixed', e);
  };

  handleReplacementRequestedChange(e: InputEvent) {
    this.handleCheckboxChange('replacementRequested', e);
  };

  handleFieldChange(field: string, e: InputEvent) {
    this.recordObj[field] = (<HTMLTextAreaElement>e.target!).value;
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

  /** End form input handlers */

  /**
   * Return Lit HTML containing the device information.
   *
   * TODO: Missing Labstation Board.
   */
  displayDeviceInfo() {
    return html`
      <div class="form-slot">
        <h3 class="form-subtitle">Device Info</h3>
        <div id="device-info">
          <mwc-textfield
            disabled
            label="Hostname"
            value="${this.recordObj.hostname}"
          ></mwc-textfield>
          <mwc-textfield
            disabled
            label="Asset Tag / ID"
            value="${this.recordObj.assetTag}"
          ></mwc-textfield>
          <mwc-textfield
            disabled
            label="Labstation Type (Board)"
            value="Coming Soon"
          ></mwc-textfield>
          <mwc-textfield
            disabled
            label="Model"
            value="${this.deviceInfo.deviceConfig.id.modelId.value}"
          ></mwc-textfield>
          <mwc-textfield
            disabled
            label="Phase"
            value="${this.deviceInfo.manufacturingConfig.devicePhase}"
          ></mwc-textfield>
          ${
        this.checkDeviceType() === TYPE_DUT ?
            html`
            <mwc-textfield
            disabled
            label="Servo Asset Tag"
            value="${
                this.deviceInfo.labConfig.dut.peripherals.servo.servoSerial}"
            ></mwc-textfield>
            <mwc-textfield
              disabled
              label="Servo Type"
              value="${
                this.deviceInfo.labConfig.dut.peripherals.servo.servoType}"
            ></mwc-textfield>
            ` :
            null}
        </div>
      </div>
    `;
  }

  /**
   * Return Lit HTML of the main repair form.
   */
  displayForm() {
    return html`
      <div id='repair-form'>
        <h2 class='form-title'>Manual Repair Record for ${
        this.getHostname()}</h2>
        ${this.displayDeviceInfo()}

        <div class="form-slot">
          <h3 class="form-subtitle">Repair Record</h3>
        </div>
        <div class="form-slot">
          <h4 class="form-subtitle">Repair Actions</h4>
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
          <h4 class="form-subtitle">Other Cables Repair Actions</h4>
          ${
        this.buildRepairCheckboxes(
            repairConst.CHECKBOX_ACTIONS.cableRepairActions)}
        </div>
        <div class="form-slot">
          <h4 class="form-subtitle">DUT Repair Actions</h4>
          ${
        this.buildRepairCheckboxes(
            repairConst.CHECKBOX_ACTIONS.dutRepairActions)}
        </div>
        <div class="form-slot">
          <h4 class="form-subtitle">Repair Info</h4>
          <div id="repair-info">
            <mwc-textfield
              label="Buganizer Bug"
              ?disabled="${this.submitting}"
              value="${this.recordObj.buganizerBugUrl}"
              @input="${this.handleBuganizerChange}"
            ></mwc-textfield>
            <mwc-textfield
              label="Diagnosis"
              ?disabled="${this.submitting}"
              value="${this.recordObj.diagnosis}"
              @input="${this.handleDiagnosisChange}"
            ></mwc-textfield>
            <mwc-textfield
              label="Fix Procedure"
              ?disabled="${this.submitting}"
              value="${this.recordObj.repairProcedure}"
              @input="${this.handleProcedureChange}"
            ></mwc-textfield>
            <mwc-formfield label="Fixed Primary Issue">
              <mwc-checkbox
                id="issueFixed"
                ?disabled="${this.submitting}"
                ?checked="${this.recordObj.issueFixed}"
                @change="${this.handleIssueFixedChange}">
              </mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Replacement Requested">
              <mwc-checkbox
                ?disabled="${this.submitting}"
                ?checked="${this.recordObj.replacementRequested}"
                @change="${this.handleReplacementRequestedChange}">
              </mwc-checkbox>
            </mwc-formfield>
          </div>
        </div>
        <div class="form-slot">
          <h4 class="form-subtitle">Additional Info</h4>
          <div id="additional-info">
            <mwc-textfield
              disabled
              label="Technician LDAP"
              value="${this.recordObj.userLdap}"
            ></mwc-textfield>
            <mwc-textfield
              disabled
              label="Created Time"
              value="${this.recordObj.createdTime || ''}"
            ></mwc-textfield>
            <mwc-textfield
              disabled
              label="Updated Time"
              value="${this.recordObj.updatedTime || ''}"
            ></mwc-textfield>
            <mwc-textarea
              label="Additional Comments"
              rows=6
              ?disabled="${this.submitting}"
              value="${this.recordObj.additionalComments}"
              @input="${this.handleCommentsChange}"
            ></mwc-textarea>
          </div>
        </div>
        <mwc-fab
          extended
          ?disabled="${this.submitting}"
          label="${
        this.formAction === FormAction.CREATE ? 'Create Record' :
                                                'Update Record'}"
          @click =${this.handleFormSubmission}>
        </mwc-fab>
      </div>`;
  }

  /**
   * Takes in a set of repair actions. If set contains 0 (the NA action) and
   * other actions, remove the 0 action and return the rest. Otherwise, return
   * just a set with the 0 action.
   */
  filterCheckboxes(actionsSet: Set<number>): Set<number> {
    let resSet = new Set(actionsSet);

    if (resSet.has(0) && resSet.size > 1) {
      resSet.delete(0);
    } else if (resSet.size === 0) {
      resSet = new Set([0]);
    }

    return resSet;
  }

  /**
   * Take this.recordObj and create an object acceptable by the creation and
   * updation RPCs.
   *  1. Update repair state when applicable.
   *  2. Convert all actions sets to arrays.
   *  3. Filter any field with undefined values.
   */
  createPayloadObj() {
    let toSubmit = Object.assign({}, this.recordObj);

    if (this.recordObj.issueFixed)
      toSubmit.repairState = repairConst.RepairState.STATE_COMPLETED;

    // Filter 0 values from checkboxes sets and convert to arrays.
    toSubmit.cableRepairActions =
        Array.from(this.filterCheckboxes(this.recordObj.cableRepairActions));
    toSubmit.dutRepairActions =
        Array.from(this.filterCheckboxes(this.recordObj.dutRepairActions));

    // Convert dropdown sets to arrays.
    toSubmit.labstationRepairActions =
        Array.from(this.recordObj.labstationRepairActions);
    toSubmit.servoRepairActions = Array.from(this.recordObj.servoRepairActions);
    toSubmit.yoshiRepairActions = Array.from(this.recordObj.yoshiRepairActions);
    toSubmit.chargerRepairActions =
        Array.from(this.recordObj.chargerRepairActions);
    toSubmit.usbStickRepairActions =
        Array.from(this.recordObj.usbStickRepairActions);
    toSubmit.rpmRepairActions = Array.from(this.recordObj.rpmRepairActions);

    toSubmit = this.filterUndefined(toSubmit);

    return toSubmit;
  }

  /**
   * Form submission handler. Uses the created payload object and submits it to
   * RPC through store dispatch. The form submission will disable the form until
   * the thunk is complete.
   */
  handleFormSubmission() {
    this.submitting = true;
    const toSubmit = this.createPayloadObj();

    let submitRes: Promise<any>;
    if (this.formAction === FormAction.CREATE) {
      submitRes =
          thunkDispatch(createRepairRecord(toSubmit, this.user.authHeaders));
    } else {
      submitRes = thunkDispatch(
          updateRepairRecord(this.recordId, toSubmit, this.user.authHeaders));
    }

    submitRes
        .then(
            () => thunkDispatch(
                getRepairRecord(toSubmit.hostname, this.user.authHeaders)))
        .finally(() => {
          this.submitting = false;
        });
  }

  render() {
    return html`${isEmpty(this.deviceInfo) ? null : this.displayForm()}`;
  }
}
