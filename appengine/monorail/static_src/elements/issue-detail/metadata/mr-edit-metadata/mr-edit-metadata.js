// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import debounce from 'debounce';

import 'elements/chops/chops-button/chops-button.js';
import 'elements/framework/mr-upload/mr-upload.js';
import 'elements/framework/mr-star-button/mr-star-button.js';
import 'elements/chops/chops-checkbox/chops-checkbox.js';
import 'elements/chops/chops-chip/chops-chip.js';
import 'elements/framework/mr-error/mr-error.js';
import 'elements/framework/mr-warning/mr-warning.js';
import 'elements/help/mr-cue/mr-cue.js';
import {cueNames} from 'elements/help/mr-cue/cue-helpers.js';
import {store, connectStore} from 'reducers/base.js';
import {UserInputError} from 'shared/errors.js';
import {fieldTypes} from 'shared/issue-fields.js';
import {displayNameToUserRef, labelStringToRef, componentStringToRef,
  componentRefsToStrings, issueStringToRef, issueStringToBlockingRef,
  issueRefToString, issueRefsToStrings, filteredUserDisplayNames,
  valueToFieldValue,
} from 'shared/converters.js';
import {isEmptyObject, equalsIgnoreCase} from 'shared/helpers.js';
import {NON_EDITING_KEY_EVENTS} from 'shared/dom-helpers.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';
import * as issueV0 from 'reducers/issueV0.js';
import * as projectV0 from 'reducers/projectV0.js';
import * as ui from 'reducers/ui.js';
import '../mr-edit-field/mr-edit-field.js';
import '../mr-edit-field/mr-edit-status.js';
import {ISSUE_EDIT_PERMISSION, ISSUE_EDIT_SUMMARY_PERMISSION,
  ISSUE_EDIT_STATUS_PERMISSION, ISSUE_EDIT_OWNER_PERMISSION,
  ISSUE_EDIT_CC_PERMISSION,
} from 'shared/permissions.js';
import {fieldDefsWithGroup, fieldDefsWithoutGroup, valuesForField,
  HARDCODED_FIELD_GROUPS} from 'shared/metadata-helpers.js';


const DEBOUNCED_PRESUBMIT_TIME_OUT = 400;


/**
 * `<mr-edit-metadata>`
 *
 * Editing form for either an approval or the overall issue.
 *
 */
