// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-button';
import '@material/mwc-dialog';
import '@vaadin/vaadin-grid/theme/lumo/vaadin-grid.js';

import {Dialog} from '@material/mwc-dialog';
import {css, customElement, html, LitElement, property} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {flattenRecordsActions} from '../../shared/helpers/repair-record-helpers';
import {SHARED_STYLES} from '../../shared/shared-styles';
import {store} from '../../state/store';

import {RepairHistoryList} from './repair-history-constants';


@customElement('repair-history-modal')
export default class RepairHistoryModal extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
      #repair-history-modal {
        --mdc-dialog-min-width: 1080px;
        --mdc-dialog-max-width: 1080px;
      }

      #show-all-btn {
        margin: 10px 0;
      }
    `,
    ];
  }

  @property({type: Object}) repairHistory;
  @property({type: Boolean}) btnDisabled = true;

  stateChanged(state) {
    this.repairHistory =
        this.parseRepairHistory(state.record.info.repairHistory);
  }

  /**
   * Parse GRPC response and display actions as a list of date, component, and
   * action string. Return all records sorted by date in descending order.
   */
  parseRepairHistory(repairHistoryRsp): RepairHistoryList {
    let repairHistoryList: RepairHistoryList = [];

    if (isEmpty(repairHistoryRsp)) {
      return repairHistoryList;
    }
    repairHistoryList = flattenRecordsActions(repairHistoryRsp);

    if (repairHistoryList.length > 5) {
      this.btnDisabled = false;
    }

    // TODO: Current GRPC returns records sorted in ascending date order. Once
    // backend is implemented properly, will remove reverse().
    return repairHistoryList.reverse();
  }

  /**
   * Select modal and set to open.
   */
  showModal() {
    const modal =
        <Dialog>this.shadowRoot!.querySelector('#repair-history-modal');
    modal.open = true;
  }

  /**
   * Return Lit HTML containing the device repair history.
   */
  render() {
    return html`
      <mwc-button
        dense
        ?disabled="${this.btnDisabled}"
        label="Show All"
        id="show-all-btn"
        @click="${this.showModal}">
      </mwc-button>
      <mwc-dialog id="repair-history-modal" heading="Repair History">
         <vaadin-grid
          .items="${this.repairHistory}"
          theme="no-border row-stripes wrap-cell-content">
          <vaadin-grid-column width="200px" flex-grow="0" path="date"></vaadin-grid-column>
          <vaadin-grid-column width="400px" path="component"></vaadin-grid-column>
          <vaadin-grid-column width="400px" path="action"></vaadin-grid-column>
        </vaadin-grid>
        <mwc-button slot="primaryAction" dialogAction="close">Close</mwc-button>
      </mwc-dialog>
    `;
  }
}
