// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';

import 'elements/chops/chops-button/chops-button.js';
import 'elements/framework/mr-upload/mr-upload.js';
import 'elements/framework/mr-star/mr-issue-star.js';
import 'elements/chops/chops-checkbox/chops-checkbox.js';
import 'elements/chops/chops-chip/chops-chip.js';
import 'elements/framework/mr-error/mr-error.js';
import 'elements/framework/mr-warning/mr-warning.js';
import 'elements/help/mr-cue/mr-cue.js';
import 'react/mr-react-autocomplete.tsx';
import {cueNames} from 'elements/help/mr-cue/cue-helpers.js';
import {store, connectStore} from 'reducers/base.js';
import {UserInputError} from 'shared/errors.js';
import {fieldTypes} from 'shared/issue-fields.js';
import {displayNameToUserRef, labelStringToRef, componentStringToRef,
  componentRefsToStrings, issueStringToRef, issueStringToBlockingRef,
  issueRefToString, issueRefsToStrings, filteredUserDisplayNames,
  valueToFieldValue, fieldDefToName,
} from 'shared/convertersV0.js';
import {arrayDifference, isEmptyObject, equalsIgnoreCase} from 'shared/helpers.js';
import {NON_EDITING_KEY_EVENTS} from 'shared/dom-helpers.js';
import * as issueV0 from 'reducers/issueV0.js';
import * as permissions from 'reducers/permissions.js';
import * as projectV0 from 'reducers/projectV0.js';
import * as userV0 from 'reducers/userV0.js';
import * as ui from 'reducers/ui.js';
import '../mr-edit-field/mr-edit-field.js';
import '../mr-edit-field/mr-edit-status.js';
import {ISSUE_EDIT_PERMISSION, ISSUE_EDIT_SUMMARY_PERMISSION,
  ISSUE_EDIT_STATUS_PERMISSION, ISSUE_EDIT_OWNER_PERMISSION,
  ISSUE_EDIT_CC_PERMISSION,
} from 'shared/consts/permissions.js';
import {fieldDefsWithGroup, fieldDefsWithoutGroup, valuesForField,
  HARDCODED_FIELD_GROUPS} from 'shared/metadata-helpers.js';
import {renderMarkdown, shouldRenderMarkdown} from 'shared/md-helper.js';
import {unsafeHTML} from 'lit-html/directives/unsafe-html.js';
import {MD_PREVIEW_STYLES} from 'shared/shared-styles.js';



/**
 * `<mr-edit-metadata>`
 *
 * Editing form for either an approval or the overall issue.
 *
 */
export class MrEditMetadata extends connectStore(LitElement) {
  /** @override */
  render() {
    return html`
      <style>
        ${MD_PREVIEW_STYLES}
        mr-edit-metadata {
          display: block;
          font-size: var(--chops-main-font-size);
        }
        mr-edit-metadata.edit-actions-right .edit-actions {
          flex-direction: row-reverse;
          text-align: right;
        }
        mr-edit-metadata.edit-actions-right .edit-actions chops-checkbox {
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
        mr-issue-star {
          margin-right: 4px;
        }
      </style>
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
        ${(this._renderMarkdown)
           ? html`
          <div class="markdown-preview preview-height-comment">
            ${unsafeHTML(renderMarkdown(this.getCommentContent()))}
          </div>`: ''}
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
        <mr-issue-star
          .issueRef=${this.issueRef}
        ></mr-issue-star>
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
      <mr-react-autocomplete
        label="ownerInput"
        vocabularyName="owner"
        .placeholder=${ownerPresubmit.placeholder}
        .value=${this._values.owner}
        .onChange=${this._changeHandlers.owner}
      ></mr-react-autocomplete>
    `;
  }

