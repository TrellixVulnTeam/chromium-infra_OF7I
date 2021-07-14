// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';
import 'elements/issue-detail/metadata/mr-edit-field/mr-edit-field.js';
import 'elements/framework/mr-error/mr-error.js';
import 'react/mr-react-autocomplete.tsx';
import {prpcClient} from 'prpc-client-instance.js';
import {EMPTY_FIELD_VALUE} from 'shared/issue-fields.js';
import {TEXT_TO_STATUS_ENUM} from 'shared/consts/approval.js';


export const NO_UPDATES_MESSAGE =
  'User lacks approver perms for approval in all issues.';
export const NO_APPROVALS_MESSAGE = 'These issues don\'t have any approvals.';

export class MrBulkApprovalUpdate extends LitElement {
  /** @override */
  render() {
    return html`
      <style>
        mr-bulk-approval-update {
          display: block;
          margin-top: 30px;
          position: relative;
        }
        button.clickable-text {
          background: none;
          border: 0;
          color: hsl(0, 0%, 39%);
          cursor: pointer;
          text-decoration: underline;
        }
        .hidden {
          display: none; !important;
        }
        .message {
          background-color: beige;
          width: 500px;
        }
        .note {
          color: hsl(0, 0%, 25%);
          font-size: 0.85em;
          font-style: italic;
        }
        mr-bulk-approval-update table {
          border: 1px dotted black;
          cellspacing: 0;
          cellpadding: 3;
        }
        #approversInput {
          border-style: none;
        }
      </style>
      <button
        class="js-showApprovals clickable-text"
        ?hidden=${this.approvalsFetched}
        @click=${this.fetchApprovals}
      >Show Approvals</button>
      ${this.approvals.length ? html`
        <form>
          <table>
            <tbody><tr>
              <th><label for="approvalSelect">Approval:</label></th>
              <td>
                <select
                  id="approvalSelect"
                  @change=${this._changeHandlers.approval}
                >
                  ${this.approvals.map(({fieldRef}) => html`
                    <option
                      value=${fieldRef.fieldName}
                      .selected=${fieldRef.fieldName === this._values.approval}
                    >
                      ${fieldRef.fieldName}
                    </option>
                  `)}
                </select>
              </td>
            </tr>
            <tr>
              <th><label for="approversInput">Approvers:</label></th>
              <td>
                <mr-react-autocomplete
                  label="approversInput"
                  vocabularyName="member"
                  .multiple=${true}
                  .value=${this._values.approvers}
                  .onChange=${this._changeHandlers.approvers}
                ></mr-react-autocomplete>
              </td>
            </tr>
            <tr><th><label for="statusInput">Status:</label></th>
              <td>
                <select
                  id="statusInput"
                  @change=${this._changeHandlers.status}
                >
                  <option .selected=${!this._values.status}>
                    ${EMPTY_FIELD_VALUE}
                  </option>
                  ${this.statusOptions.map((status) => html`
                    <option
                      value=${status}
                      .selected=${status === this._values.status}
                    >${status}</option>
                  `)}
                </select>
              </td>
            </tr>
            <tr>
              <th><label for="commentText">Comment:</label></th>
              <td colspan="4">
                <textarea
                  cols="30"
                  rows="3"
                  id="commentText"
                  placeholder="Add an approval comment"
                  .value=${this._values.comment || ''}
                  @change=${this._changeHandlers.comment}
                ></textarea>
              </td>
            </tr>
            <tr>
              <td>
                <button
                  class="js-save"
                  @click=${this.save}
                >Update Approvals only</button>
              </td>
              <td>
                <span class="note">
                 Note: Some approvals may not be updated if you lack
                 approver perms.
                </span>
              </td>
            </tr>
          </tbody></table>
        </form>
      `: ''}
      <div class="message">
        ${this.responseMessage}
        ${this.errorMessage ? html`
          <mr-error>${this.errorMessage}</mr-error>
        ` : ''}
      </div>
    `;
  }

