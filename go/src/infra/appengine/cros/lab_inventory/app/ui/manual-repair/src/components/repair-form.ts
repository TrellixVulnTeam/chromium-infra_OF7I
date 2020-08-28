// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-textfield';
import '@material/mwc-textarea';
import '@material/mwc-formfield';
import '@material/mwc-checkbox';
import '@material/mwc-fab';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {TYPE_DUT, TYPE_LABSTATION, TYPE_UNKNOWN} from '../constants';
import {store} from '../state/store';


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

      #device-info, #repair-info, #repair-actions, #additional-info {
        margin-bottom: 1.5em;
        flex: 1;
        display: flex;
        flex-direction: row;
        flex-wrap: wrap;
      }

      #device-info mwc-textfield {
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

      #repair-actions mwc-formfield {
        width: 30%;
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

  @property({type: Boolean}) submitDisabled = false;
  @property({type: Object}) form;
  @property({type: Object}) deviceInfo;
  @property({type: Object}) recordInfo;
  @property({type: Object}) user;

  stateChanged(state) {
    this.deviceInfo = state.repairRecord.deviceInfo;
    this.recordInfo = state.repairRecord.recordInfo;
    this.user = state.user;
  }

  /**
   * Checks the type of the device that is managed in state by this form. It
   * returns a type constant defined in ../constants.
   */
  checkDeviceType(): string {
    if (!this.deviceInfo) return TYPE_UNKNOWN;

    if ('dut' in this.deviceInfo.labConfig) {
      return TYPE_DUT;
    } else if ('labstation' in this.deviceInfo.labConfig) {
      return TYPE_LABSTATION;
    }
    return TYPE_UNKNOWN;
  }

  displayDeviceInfo() {
    return html`
      <div class="form-slot">
        <h3 class="form-subtitle">Device Info</h3>
        <div id="device-info">
          <mwc-textfield
            disabled
            label="Hostname"
            value="${
        this.checkDeviceType() === TYPE_DUT ?
            this.deviceInfo.labConfig.dut.hostname :
            this.deviceInfo.labConfig.labstation.hostname}"
          ></mwc-textfield>
          <mwc-textfield
            disabled
            label="Asset Tag / ID"
            value="${this.deviceInfo.labConfig.id.value}"
          ></mwc-textfield>
          <mwc-textfield
            disabled
            label="Labstation Type"
            value="Same as Labstation Board?"
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
          <mwc-textfield
            disabled
            label="Board"
            value="Redundant with Model?"
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

  displayForm() {
    return html`
      <div id='repair-form'>
        <h2 class='form-title'>Manual Repair Record for ${
        this.checkDeviceType() === TYPE_DUT ?
            this.deviceInfo.labConfig.dut.hostname :
            this.deviceInfo.labConfig.labstation.hostname}</h2>
        ${this.displayDeviceInfo()}

        <div class="form-slot">
          <h3 class="form-subtitle">Repair Record</h3>
        </div>
        <div class="form-slot">
          <h4 class="form-subtitle">Repair Info</h4>
          <div id="repair-info">
            <mwc-textfield
              label="Buganizer Bug"
            ></mwc-textfield>
            <mwc-textfield
              label="Verifier Failure Description (Servo)"
            ></mwc-textfield>
            <mwc-textfield
              label="Repair Action Failure Description (Servo)"
            ></mwc-textfield>
            <mwc-textfield
              label="Verifier Failure Description (DUT)"
            ></mwc-textfield>
            <mwc-textfield
              label="Repair Action Failure Description (DUT)"
            ></mwc-textfield>
            <mwc-textfield
              label="Diagnosis"
            ></mwc-textfield>
            <mwc-textfield
              label="Fix Procedure"
            ></mwc-textfield>
            <mwc-formfield label="Fixed Primary Issue">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Replacement Requested">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
          </div>
        </div>
        <div class="form-slot">
          <h4 class="form-subtitle">Repair Actions</h4>
          <div id="repair-actions">
            <mwc-formfield label="Read Log">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Fix Labstation">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Fix Servo">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Fix Yoshi Cable / servo_micro">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Visual Inspection">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Check / Fix Power for DUT">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Troubleshoot DUT">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
            <mwc-formfield label="Reimage / Reflash for DUT">
              <mwc-checkbox></mwc-checkbox>
            </mwc-formfield>
          </div>
        </div>
        <div class="form-slot">
          <h4 class="form-subtitle">Additional Info</h4>
          <div id="additional-info">
            <mwc-textfield
              disabled
              label="Technician LDAP"
              value=${
        this.recordInfo ? this.recordInfo.userLdap : this.user.profile.Cd}
            ></mwc-textfield>
            <mwc-textfield
              disabled
              label="Created Time"
            ></mwc-textfield>
            <mwc-textfield
              disabled
              label="Updated Time"
            ></mwc-textfield>
            <mwc-textarea
              label="Additional Comments"
              rows=6
            ></mwc-textarea>
          </div>
        </div>
        <mwc-fab extended label="Create / Update Record" @click=${
        this.handleFormSubmission}></mwc-fab>
      </div>
    `;
  }

  displayError() {
    console.log('No device!');
  }

  handleFormSubmission() {
    console.log('Nice! You clicked a button!')
  }

  render() {
    return html`
      ${isEmpty(this.deviceInfo) ? this.displayError() : this.displayForm()}
    `;
  }
}