  /**
   * @return {TemplateResult}
   * @private
   */
  _renderCC() {
    return html`
      <label for="ccInput" @click=${this._clickLabelForCustomInput}>CC:</label>
      <mr-react-autocomplete
        label="ccInput"
        vocabularyName="member"
        .multiple=${true}
        .fixedValues=${this._derivedCCs}
        .value=${this._values.cc}
        .onChange=${this._changeHandlers.cc}
      ></mr-react-autocomplete>
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
      <mr-react-autocomplete
        label="componentsInput"
        vocabularyName="component"
        .multiple=${true}
        .value=${this._values.components}
        .onChange=${this._changeHandlers.components}
      ></mr-react-autocomplete>
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
      <mr-react-autocomplete
        label="labelsInput"
        vocabularyName="label"
        .multiple=${true}
        .fixedValues=${this.derivedLabels}
        .value=${this._values.labels}
        .onChange=${this._changeHandlers.labels}
      ></mr-react-autocomplete>
    `;
  }

  /**
   * @return {TemplateResult}
   * @param {FieldDef} field The custom field beinf rendered.
   * @private
   */
  _renderCustomField(field) {
    if (!field || !field.fieldRef) return '';
    const userCanEdit = this._userCanEdit(field);
    const {fieldRef, isNiche, docstring, isMultivalued} = field;
    const isHidden = (!this.showNicheFields && isNiche) || !userCanEdit;

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
      _permissions: {type: Array},
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
      fieldValueMap: {type: Object},
      issueType: {type: String},
      optionsPerEnumField: {type: String},
      fieldGroups: {type: Object},
      prefs: {type: Object},
      saving: {type: Boolean},
      isDirty: {type: Boolean},
      _values: {type: Object},
      _initialValues: {type: Object},
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

    this._permissions = {};
    this.saving = false;
    this.isDirty = false;
    this.prefs = {};
    this._values = {};
    this._initialValues = {};

    this._changeHandlers = {
      owner: this._onChange.bind(this, 'owner'),
      cc: this._onChange.bind(this, 'cc'),
      components: this._onChange.bind(this, 'components'),
      labels: this._onChange.bind(this, 'labels'),
    };
  }

  /** @override */
  createRenderRoot() {
    return this;
  }