export class MrEditMetadata extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          display: block;
          font-size: var(--chops-main-font-size);
        }
        :host(.edit-actions-right) .edit-actions {
          flex-direction: row-reverse;
          text-align: right;
        }
        :host(.edit-actions-right) .edit-actions chops-checkbox {
          text-align: left;
        }
        .edit-actions chops-checkbox {
          max-width: 200px;
          margin-top: 2px;
          flex-grow: 2;
          text-align: right;
        }
        .edit-actions {
          width: 100%;
          max-width: 500px;
          margin: 0.5em 0;
          text-align: left;
          display: flex;
          flex-direction: row;
          align-items: center;
        }
        .edit-actions chops-button {
          flex-grow: 0;
          flex-shrink: 0;
        }
        .edit-actions .emphasized {
          margin-left: 0;
        }
        input {
          box-sizing: border-box;
          width: var(--mr-edit-field-width);
          padding: var(--mr-edit-field-padding);
          font-size: var(--chops-main-font-size);
        }
        mr-upload {
          margin-bottom: 0.25em;
        }
        textarea {
          font-family: var(--mr-toggled-font-family);
          width: 100%;
          margin: 0.25em 0;
          box-sizing: border-box;
          border: var(--chops-accessible-border);
          height: 8em;
          transition: height 0.1s ease-in-out;
          padding: 0.5em 4px;
          grid-column-start: 1;
          grid-column-end: 2;
        }
        button.toggle {
          background: none;
          color: var(--chops-link-color);
          border: 0;
          width: 100%;
          padding: 0.25em 0;
          text-align: left;
        }
        button.toggle:hover {
          cursor: pointer;
          text-decoration: underline;
        }
        .presubmit-derived {
          color: gray;
          font-style: italic;
          text-decoration-line: underline;
          text-decoration-style: dotted;
        }
        .presubmit-derived-header {
          color: gray;
          font-weight: bold;
        }
        .discard-button {
          margin-right: 16px;
          margin-left: 16px;
        }
        .group {
          width: 100%;
          border: 1px solid hsl(0, 0%, 83%);
          grid-column: 1 / -1;
          margin: 0;
          margin-bottom: 0.5em;
          padding: 0;
          padding-bottom: 0.5em;
        }
        .group legend {
          margin-left: 130px;
        }
        .group-title {
          text-align: center;
          font-style: oblique;
          margin-top: 4px;
          margin-bottom: -8px;
        }
        .star-line {
          display: flex;
          align-items: center;
          background: var(--chops-notice-bubble-bg);
          border: var(--chops-notice-border);
          justify-content: flex-start;
          margin-top: 4px;
          padding: 2px 4px 2px 8px;
        }
        mr-star-button {
          margin-right: 4px;
        }
        .predicted-component {
          cursor: pointer;
        }
      `,
    ];
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons"
            rel="stylesheet">
      <form id="editForm"
        @submit=${this._save}
        @keydown=${this._saveOnCtrlEnter}
      >
        <mr-cue cuePrefName=${cueNames.CODE_OF_CONDUCT}></mr-cue>
        ${this._renderStarLine()}
        <textarea
          id="commentText"
          placeholder="Add a comment"
          @keyup=${this._processChanges}
          aria-label="Comment"
        ></textarea>
        <mr-upload
          ?hidden=${this.disableAttachments}
          @change=${this._processChanges}
        ></mr-upload>
        <div class="input-grid">
          ${this._renderEditFields()}
          ${this._renderErrorsAndWarnings()}

          <span></span>
          <div class="edit-actions">
            <chops-button
              @click=${this._save}
              class="save-changes emphasized"
              ?disabled=${this.disabled}
              title="Save changes (Ctrl+Enter / \u2318+Enter)"
            >
              Save changes
            </chops-button>
            <chops-button
              @click=${this.discard}
              class="de-emphasized discard-button"
              ?disabled=${this.disabled}
            >
              Discard
            </chops-button>

            <chops-checkbox
              id="sendEmail"
              @checked-change=${this._sendEmailChecked}
              ?checked=${this.sendEmail}
            >Send email</chops-checkbox>
          </div>

          ${!this.isApproval ? this._renderPresubmitChanges() : ''}
        </div>
      </form>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderStarLine() {
    if (this._canEditIssue || this.isApproval) return '';

    return html`
      <div class="star-line">
        <mr-star-button
          .issueRef=${this.issueRef}
        ></mr-star-button>
        <span>
          ${this.isStarred ? `
            You have voted for this issue and will receive notifications.
          ` : `
            Star this issue instead of commenting "+1 Me too!" to add a vote
            and get notifications.`}
        </span>
      </div>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderPresubmitChanges() {
    const {derivedCcs, derivedLabels} = this.presubmitResponse || {};
    const hasCcs = derivedCcs && derivedCcs.length;
    const hasLabels = derivedLabels && derivedLabels.length;
    const hasDerivedValues = hasCcs || hasLabels;
    return html`
      ${hasDerivedValues ? html`
        <span></span>
        <div class="presubmit-derived-header">
          Filter rules and components will add
        </div>
        ` : ''}

      ${hasCcs? html`
        <label
          for="derived-ccs"
          class="presubmit-derived-header"
        >CC:</label>
        <div id="derived-ccs">
          ${derivedCcs.map((cc) => html`
            <span
              title=${cc.why}
              class="presubmit-derived"
            >${cc.value}</span>
          `)}
        </div>
        ` : ''}

      ${hasLabels ? html`
        <label
          for="derived-labels"
          class="presubmit-derived-header"
        >Labels:</label>
        <div id="derived-labels">
          ${derivedLabels.map((label) => html`
            <span
              title=${label.why}
              class="presubmit-derived"
            >${label.value}</span>
          `)}
        </div>
        ` : ''}
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderErrorsAndWarnings() {
    const presubmitResponse = this.presubmitResponse || {};
    const presubmitWarnings = presubmitResponse.warnings || [];
    const presubmitErrors = presubmitResponse.errors || [];
    return (this.error || presubmitWarnings.length || presubmitErrors.length) ?
      html`
        <span></span>
        <div>
          ${presubmitWarnings.map((warning) => html`
            <mr-warning title=${warning.why}>${warning.value}</mr-warning>
          `)}
          <!-- TODO(ehmaldonado): Look into blocking submission on presubmit
          -->
          ${presubmitErrors.map((error) => html`
            <mr-error title=${error.why}>${error.value}</mr-error>
          `)}
          ${this.error ? html`
            <mr-error>${this.error}</mr-error>` : ''}
        </div>
      ` : '';
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderEditFields() {
    if (this.isApproval) {
      return html`
        ${this._renderStatus()}
        ${this._renderApprovers()}
        ${this._renderFieldDefs()}

        ${this._renderNicheFieldToggle()}
      `;
    }

    return html`
      ${this._canEditSummary ? this._renderSummary() : ''}
      ${this._canEditStatus ? this._renderStatus() : ''}
      ${this._canEditOwner ? this._renderOwner() : ''}
      ${this._canEditCC ? this._renderCC() : ''}
      ${this._canEditIssue ? html`
        ${this._renderComponents()}

        ${this._renderFieldDefs()}
        ${this._renderRelatedIssues()}
        ${this._renderLabels()}

        ${this._renderNicheFieldToggle()}
      ` : ''}
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderSummary() {
    return html`
      <label for="summaryInput">Summary:</label>
      <input
        id="summaryInput"
        value=${this.summary}
        @keyup=${this._processChanges}
      />
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderOwner() {
    const ownerPresubmit = this._ownerPresubmit;
    return html`
      <label for="ownerInput" @click=${this._clickLabelForCustomInput}>
        ${ownerPresubmit.message ? html`
          <i
            class=${`material-icons inline-${ownerPresubmit.icon}`}
            title=${ownerPresubmit.message}
          >${ownerPresubmit.icon}</i>
        ` : ''}
        Owner:
      </label>
      <mr-edit-field
        id="ownerInput"
        .name=${'Owner'}
        .type=${'USER_TYPE'}
        .initialValues=${this.ownerName ? [this.ownerName] : []}
        .acType=${'owner'}
        .placeholder=${ownerPresubmit.placeholder}
        @change=${this._processChanges}
      ></mr-edit-field>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderCC() {
    return html`
      <label for="ccInput" @click=${this._clickLabelForCustomInput}>CC:</label>
      <mr-edit-field
        id="ccInput"
        .name=${'CC'}
        .type=${'USER_TYPE'}
        .initialValues=${this._ccNames}
        .derivedValues=${this._derivedCCs}
        .acType=${'member'}
        @change=${this._processChanges}
        multi
      ></mr-edit-field>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderComponents() {
    return html`
      <label
        for="componentsInput"
        @click=${this._clickLabelForCustomInput}
      >Components:</label>
      <mr-edit-field
        id="componentsInput"
        .name=${'component'}
        .type=${'STR_TYPE'}
        .initialValues=${componentRefsToStrings(this.components)}
        .acType=${'component'}
        @change=${this._processChanges}
        multi
      ></mr-edit-field>
      ${this._renderPredictedComponent()}
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderPredictedComponent() {
    if (!this.predictedComponent) return '';

    const componentsInput = this.shadowRoot.getElementById('componentsInput');
    const components = componentsInput ?
      [...componentsInput.values] :
      componentRefsToStrings(this.components);
    if (components.includes(this.predictedComponent)) {
      return '';
    }

    return html`
      <span></span>
      <div>
        <i>Suggested:</i>
        <chops-chip
          class="predicted-component"
          title="Click to add ${this.predictedComponent} to components"
          @keyup=${this._addPredictedComponent}
          @click=${this._addPredictedComponent}
          focusable
        >
          ${this.predictedComponent}
        </chops-chip>
      </div>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderApprovers() {
    return this.hasApproverPrivileges && this.isApproval ? html`
      <label for="approversInput" @click=${this._clickLabelForCustomInput}>Approvers:</label>
      <mr-edit-field
        id="approversInput"
        .type=${'USER_TYPE'}
        .initialValues=${filteredUserDisplayNames(this.approvers)}
        .name=${'approver'}
        .acType=${'member'}
        @change=${this._processChanges}
        multi
      ></mr-edit-field>
    ` : '';
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderStatus() {
    return this.statuses && this.statuses.length ? html`
      <label for="statusInput">Status:</label>

      <mr-edit-status
        id="statusInput"
        .initialStatus=${this.status}
        .statuses=${this.statuses}
        .mergedInto=${issueRefToString(this.mergedInto, this.projectName)}
        ?isApproval=${this.isApproval}
        @change=${this._processChanges}
      ></mr-edit-status>
    ` : '';
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderFieldDefs() {
    return html`
      ${fieldDefsWithGroup(this.fieldDefs, this.fieldGroups, this.issueType).map((group) => html`
        <fieldset class="group">
          <legend>${group.groupName}</legend>
          <div class="input-grid">
            ${group.fieldDefs.map((field) => this._renderCustomField(field))}
          </div>
        </fieldset>
      `)}

      ${fieldDefsWithoutGroup(this.fieldDefs, this.fieldGroups, this.issueType).map((field) => this._renderCustomField(field))}
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderRelatedIssues() {
    return html`
      <label for="blockedOnInput" @click=${this._clickLabelForCustomInput}>BlockedOn:</label>
      <mr-edit-field
        id="blockedOnInput"
        .initialValues=${issueRefsToStrings(this.blockedOn, this.projectName)}
        .name=${'blockedOn'}
        @change=${this._processChanges}
        multi
      ></mr-edit-field>

      <label for="blockingInput" @click=${this._clickLabelForCustomInput}>Blocking:</label>
      <mr-edit-field
        id="blockingInput"
        .initialValues=${issueRefsToStrings(this.blocking, this.projectName)}
        .name=${'blocking'}
        @change=${this._processChanges}
        multi
      ></mr-edit-field>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderLabels() {
    return html`
      <label for="labelsInput" @click=${this._clickLabelForCustomInput}>Labels:</label>
      <mr-edit-field
        id="labelsInput"
        .acType=${'label'}
        .initialValues=${this.labelNames}
        .derivedValues=${this.derivedLabels}
        .name=${'labels'}
        @change=${this._processChanges}
        multi
      ></mr-edit-field>
    `;
  }

  /**
   * @return {TemplateResult}
   * @param {FieldDef} field The custom field beinf rendered.
   * @private
   */
  _renderCustomField(field) {
    if (!field || !field.fieldRef) return '';
    const {fieldRef, isNiche, docstring, isMultivalued} = field;
    const isHidden = !this.showNicheFields && isNiche;

    let acType;
    if (fieldRef.type === fieldTypes.USER_TYPE) {
      acType = isMultivalued ? 'member' : 'owner';
    }
    return html`
      <label
        ?hidden=${isHidden}
        for=${this._idForField(fieldRef.fieldName)}
        @click=${this._clickLabelForCustomInput}
        title=${docstring}
      >
        ${fieldRef.fieldName}:
      </label>
      <mr-edit-field
        ?hidden=${isHidden}
        id=${this._idForField(fieldRef.fieldName)}
        .name=${fieldRef.fieldName}
        .type=${fieldRef.type}
        .options=${this._optionsForField(this.optionsPerEnumField, this.fieldValueMap, fieldRef.fieldName, this.phaseName)}
        .initialValues=${valuesForField(this.fieldValueMap, fieldRef.fieldName, this.phaseName)}
        .acType=${acType}
        ?multi=${isMultivalued}
        @change=${this._processChanges}
      ></mr-edit-field>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderNicheFieldToggle() {
    return this._nicheFieldCount ? html`
      <span></span>
      <button type="button" class="toggle" @click=${this.toggleNicheFields}>
        <span ?hidden=${this.showNicheFields}>
          Show all fields (${this._nicheFieldCount} currently hidden)
        </span>
        <span ?hidden=${!this.showNicheFields}>
          Hide niche fields (${this._nicheFieldCount} currently shown)
        </span>
      </button>
    ` : '';
  }

  /** @override */
  static get properties() {
    return {
      fieldDefs: {type: Array},
      formName: {type: String},
      approvers: {type: Array},
      setter: {type: Object},
      summary: {type: String},
      cc: {type: Array},
      components: {type: Array},
      status: {type: String},
      statuses: {type: Array},
      blockedOn: {type: Array},
      blocking: {type: Array},
      mergedInto: {type: Object},
      ownerName: {type: String},
      labelNames: {type: Array},
      derivedLabels: {type: Array},
      phaseName: {type: String},
      projectConfig: {type: Object},
      projectName: {type: String},
      isApproval: {type: Boolean},
      isStarred: {type: Boolean},
      issuePermissions: {type: Object},
      issueRef: {type: Object},
      hasApproverPrivileges: {type: Boolean},
      showNicheFields: {type: Boolean},
      disableAttachments: {type: Boolean},
      error: {type: String},
      sendEmail: {type: Boolean},
      presubmitResponse: {type: Object},
      predictedComponent: {type: String},
      fieldValueMap: {type: Object},
      issueType: {type: String},
      optionsPerEnumField: {type: String},
      fieldGroups: {type: Object},
      saving: {type: Boolean},
      isDirty: {type: Boolean},
      _debouncedProcessChanges: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();
    this.summary = '';
    this.ownerName = '';
    this.sendEmail = true;
    this.mergedInto = {};
    this.issueRef = {};
    this.fieldGroups = HARDCODED_FIELD_GROUPS;

    this.presubmitDebounceTimeOut = DEBOUNCED_PRESUBMIT_TIME_OUT;
    this.saving = false;
    this.isDirty = false;
  }

  /** @override */
  firstUpdated() {
    this.hasRendered = true;
  }

  get disabled() {
    return !this.isDirty || this.saving;
  }

  // Set isDirty to a property instead of only using a getter to cause
  // lit-element to re-render when dirty state change.
  _updateIsDirty() {
    if (!this.hasRendered) return;

    const commentContent = this.getCommentContent();
    const attachmentsElement = this.shadowRoot.querySelector('mr-upload');
    this.isDirty = !isEmptyObject(this.delta) || Boolean(commentContent) ||
      attachmentsElement.hasAttachments;
  }

  get _nicheFieldCount() {
    const fieldDefs = this.fieldDefs || [];
    return fieldDefs.reduce((acc, fd) => acc + (fd.isNiche | 0), 0);
  }

  get _canEditIssue() {
    const issuePermissions = this.issuePermissions || [];
    return issuePermissions.includes(ISSUE_EDIT_PERMISSION);
  }

  get _canEditSummary() {
    const issuePermissions = this.issuePermissions || [];
    return this._canEditIssue ||
      issuePermissions.includes(ISSUE_EDIT_SUMMARY_PERMISSION);
  }

  get _canEditStatus() {
    const issuePermissions = this.issuePermissions || [];
    return this._canEditIssue ||
      issuePermissions.includes(ISSUE_EDIT_STATUS_PERMISSION);
  }

  get _canEditOwner() {
    const issuePermissions = this.issuePermissions || [];
    return this._canEditIssue ||
      issuePermissions.includes(ISSUE_EDIT_OWNER_PERMISSION);
  }

  get _canEditCC() {
    const issuePermissions = this.issuePermissions || [];
    return this._canEditIssue ||
      issuePermissions.includes(ISSUE_EDIT_CC_PERMISSION);
  }

  get _ccNames() {
    const users = this.cc || [];
    return filteredUserDisplayNames(users.filter((u) => !u.isDerived));
  }

  get _derivedCCs() {
    const users = this.cc || [];
    return filteredUserDisplayNames(users.filter((u) => u.isDerived));
  }

  get _ownerPresubmit() {
    const response = this.presubmitResponse;
    if (!response) return {};

    const ownerView = {message: '', placeholder: '', icon: ''};

    if (response.ownerAvailability) {
      ownerView.message = response.ownerAvailability;
      ownerView.icon = 'warning';
    } else if (response.derivedOwners && response.derivedOwners.length) {
      ownerView.placeholder = response.derivedOwners[0].value;
      ownerView.message = response.derivedOwners[0].why;
      ownerView.icon = 'info';
    }
    return ownerView;
  }

  /** @override */
  stateChanged(state) {
    this.fieldValueMap = issueV0.fieldValueMap(state);
    this.issueType = issueV0.type(state);
    this.issueRef = issueV0.viewedIssueRef(state);
    this.presubmitResponse = issueV0.presubmitResponse(state);
    this.predictedComponent = issueV0.predictedComponent(state);
    this.projectConfig = projectV0.viewedConfig(state);
    this.projectName = issueV0.viewedIssueRef(state).projectName;
    this.issuePermissions = issueV0.permissions(state);
    this.optionsPerEnumField = projectV0.optionsPerEnumField(state);
    // Access boolean value from allStarredIssues
    const starredIssues = issueV0.starredIssues(state);
    this.isStarred = starredIssues.has(issueRefToString(this.issueRef));
  }

  /** @override */
  disconnectedCallback() {
    super.disconnectedCallback();

    if (this._debouncedProcessChanges) {
      this._debouncedProcessChanges.clear();
    }

    store.dispatch(ui.reportDirtyForm(this.formName, false));
  }

  /**
   * Resets the edit form values to their default values.
   */
  reset() {
    const form = this.shadowRoot.querySelector('#editForm');
    if (!form) return;

    form.reset();
    const statusInput = this.shadowRoot.querySelector('#statusInput');
    if (statusInput) {
      statusInput.reset();
    }

    // Since custom elements containing <input> elements have the inputs
    // wrapped in ShadowDOM, those inputs don't get reset with the rest of
    // the form. Haven't been able to figure out a way to replicate form reset
    // behavior with custom input elements.
    if (this.isApproval) {
      if (this.hasApproverPrivileges) {
        const approversInput = this.shadowRoot.querySelector(
            '#approversInput');
        if (approversInput) {
          approversInput.reset();
        }
      }
    }
    this.shadowRoot.querySelectorAll('mr-edit-field').forEach((el) => {
      el.reset();
    });

    const uploader = this.shadowRoot.querySelector('mr-upload');
    if (uploader) {
      uploader.reset();
    }

    this._processChanges();
  }

  /**
   * @param {MouseEvent|SubmitEvent} event
   * @private
   */
  _save(event) {
    event.preventDefault();
    this.save();
  }

  /**
   * Users may use either Ctrl+Enter or Command+Enter to save an issue edit
   * while the issue edit form is focused.
   * @param {KeyboardEvent} event
   * @private
   */
  _saveOnCtrlEnter(event) {
    if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
      event.preventDefault();
      this.save();
    }
  }

  /**
   * Tells the parent to save the current edited values in the form.
   * @fires CustomEvent#save
   */
  save() {
    this.dispatchEvent(new CustomEvent('save'));
  }

  /**
   * Tells the parent component that the user is trying to discard the form,
   * if they confirm that that's what they're doing. The parent decides what
   * to do in order to quit the editing session.
   * @fires CustomEvent#discard
   */
  discard() {
    const isDirty = this.isDirty;
    if (!isDirty || confirm('Discard your changes?')) {
      this.dispatchEvent(new CustomEvent('discard'));
    }
  }

  /**
   * Focuses the comment form.
   */
  async focus() {
    await this.updateComplete;
    this.shadowRoot.querySelector('#commentText').focus();
  }

  /**
   * Retrieves the value of the comment that the user added from the DOM.
   * @return {string}
   */
  getCommentContent() {
    return this.shadowRoot.querySelector('#commentText').value;
  }

  async getAttachments() {
    try {
      return await this.shadowRoot.querySelector('mr-upload').loadFiles();
    } catch (e) {
      this.error = `Error while loading file for attachment: ${e.message}`;
    }
  }

  /**
   * Shows or hides custom fields with the "isNiche" attribute set to true.
   */
  toggleNicheFields() {
    this.showNicheFields = !this.showNicheFields;
  }

  /**
   * @return {IssueDelta}
   * @throws {UserInputError}
   */
  get delta() {
    try {
      this.error = '';
      return this._getDelta();
    } catch (e) {
      if (!(e instanceof UserInputError)) throw e;
      this.error = e.message;
      return {};
    }
  }

  /**
   * Generates a change between the initial Issue state and what the user
   * inputted.
   * @return {IssueDelta}
   */
  _getDelta() {
    const result = {};
    const root = this.shadowRoot;

    const {projectName, localId} = this.issueRef;

    const statusInput = root.querySelector('#statusInput');
    if (this._canEditStatus && statusInput) {
      const statusDelta = statusInput.delta;
      if (statusDelta.mergedInto) {
        result.mergedIntoRef = issueStringToBlockingRef(
            {projectName, localId}, statusDelta.mergedInto);
      }
      if (statusDelta.status) {
        result.status = statusDelta.status;
      }
    }

    if (this.isApproval) {
      if (this._canEditIssue && this.hasApproverPrivileges) {
        this._updateDeltaWithAddedAndRemoved(
            result, 'approvers', 'approverRefs', displayNameToUserRef);
      }
    } else {
      // TODO(zhangtiff): Consider representing baked-in fields such as owner,
      // cc, and status similarly to custom fields to reduce repeated code.

      if (this._canEditSummary) {
        const summaryInput = root.querySelector('#summaryInput');
        if (summaryInput) {
          const newSummary = summaryInput.value;
          if (newSummary !== this.summary) {
            result.summary = newSummary;
          }
        }
      }

      if (this._canEditOwner) {
        const ownerInput = root.querySelector('#ownerInput');
        if (ownerInput) {
          const newOwner = ownerInput.value;
          if (newOwner !== this.ownerName) {
            result.ownerRef = displayNameToUserRef(newOwner);
          }
        }
      }

      if (this._canEditCC) {
        this._updateDeltaWithAddedAndRemoved(
            result, 'cc', 'ccRefs', displayNameToUserRef);
      }

      if (this._canEditIssue) {
        const blockerAddFn = (refString) =>
          issueStringToBlockingRef({projectName, localId}, refString);
        const blockerRemoveFn = (refString) =>
          issueStringToRef(refString, projectName);
        this._updateDeltaWithAddedAndRemoved(
            result, 'labels', 'labelRefs', labelStringToRef);
        this._updateDeltaWithAddedAndRemoved(
            result, 'components', 'compRefs', componentStringToRef);
        this._updateDeltaWithAddedAndRemoved(
            result, 'blockedOn', 'blockedOnRefs',
            blockerAddFn, blockerRemoveFn);
        this._updateDeltaWithAddedAndRemoved(
            result, 'blocking', 'blockingRefs',
            blockerAddFn, blockerRemoveFn);
      }
    }

    if (this._canEditIssue) {
      const fieldDefs = this.fieldDefs || [];
      fieldDefs.forEach(({fieldRef}) => {
        this._updateDeltaWithAddedAndRemoved(
            result, fieldRef.fieldName, 'fieldVals',
            valueToFieldValue.bind(null, fieldRef));
      });
    }

    return result;
  }

  /**
   * Helper function for adding values for a single field to a delta.
   * @param {IssueDelta} delta A delta Object that's edited in place.
   * @param {string} fieldName Name of the field being edited.
   * @param {string} key The key in the delta Object that changes will be
   *   saved in.
   * @param {function(string): any} addFn A function to specify how to format
   *   the message for a given added field.
   * @param {function(string): any} removeFn A function to specify how to format
   *   the message for a given removed field.
   */
  _updateDeltaWithAddedAndRemoved(delta, fieldName, key, addFn, removeFn) {
    const input = this.shadowRoot.querySelector(`#${fieldName}Input`);
    if (!input) return;

    const valuesAdd = input.getValuesAdded();
    if (valuesAdd && valuesAdd.length) {
      delta[key + 'Add'] = (delta[key + 'Add'] || []).concat(
          valuesAdd.map(addFn));
    }

    const valuesRemove = input.getValuesRemoved();
    if (valuesRemove && valuesRemove.length) {
      delta[key + 'Remove'] = (delta[key + 'Remove'] || []).concat(
          valuesRemove.map(removeFn || addFn));
    }
  }

  _processChanges(e) {
    if (e instanceof KeyboardEvent) {
      if (NON_EDITING_KEY_EVENTS.has(e.key)) return;
    }
    this._updateIsDirty();

    if (!this._debouncedProcessChanges) {
      this._debouncedProcessChanges = debounce(() => this._runProcessChanges(),
          this.presubmitDebounceTimeOut);
    }
    this._debouncedProcessChanges();
  }

  /**
   * Non-debounced version of _processChanges
   * @fires CustomEvent#change
   * @private
   */
  _runProcessChanges() {
    // Don't run this functionality if the element has disconnected.
    if (!this.isConnected) return;

    store.dispatch(ui.reportDirtyForm(this.formName, this.isDirty));
    this.dispatchEvent(new CustomEvent('change', {
      detail: {
        delta: this.delta,
        commentContent: this.getCommentContent(),
      },
    }));
  }

  _addPredictedComponent(e) {
    if (e instanceof MouseEvent || e.code === 'Enter') {
      const components = this.shadowRoot.getElementById('componentsInput');
      if (!components) return;
      components.setValue(components.values.concat([this.predictedComponent]));
    }
  }

  // This function exists because <label for="inputId"> doesn't work for custom
  // input elements.
  _clickLabelForCustomInput(e) {
    const target = e.target;
    const forValue = target.getAttribute('for');
    if (forValue) {
      const customInput = this.shadowRoot.querySelector('#' + forValue);
      if (customInput && customInput.focus) {
        customInput.focus();
      }
    }
  }

  _idForField(name) {
    return `${name}Input`;
  }

  _optionsForField(optionsPerEnumField, fieldValueMap, fieldName, phaseName) {
    if (!optionsPerEnumField || !fieldName) return [];
    const key = fieldName.toLowerCase();
    if (!optionsPerEnumField.has(key)) return [];
    const options = [...optionsPerEnumField.get(key)];
    const values = valuesForField(fieldValueMap, fieldName, phaseName);
    values.forEach((v) => {
      const optionExists = options.find(
          (opt) => equalsIgnoreCase(opt.optionName, v));
      if (!optionExists) {
        // Note that enum fields which are not explicitly defined can be set,
        // such as in the case when an issue is moved.
        options.push({optionName: v});
      }
    });
    return options;
  }

  _sendEmailChecked(evt) {
    this.sendEmail = evt.detail.checked;
  }
}

customElements.define('mr-edit-metadata', MrEditMetadata);
