// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@vaadin/vaadin-grid/theme/lumo/vaadin-grid.js';
import '@material/mwc-button';
import '@material/mwc-checkbox';
import '@material/mwc-icon-button';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {render} from 'lit-html';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {getManualRepairBaseUrl} from '../shared/helpers/environment-helpers';
import {formatRecordTimestamp} from '../shared/helpers/repair-record-helpers';
import {SHARED_STYLES} from '../shared/shared-styles';
import {prpcClient} from '../state/prpc';
import {store} from '../state/store';


@customElement('dashboard-view') export class DashboardView extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        h1 {
          margin-bottom: 0.5em;
        }

        h2 {
          margin-bottom: 0.3em;
        }

        #dashboard-nav {
          display: flex;
          justify-content: center;
          margin-top: 10px;
        }

        #dashboard-nav mwc-button {
          margin: 0 5px;
        }

        #dashboard-filters {
          position: relative;
          margin-bottom: 15px;
          padding: 0 15px;
        }

        #user-ldap-filters, #repair-state-filters {
          display: flex;
          flex-direction: row;
        }

        #user-ldap-filters mwc-textfield {
          width: 30%;
        }

        .filter-subtitle {
          display: flex;
          align-items: center;
          width: 15%;
        }

        #filter-btn {
          position: absolute;
          right: 10px;
          bottom: 15px;
        }

        vaadin-grid {
          height: calc(100vh - 400px);
        }
      `,
    ]
  }

  @property({type: Object}) user;
  @property({type: Object}) data;
  @property({type: Number}) limit = 25;
  @property({type: Number}) currOffset = 0;
  @property({type: Object})
  dashboardFilters = {
    repairState: {
      inProgress: true,
      completed: true,
    },
    userLdap: '',
  };

  constructor() {
    super();
    this.recordUrlRenderer = this.recordUrlRenderer.bind(this);
  }

  stateChanged(state) {
    this.user = state.user;

    // Only attempt to get records when user profile is loaded.
    if (!isEmpty(this.user.profile)) {
      this.dashboardFilters.userLdap = this.user.profile.getEmail();
      this.getRepairRecords();
    }
  }

  /**
   * getRepairRecords applies user selected filters and makes an RPC call to get
   * existing datastore repair records.
   */
  async getRepairRecords() {
    const recordMsg: {[key: string]: any} = {
      'limit': this.limit,
      'offset': this.currOffset,
    };

    if (this.dashboardFilters.userLdap) {
      recordMsg['user_ldap'] = this.dashboardFilters.userLdap;
    }

    const {inProgress, completed} = this.dashboardFilters.repairState;
    if (inProgress && !completed) {
      recordMsg['repair_state'] = 'STATE_IN_PROGRESS';
    } else if (completed && !inProgress) {
      recordMsg['repair_state'] = 'STATE_COMPLETED';
    }

    let response = await prpcClient
                       .call(
                           'inventory.Inventory', 'ListManualRepairRecords',
                           recordMsg, this.user.authHeaders)
                       .then(
                           res => {
                             this.data = this.processRepairRecords(res);
                           },
                           err => {
                             throw Error(err.description);
                           },
                       );
    return response;
  }

  /**
   * processRepairRecords applies a timestamp formatter to the updatedTime
   * column and creates a link recordUrl to the host of the record.
   */
  processRepairRecords(rsp) {
    const repairRecords: Array<{[key: string]: any}> = rsp.repairRecords || [];

    if (repairRecords.length > 0) {
      repairRecords.forEach(el => {
        el.updatedTime = formatRecordTimestamp(el.updatedTime);
        el.recordUrl =
            getManualRepairBaseUrl() + '#/repairs?hostname=' + el.hostname;
      });
    }

    return repairRecords;
  }

  /**
   * prevPage decreases the offset used for the RPC get call and makes the RPC
   * call to get the paginated records.
   */
  prevPage() {
    if (this.currOffset > 0) {
      this.currOffset -= this.limit;
      this.getRepairRecords();
    }
  }

  /**
   * prevPage increases the offset used for the RPC get call and makes the RPC
   * call to get the paginated records.
   */
  nextPage() {
    // TODO: Can return page or remaining records from RPC to optimize this and
    // show page numbers in UI.
    if (this.data.length == this.limit) {
      this.currOffset += this.limit;
      this.getRepairRecords();
    }
  }

  /**
   * Event handlers for user dashboard filters.
   */
  handleLdapChange(e: InputEvent) {
    this.dashboardFilters.userLdap = (<HTMLTextAreaElement>e.target!).value;
  };

  handleRepairStateCheckboxes(e: InputEvent) {
    const t: any = e.target;
    this.dashboardFilters.repairState[t.name] = t.checked;
  }

  handleFilterBtn() {
    this.getRepairRecords();
  }

  handleEnterFilter(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      this.getRepairRecords();
    }
  }

  /**
   * recordUrlRenderer renders the link icon for each row of the dashboard grid.
   */
  recordUrlRenderer(root, _, rowData) {
    render(
        html`
        <mwc-icon-button
          .url="${rowData.item.recordUrl}"
          icon="link"
          @click="${this.handleIconClick}">
        </mwc-icon-button>
      `,
        root);
  }

  handleIconClick(e: Event) {
    const t: any = e.target;
    window.location.href = t.url;
  }

  render() {
    return html`
      <h1>Dashboard</h1>
      <div id="dashboard-filters">
        <div id="user-ldap-filters">
          <h3 class="filter-subtitle">User LDAP</h3>
          <mwc-textfield
            icon="person"
            helper="Enter a user LDAP"
            value="${this.dashboardFilters.userLdap}"
            @keydown="${this.handleEnterFilter}"
            @input="${this.handleLdapChange}">
          </mwc-textfield>
        </div>
        <div id="repair-state-filters">
          <h3 class="filter-subtitle">Repair State</h3>
          <mwc-formfield label="In Progress">
            <mwc-checkbox
              .name="${'inProgress'}"
              ?checked="${this.dashboardFilters.repairState.inProgress}"
              @change="${this.handleRepairStateCheckboxes}">
            </mwc-checkbox>
          </mwc-formfield>
          <mwc-formfield label="Completed">
            <mwc-checkbox
              .name="${'completed'}"
              ?checked="${this.dashboardFilters.repairState.completed}"
              @change="${this.handleRepairStateCheckboxes}">
            </mwc-checkbox>
          </mwc-formfield>
        </div>
        <mwc-button
          id="filter-btn"
          icon="filter_alt"
          label="Filter"
          @click="${this.handleFilterBtn}"
          raised>
        </mwc-button>
      </div>
      <div id="dashboard">
        <vaadin-grid
          .items="${this.data}"
          theme="row-stripes wrap-cell-content column-borders">
          <vaadin-grid-column width="200px" flex-grow="0" path="updatedTime" header="Last Updated"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="hostname"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="assetTag" header="Asset Tag"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="repairState" header="Record State"></vaadin-grid-column>
          <vaadin-grid-column auto-width path="userLdap" header="User LDAP"></vaadin-grid-column>
          <vaadin-grid-column
            auto-width
            path="recordUrl"
            header="View Host"
            .renderer="${this.recordUrlRenderer}">
          </vaadin-grid-column>
        </vaadin-grid>
        <div id="dashboard-nav">
          <mwc-button
            label="Prev"
            icon="keyboard_arrow_left"
            @click="${this.prevPage}">
          </mwc-button>
          <mwc-button
            label="Next"
            icon="keyboard_arrow_right"
            trailingIcon
            @click="${this.nextPage}">
          </mwc-button>
        </div>
      </div>
    `;
  }
}