  /** @override */
  firstUpdated() {
    this.hasRendered = true;
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('ownerName') || changedProperties.has('cc')
        || changedProperties.has('components')
        || changedProperties.has('labels')) {
      this._initialValues.owner = this.ownerName;
      this._initialValues.cc = this._ccNames;
      this._initialValues.components = componentRefsToStrings(this.components);
      this._initialValues.labels = this.labelNames;

      this._values = {...this._initialValues};
    }
  }

  /**
   * Getter for checking if the user has Markdown enabled.
   * @return {boolean} Whether Markdown preview should be rendered or not.
   */
  get _renderMarkdown() {
    if (!this.getCommentContent()) {
      return false;
    }
    const enabled = this.prefs.get('render_markdown');
    return shouldRenderMarkdown({project: this.projectName, enabled});
  }

  /**
   * @return {boolean} Whether the "Save changes" button is disabled.
   */
  get disabled() {
    return !this.isDirty || this.saving;
  }

  /**
   * Set isDirty to a property instead of only using a getter to cause
   * lit-element to re-render when dirty state change.
   */
  _updateIsDirty() {
    if (!this.hasRendered) return;

    const commentContent = this.getCommentContent();
    const attachmentsElement = this.querySelector('mr-upload');
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

  /**
   * @return {Array<string>}
   */
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
    this._permissions = permissions.byName(state);
    this.presubmitResponse = issueV0.presubmitResponse(state);
    this.projectConfig = projectV0.viewedConfig(state);
    this.projectName = issueV0.viewedIssueRef(state).projectName;
    this.issuePermissions = issueV0.permissions(state);
    this.optionsPerEnumField = projectV0.optionsPerEnumField(state);
    // Access boolean value from allStarredIssues
    const starredIssues = issueV0.starredIssues(state);
    this.isStarred = starredIssues.has(issueRefToString(this.issueRef));
    this.prefs = userV0.prefs(state);
  }

  /** @override */
  disconnectedCallback() {
    super.disconnectedCallback();

    store.dispatch(ui.reportDirtyForm(this.formName, false));
  }

  /**
   * Resets the edit form values to their default values.
   */
  reset() {
    this._values = {...this._initialValues};

    const form = this.querySelector('#editForm');
    if (!form) return;

    form.reset();
    const statusInput = this.querySelector('#statusInput');
    if (statusInput) {
      statusInput.reset();
    }

    // Since custom elements containing <input> elements have the inputs
    // wrapped in ShadowDOM, those inputs don't get reset with the rest of
    // the form. Haven't been able to figure out a way to replicate form reset
    // behavior with custom input elements.
    if (this.isApproval) {
      if (this.hasApproverPrivileges) {
        const approversInput = this.querySelector(
            '#approversInput');
        if (approversInput) {
          approversInput.reset();
        }
      }
    }
    this.querySelectorAll('mr-edit-field').forEach((el) => {
      el.reset();
    });

    const uploader = this.querySelector('mr-upload');
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
    this.querySelector('#commentText').focus();
  }

  /**
   * Retrieves the value of the comment that the user added from the DOM.
   * @return {string}
   */
  getCommentContent() {
    if (!this.querySelector('#commentText')) {
      return '';
    }
    return this.querySelector('#commentText').value;
  }

  async getAttachments() {
    try {
      return await this.querySelector('mr-upload').loadFiles();
    } catch (e) {
      this.error = `Error while loading file for attachment: ${e.message}`;
    }
  }

  /**
   * @param {FieldDef} field
   * @return {boolean}
   * @private
   */
  _userCanEdit(field) {
    const fieldName = fieldDefToName(this.projectName, field);
    if (!this._permissions[fieldName] ||
        !this._permissions[fieldName].permissions) return false;
    const userPerms = this._permissions[fieldName].permissions;
    return userPerms.includes(permissions.FIELD_DEF_VALUE_EDIT);
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
    let result = {};

    const {projectName, localId} = this.issueRef;

    const statusInput = this.querySelector('#statusInput');
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
        result = {
          ...result,
          ...this._changedValuesDom(
            'approvers', 'approverRefs', displayNameToUserRef),
        };
      }
    } else {
      // TODO(zhangtiff): Consider representing baked-in fields such as owner,
      // cc, and status similarly to custom fields to reduce repeated code.

      if (this._canEditSummary) {
        const summaryInput = this.querySelector('#summaryInput');
        if (summaryInput) {
          const newSummary = summaryInput.value;
          if (newSummary !== this.summary) {
            result.summary = newSummary;
          }
        }
      }

      if (this._values.owner !== this.ownerName) {
        result.ownerRef = displayNameToUserRef(this._values.owner);
      }

      result = {
        ...result,
        ...this._changedValuesControlled(
          'cc', 'ccRefs', displayNameToUserRef),
      }

      result = {
        ...result,
        ...this._changedValuesControlled(
          'components', 'compRefs', componentStringToRef),
      }

      result = {
        ...result,
        ...this._changedValuesControlled(
          'labels', 'labelRefs', labelStringToRef),
      }

      if (this._canEditIssue) {
        const blockerAddFn = (refString) =>
          issueStringToBlockingRef({projectName, localId}, refString);
        const blockerRemoveFn = (refString) =>
          issueStringToRef(refString, projectName);
        result = {
          ...result,
          ...this._changedValuesDom(
            'blockedOn', 'blockedOnRefs', blockerAddFn, blockerRemoveFn),
        };
        result = {
          ...result,
          ...this._changedValuesDom(
            'blocking', 'blockingRefs', blockerAddFn, blockerRemoveFn),
        };
      }
    }

    if (this._canEditIssue) {
      const fieldDefs = this.fieldDefs || [];
      fieldDefs.forEach(({fieldRef}) => {
        const {fieldValsAdd = [], fieldValsRemove = []} =
          this._changedValuesDom(fieldRef.fieldName, 'fieldVals',
            valueToFieldValue.bind(null, fieldRef));

        // Because multiple custom fields share the same "fieldVals" key in
        // delta, we hav to make sure to concatenate updated delta values with
        // old delta values.
        if (fieldValsAdd.length) {
          result.fieldValsAdd = [...(result.fieldValsAdd || []),
            ...fieldValsAdd];
        }

        if (fieldValsRemove.length) {
          result.fieldValsRemove = [...(result.fieldValsRemove || []),
            ...fieldValsRemove];
        }
      });
    }

    return result;
  }

  /**
   * Computes delta values for a controlled input.
   * @param {string} fieldName The key in the values property to retrieve data.
   *   from.
   * @param {string} responseKey The key in the delta Object that changes will be
   *   saved in.
   * @param {function(string): any} addFn A function to specify how to format
   *   the message for a given added field.
   * @param {function(string): any} removeFn A function to specify how to format
   *   the message for a given removed field.
   * @return {Object} delta fragment for added and removed values.
   */
  _changedValuesControlled(fieldName, responseKey, addFn, removeFn) {
    const values = this._values[fieldName];
    const initialValues = this._initialValues[fieldName];

    const valuesAdd = arrayDifference(values, initialValues, equalsIgnoreCase);
    const valuesRemove =
      arrayDifference(initialValues, values, equalsIgnoreCase);

    return this._changedValues(valuesAdd, valuesRemove, responseKey, addFn, removeFn);
  }

  /**
   * Gets changes values when reading from a legacy <mr-edit-field> element.
   * @param {string} fieldName Name of the form input we're checking values on.
   * @param {string} responseKey The key in the delta Object that changes will be
   *   saved in.
   * @param {function(string): any} addFn A function to specify how to format
   *   the message for a given added field.
   * @param {function(string): any} removeFn A function to specify how to format
   *   the message for a given removed field.
   * @return {Object} delta fragment for added and removed values.
   */
  _changedValuesDom(fieldName, responseKey, addFn, removeFn) {
    const input = this.querySelector(`#${fieldName}Input`);
    if (!input) return;

    const valuesAdd = input.getValuesAdded();
    const valuesRemove = input.getValuesRemoved();

    return this._changedValues(valuesAdd, valuesRemove, responseKey, addFn, removeFn);
  }

  /**
   * Shared helper function for computing added and removed values for a
   * single field in a delta.
   * @param {Array<string>} valuesAdd The added values. For example, new CCed
   *   users.
   * @param {Array<string>} valuesRemove Values that were removed in this edit.
   * @param {string} responseKey The key in the delta Object that changes will be
   *   saved in.
   * @param {function(string): any} addFn A function to specify how to format
   *   the message for a given added field.
   * @param {function(string): any} removeFn A function to specify how to format
   *   the message for a given removed field.
   * @return {Object} delta fragment for added and removed values.
   */
  _changedValues(valuesAdd, valuesRemove, responseKey, addFn, removeFn) {
    const delta = {};

    if (valuesAdd && valuesAdd.length) {
      delta[responseKey + 'Add'] = valuesAdd.map(addFn);
    }

    if (valuesRemove && valuesRemove.length) {
      delta[responseKey + 'Remove'] = valuesRemove.map(removeFn || addFn);
    }

    return delta;
  }

  /**
   * Generic onChange handler to be bound to each form field.
   * @param {string} key Unique name for the form field we're binding this
   *   handler to. For example, 'owner', 'cc', or the name of a custom field.
   * @param {Event} event
   * @param {string} value The new form value.
   * @param {*} _reason
   */
  _onChange(key, event, value, _reason) {
    this._values = {...this._values, [key]: value};
    this._processChanges(event);
  }

  /**
   * Event handler for running filter rules presubmit logic.
   * @param {Event} e
   */
  _processChanges(e) {
    if (e instanceof KeyboardEvent) {
      if (NON_EDITING_KEY_EVENTS.has(e.key)) return;
    }
    this._updateIsDirty();

    store.dispatch(ui.reportDirtyForm(this.formName, this.isDirty));

    this.dispatchEvent(new CustomEvent('change', {
      detail: {
        delta: this.delta,
        commentContent: this.getCommentContent(),
      },
    }));
  }

  // This function exists because <label for="inputId"> doesn't work for custom
  // input elements.
  _clickLabelForCustomInput(e) {
    const target = e.target;
    const forValue = target.getAttribute('for');
    if (forValue) {
      const customInput = this.querySelector('#' + forValue);
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
