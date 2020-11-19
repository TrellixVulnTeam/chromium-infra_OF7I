// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-button';
import '@vaadin/vaadin-grid/theme/lumo/vaadin-grid.js';
import './repair-history-modal';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {flattenRecordsActions, getAssetTag, getHostname} from '../../shared/helpers/repair-record-helpers';
import {SHARED_STYLES} from '../../shared/shared-styles';
import {getRepairHistory} from '../../state/reducers/repair-record';
import {store, thunkDispatch} from '../../state/store';
import {RepairHistoryList} from './repair-history-constants';


@customElement('repair-history-sidebar')
export default class RepairHistorySidebar extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
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

      #repair-history-sidebar {
        margin-bottom: 1em;
      }

      #show-all-btn {
        margin: 10px 0;
      }
    `,
    ];
  }

  @property({type: Object}) user;
  @property({type: Object}) deviceInfo;
  @property({type: Object}) repairHistory;

  stateChanged(state) {
    this.deviceInfo = state.record.info.deviceInfo;
    this.user = state.user;

    this.checkHistoryAndQuery(state);
  }

  /**
   * Checks if repair history is already in state. It will pull the history if
   * the device exists in state and the history is empty.
   */
  checkHistoryAndQuery(state) {
    if (isEmpty(this.repairHistory) && !isEmpty(this.deviceInfo)) {
      const assetTag = getAssetTag(this.deviceInfo);
      const hostname = getHostname(this.deviceInfo);

      thunkDispatch(getRepairHistory(hostname, assetTag, this.user.authHeaders))
          .then(() => {
            this.repairHistory =
                this.parseRepairHistory(state.record.info.repairHistory);
          });
    }
  }

  /**
   * Parse GRPC response and display actions as a list of date, component, and
   * action string. Return 5 records sorted by date in descending order.
   */
  parseRepairHistory(repairHistoryRsp): RepairHistoryList {
    let repairHistoryList: RepairHistoryList = [];

    if (isEmpty(repairHistoryRsp)) {
      return repairHistoryList;
    }
    repairHistoryList = flattenRecordsActions(repairHistoryRsp);

    // TODO: Current GRPC returns records sorted in ascending date order. Once
    // backend is implemented properly, will remove reverse().
    return repairHistoryList.reverse().slice(0, 5);
  }

  /**
   * Return Lit HTML containing the device repair history.
   */
  displayRepairHistory() {
    return html`
      <div class="form-slot">
        <h3 class="form-subtitle">Repair History</h3>
        <div id="repair-history-sidebar">
          <vaadin-grid
            .items="${this.repairHistory}"
            .heightByRows="${true}"
            theme="compact no-border row-stripes wrap-cell-content">
            <vaadin-grid-column width="120px" flex-grow="0" path="date"></vaadin-grid-column>
            <vaadin-grid-column auto-width path="component"></vaadin-grid-column>
            <vaadin-grid-column auto-width path="action"></vaadin-grid-column>
          </vaadin-grid>
          <repair-history-modal></repair-history-modal>
        </div>
      </div>
    `;
  }

  render() {
    if (isEmpty(this.repairHistory)) {
      return html`
        <div class="form-slot">
          <h3 class="form-subtitle">Repair History</h3>
          <div id="repair-history-sidebar">
            <h4 class="form-subtitle">No history available.</h4>
          </div>
        </div>
      `;
    }

    return html`${this.displayRepairHistory()}`;
  }
}