  /** @override */
  static get properties() {
    return {
      approvals: {type: Array},
      approvalsFetched: {type: Boolean},
      statusOptions: {type: Array},
      localIdsStr: {type: String},
      projectName: {type: String},
      responseMessage: {type: String},
      _values: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();
    this.approvals = [];
    this.statusOptions = Object.keys(TEXT_TO_STATUS_ENUM);
    this.responseMessage = '';

    this._values = {};
    this._changeHandlers = {
      approval: this._onChange.bind(this, 'approval'),
      approvers: this._onChange.bind(this, 'approvers'),
      status: this._onChange.bind(this, 'status'),
      comment: this._onChange.bind(this, 'comment'),
    };
  }

  /** @override */
  createRenderRoot() {
    return this;
  }

  get issueRefs() {
    const {projectName, localIdsStr} = this;
    if (!projectName || !localIdsStr) return [];
    const issueRefs = [];
    const localIds = localIdsStr.split(',');
    localIds.forEach((localId) => {
      issueRefs.push({projectName: projectName, localId: localId});
    });
    return issueRefs;
  }

  fetchApprovals(evt) {
    const message = {issueRefs: this.issueRefs};
    prpcClient.call('monorail.Issues', 'ListApplicableFieldDefs', message).then(
        (resp) => {
          if (resp.fieldDefs) {
            this.approvals = resp.fieldDefs.filter((fieldDef) => {
              return fieldDef.fieldRef.type == 'APPROVAL_TYPE';
            });
          }
          if (!this.approvals.length) {
            this.errorMessage = NO_APPROVALS_MESSAGE;
          }
          this.approvalsFetched = true;
        }, (error) => {
          this.approvalsFetched = true;
          this.errorMessage = error;
        });
  }

  save(evt) {
    this.responseMessage = '';
    this.errorMessage = '';
    this.toggleDisableForm();
    const selectedFieldDef = this.approvals.find(
        (approval) => approval.fieldRef.fieldName === this._values.approval
    ) || this.approvals[0];
    const message = {
      issueRefs: this.issueRefs,
      fieldRef: selectedFieldDef.fieldRef,
      send_email: true,
    };
    message.commentContent = this._values.comment;
    const delta = {};
    if (this._values.status !== EMPTY_FIELD_VALUE) {
      delta.status = TEXT_TO_STATUS_ENUM[this._values.status];
    }
    const approversAdded = this._values.approvers;
    if (approversAdded) {
      delta.approverRefsAdd = approversAdded.map(
          (name) => ({'displayName': name}));
    }
    if (Object.keys(delta).length) {
      message.approvalDelta = delta;
    }
    prpcClient.call('monorail.Issues', 'BulkUpdateApprovals', message).then(
        (resp) => {
          if (resp.issueRefs && resp.issueRefs.length) {
            const idsStr = Array.from(resp.issueRefs,
                (ref) => ref.localId).join(', ');
            this.responseMessage = `${this.getTimeStamp()}: Updated ${
              selectedFieldDef.fieldRef.fieldName} in issues: ${idsStr} (${
              resp.issueRefs.length} of ${this.issueRefs.length}).`;
            this._values = {};
          } else {
            this.errorMessage = NO_UPDATES_MESSAGE;
          };
          this.toggleDisableForm();
        }, (error) => {
          this.errorMessage = error;
          this.toggleDisableForm();
        });
  }

  getTimeStamp() {
    const date = new Date();
    return `${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}`;
  }

  toggleDisableForm() {
    this.querySelectorAll('input, textarea, select, button').forEach(
        (input) => {
          input.disabled = !input.disabled;
        });
  }

  /**
   * Generic onChange handler to be bound to each form field.
   * @param {string} key Unique name for the form field we're binding this
   *   handler to. For example, 'owner', 'cc', or the name of a custom field.
   * @param {Event | React.SyntheticEvent} event
   * @param {string} value The new form value.
   */
  _onChange(key, event, value) {
    this._values = {...this._values, [key]: value || event.target.value};
  }
}

customElements.define('mr-bulk-approval-update', MrBulkApprovalUpdate);
