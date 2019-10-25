// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {connectStore} from 'reducers/base.js';
import 'elements/chops/chops-timestamp/chops-timestamp.js';
import 'elements/framework/links/mr-issue-link/mr-issue-link.js';
import 'elements/framework/links/mr-user-link/mr-user-link.js';

import * as issue from 'reducers/issue.js';
import './mr-field-values.js';
import {EMPTY_FIELD_VALUE} from 'shared/issue-fields.js';
import {HARDCODED_FIELD_GROUPS, valuesForField, fieldDefsWithGroup,
  fieldDefsWithoutGroup} from 'shared/metadata-helpers.js';
import 'shared/typedef.js';
import {AVAILABLE_CUES, cueNames, specToCueName,
  cueNameToSpec} from 'elements/help/mr-cue/cue-helpers.js';


/**
 * `<mr-metadata>`
 *
 * Generalized metadata components, used for either approvals or issues.
 *
 */
export class MrMetadata extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: table;
        table-layout: fixed;
        width: 100%;
      }
      td, th {
        padding: 0.5em 4px;
        vertical-align: top;
        text-overflow: ellipsis;
        overflow: hidden;
      }
      td {
        width: 60%;
      }
      td.allow-overflow {
        overflow: visible;
      }
      th {
        text-align: left;
        width: 40%;
      }
      .group-separator {
        border-top: var(--chops-normal-border);
      }
      .group-title {
        font-weight: normal;
        font-style: oblique;
        border-bottom: var(--chops-normal-border);
        text-align: center;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons"
            rel="stylesheet">
      ${this._renderBuiltInFields()}
      ${this._renderCustomFieldGroups()}
    `;
  }

  /**
   * Helper for handling the rendering of built in fields.
   * @return {Array<TemplateResult>}
   */
  _renderBuiltInFields() {
    return this.builtInFieldSpec.map((fieldName) => {
      const fieldKey = fieldName.toLowerCase();

      // Adding classes to table rows based on field names makes selecting
      // rows with specific values easier, for example in tests.
      let className = `row-${fieldKey}`;

      const cueName = specToCueName(fieldKey);
      if (cueName) {
        className = `cue-${cueName}`;

        if (!AVAILABLE_CUES.has(cueName)) return '';

        return html`
          <tr class=${className}>
            <td colspan="2">
              <mr-cue cuePrefName=${cueName}></mr-cue>
            </td>
          </tr>
        `;
      }

      const isApprovalStatus = fieldKey === 'approvalstatus';
      const isMergedInto = fieldKey === 'mergedinto';

      const fieldValueTemplate = this._renderBuiltInFieldValue(fieldName);

      if (!fieldValueTemplate) return '';

      // Allow overflow to enable the FedRef popup to expand.
      // TODO(jeffcarp): Look into a more elegant solution.
      return html`
        <tr class=${className}>
          <th>${isApprovalStatus ? 'Status' : fieldName}:</th>
          <td class=${isMergedInto ? 'allow-overflow' : ''}>
            ${fieldValueTemplate}
          </td>
        </tr>
      `;
    });
  }

  /**
   * A helper to display a single built-in field.
   *
   * @param {String} fieldName The name of the built in field to render.
   * @return {TemplateResult|undefined} lit-html template for displaying the
   *   value of the built in field. If undefined, the rendering code assumes
   *   that the field should be hidden if empty.
   */
  _renderBuiltInFieldValue(fieldName) {
    // TODO(zhangtiff): Merge with code in shared/issue-fields.js for further
    // de-duplication.
    switch (fieldName.toLowerCase()) {
      case 'approvalstatus':
        return this.approvalStatus || EMPTY_FIELD_VALUE;
      case 'approvers':
        return this.approvers && this.approvers.length ?
          this.approvers.map((approver) => html`
            <mr-user-link
              .userRef=${approver}
              showAvailabilityIcon
            ></mr-user-link>
            <br />
          `) : EMPTY_FIELD_VALUE;
      case 'setter':
        return this.setter ? html`
          <mr-user-link
            .userRef=${this.setter}
            showAvailabilityIcon
          ></mr-user-link>
          ` : undefined; // Hide the field when empty.
      case 'owner':
        return this.owner ? html`
          <mr-user-link
            .userRef=${this.owner}
            showAvailabilityIcon
            showAvailabilityText
          ></mr-user-link>
          ` : EMPTY_FIELD_VALUE;
      case 'cc':
        return this.cc && this.cc.length ?
          this.cc.map((cc) => html`
            <mr-user-link
              .userRef=${cc}
              showAvailabilityIcon
            ></mr-user-link>
            <br />
          `) : EMPTY_FIELD_VALUE;
      case 'status':
        return this.issueStatus ? html`
          ${this.issueStatus.status} <em>${
            this.issueStatus.meansOpen ? '(Open)' : '(Closed)'}
          </em>` : EMPTY_FIELD_VALUE;
      case 'mergedinto':
        // TODO(zhangtiff): This should use the project config to determine if a
        // field allows merging rather than used a hard-coded value.
        return this.issueStatus && this.issueStatus.status === 'Duplicate' ?
          html`
            <mr-issue-link
              .projectName=${this.issueRef.projectName}
              .issue=${this.mergedInto}
            ></mr-issue-link>
          `: undefined; // Hide the field when empty.
      case 'components':
        return (this.components && this.components.length) ?
          this.components.map((comp) => html`
            <a
              href="/p/${this.issueRef.projectName
                }/issues/list?q=component:${comp.path}"
              title="${comp.path}${comp.docstring ?
                ' = ' + comp.docstring : ''}"
            >
              ${comp.path}</a><br />
          `) : EMPTY_FIELD_VALUE;
      case 'modified':
        return this.modifiedTimestamp ? html`
            <chops-timestamp
              .timestamp=${this.modifiedTimestamp}
              short
            ></chops-timestamp>
          ` : EMPTY_FIELD_VALUE;
    }

    // Non-existent field.
    return;
  }

  /**
   * Helper for handling the rendering of custom fields defined in a project
   * config.
   * @return {TemplateResult} lit-html template.
   */
  _renderCustomFieldGroups() {
    const grouped = fieldDefsWithGroup(this.fieldDefs,
        this.fieldGroups, this.issueType);
    const ungrouped = fieldDefsWithoutGroup(this.fieldDefs,
        this.fieldGroups, this.issueType);
    return html`
      ${grouped.map((group) => html`
        <tr>
          <th class="group-title" colspan="2">
            ${group.groupName}
          </th>
        </tr>
        ${this._renderCustomFields(group.fieldDefs)}
        <tr>
          <th class="group-separator" colspan="2"></th>
        </tr>
      `)}

      ${this._renderCustomFields(ungrouped)}
    `;
  }

  /**
   * Helper for handling the rendering of built in fields.
   *
   * @param {Array<FieldDef>} fieldDefs Arrays of configurations Objects
   *   for fields to render.
   * @return {Array<TemplateResult>} Array of lit-html templates to render, each
   *   representing a single table row for a custom field.
   */
  _renderCustomFields(fieldDefs) {
    if (!fieldDefs || !fieldDefs.length) return [];
    return fieldDefs.map((field) => {
      const fieldValues = valuesForField(
          this.fieldValueMap, field.fieldRef.fieldName) || [];
      return html`
        <tr ?hidden=${field.isNiche && !fieldValues.length}>
          <th title=${field.docstring}>${field.fieldRef.fieldName}:</th>
          <td>
            <mr-field-values
              .name=${field.fieldRef.fieldName}
              .type=${field.fieldRef.type}
              .values=${fieldValues}
              .projectName=${this.issueRef.projectName}
            ></mr-field-values>
          </td>
        </tr>
      `;
    });
  }

  /** @override */
  static get properties() {
    return {
      /**
       * An Array of Strings to specify which built in fields to display.
       */
      builtInFieldSpec: {type: Array},
      approvalStatus: {type: Array},
      approvers: {type: Array},
      setter: {type: Object},
      cc: {type: Array},
      components: {type: Array},
      fieldDefs: {type: Array},
      fieldGroups: {type: Array},
      issueStatus: {type: String},
      issueType: {type: String},
      mergedInto: {type: Object},
      modifiedTimestamp: {type: Number},
      owner: {type: Object},
      isApproval: {type: Boolean},
      issueRef: {type: Object},
      fieldValueMap: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();
    this.isApproval = false;
    this.fieldGroups = HARDCODED_FIELD_GROUPS;
    this.issueRef = {};

    // Default built in fields used by issue metadata.
    this.builtInFieldSpec = [
      'Owner', 'CC', cueNameToSpec(cueNames.AVAILABILITY_MSGS),
      'Status', 'MergedInto', 'Components', 'Modified',
    ];
    this.fieldValueMap = new Map();

    this.approvalStatus = undefined;
    this.approvers = undefined;
    this.setter = undefined;
    this.cc = undefined;
    this.components = undefined;
    this.fieldDefs = undefined;
    this.issueStatus = undefined;
    this.issueType = undefined;
    this.mergedInto = undefined;
    this.owner = undefined;
    this.modifiedTimestamp = undefined;
  }

  /** @override */
  connectedCallback() {
    super.connectedCallback();

    // This is set for accessibility. Do not override.
    this.setAttribute('role', 'table');
  }

  /** @override */
  stateChanged(state) {
    this.fieldValueMap = issue.fieldValueMap(state);
    this.issueType = issue.type(state);
    this.issueRef = issue.issueRef(state);
    this.relatedIssues = issue.relatedIssues(state);
  }
}

customElements.define('mr-metadata', MrMetadata);
