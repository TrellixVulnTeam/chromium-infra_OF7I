// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import 'elements/chops/chops-dialog/chops-dialog.js';
import {store, connectStore} from 'reducers/base.js';
import * as issueV0 from 'reducers/issueV0.js';
import * as projectV0 from 'reducers/projectV0.js';
import '../mr-approval-card/mr-approval-card.js';
import {valueForField, valuesForField} from 'shared/metadata-helpers.js';
import 'elements/issue-detail/metadata/mr-edit-metadata/mr-edit-metadata.js';
import 'elements/issue-detail/metadata/mr-metadata/mr-field-values.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';

const TARGET_PHASE_MILESTONE_MAP = {
  'Beta': 'feature_freeze',
  'Stable-Exp': 'final_beta_cut',
  'Stable': 'stable_cut',
  'Stable-Full': 'stable_cut',
};

const APPROVED_PHASE_MILESTONE_MAP = {
  'Beta': 'earliest_beta',
  'Stable-Exp': 'final_beta',
  'Stable': 'stable_date',
  'Stable-Full': 'stable_date',
};

// The following milestones are unique to ios.
const IOS_APPROVED_PHASE_MILESTONE_MAP = {
  'Beta': 'earliest_beta_ios',
};

// See monorail:4692 and the use of PHASES_WITH_MILESTONES
// in tracker/issueentry.py
const PHASES_WITH_MILESTONES = ['Beta', 'Stable', 'Stable-Exp', 'Stable-Full'];

/**
 * `<mr-phase>`
 *
 * This is the component for a single phase.
 *
 */
export class MrPhase extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          display: block;
        }
        chops-dialog {
          --chops-dialog-theme: {
            width: 500px;
            max-width: 100%;
          };
        }
        h2 {
          margin: 0;
          font-size: var(--chops-large-font-size);
          font-weight: normal;
          padding: 0.5em 8px;
          box-sizing: border-box;
          display: flex;
          align-items: center;
          flex-direction: row;
          justify-content: space-between;
        }
        h2 em {
          margin-left: 16px;
          font-size: var(--chops-main-font-size);
        }
        .chip {
          display: inline-block;
          font-size: var(--chops-main-font-size);
          padding: 0.25em 8px;
          margin: 0 2px;
          border-radius: 16px;
          background: var(--chops-blue-gray-50);
        }
        .phase-edit {
          padding: 0.1em 8px;
        }
      `,
    ];
  }

  /** @override */
  render() {
    const isPhaseWithMilestone = PHASES_WITH_MILESTONES.includes(
        this.phaseName);
    const noApprovals = !this.approvals || !this.approvals.length;
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <h2>
        <div>
          Approvals<span ?hidden=${!this.phaseName || !this.phaseName.length}>:
            ${this.phaseName}
          </span>
          ${isPhaseWithMilestone ? html`${this.fieldDefs &&
              this.fieldDefs.map((field) => this._renderPhaseField(field))}
            <em ?hidden=${!this._nextDate}>
              ${this._dateDescriptor}
              <chops-timestamp .timestamp=${this._nextDate}></chops-timestamp>
            </em>
            <em ?hidden=${!this._nextUniqueiOSDate}>
              <b>iOS</b> ${this._dateDescriptor}
              <chops-timestamp .timestamp=${this._nextUniqueiOSDate}
              ></chops-timestamp>
            </em>
          `: ''}
        </div>
        ${isPhaseWithMilestone ? html`
          <chops-button @click=${this.edit} class="de-emphasized phase-edit">
            <i class="material-icons" role="presentation">create</i>
            Edit
          </chops-button>
        `: ''}
      </h2>
      ${this.approvals && this.approvals.map((approval) => html`
        <mr-approval-card
          .approvers=${approval.approverRefs}
          .setter=${approval.setterRef}
          .fieldName=${approval.fieldRef.fieldName}
          .phaseName=${this.phaseName}
          .statusEnum=${approval.status}
          .survey=${approval.survey}
          .surveyTemplate=${approval.surveyTemplate}
          .urls=${approval.urls}
          .labels=${approval.labels}
          .users=${approval.users}
        ></mr-approval-card>
      `)}
      ${noApprovals ? html`No tasks for this phase.` : ''}
      <!-- TODO(ehmaldonado): Move to /issue-detail/dialogs -->
      <chops-dialog id="editPhase" aria-labelledby="phaseDialogTitle">
        <h3 id="phaseDialogTitle" class="medium-heading">
          Editing phase: ${this.phaseName}
        </h3>
        <mr-edit-metadata
          id="metadataForm"
          class="edit-actions-right"
          .formName=${this.phaseName}
          .fieldDefs=${this.fieldDefs}
          .phaseName=${this.phaseName}
          ?disabled=${this._updatingIssue}
          .error=${this._updateIssueError && this._updateIssueError.description}
          @save=${this.save}
          @discard=${this.cancel}
          isApproval
          disableAttachments
        ></mr-edit-metadata>
      </chops-dialog>
    `;
  }

  /**
   *
   * @param {FieldDef} field The field to be rendered.
   * @return {TemplateResult}
   * @private
   */
  _renderPhaseField(field) {
    const values = valuesForField(this._fieldValueMap, field.fieldRef.fieldName,
        this.phaseName);
    return html`
      <div class="chip">
        ${field.fieldRef.fieldName}:
        <mr-field-values
          .name=${field.fieldRef.fieldName}
          .type=${field.fieldRef.type}
          .values=${values}
          .projectName=${this.issueRef.projectName}
        ></mr-field-values>
      </div>
    `;
  }

  /** @override */
  static get properties() {
    return {
      issue: {type: Object},
      issueRef: {type: Object},
      phaseName: {type: String},
      approvals: {type: Array},
      fieldDefs: {type: Array},

      _updatingIssue: {type: Boolean},
      _updateIssueError: {type: Object},
      _fieldValueMap: {type: Object},
      _milestoneData: {type: Object},
      _isFetchingMilestone: {type: Boolean},
      _fetchedMilestone: {type: String},
    };
  }

  /** @override */
  constructor() {
    super();

    this.issue = {};
    this.issueRef = {};
    this.phaseName = '';
    this.approvals = [];
    this.fieldDefs = [];

    this._updatingIssue = false;
    this._updateIssueError = undefined;

    // A response Object from
    // https://chromiumdash.appspot.com/fetch_milestone_schedule?mstone=xx
    this._milestoneData = {};
    this._isFetchingMilestone = false;
    this._fetchedMilestone = undefined;
    /**
     * @type {Promise} Used for testing to allow waiting for milestone
     *   fetch operations to finish.
     */
    this._fetchMilestoneComplete = undefined;
  }

  /** @override */
  stateChanged(state) {
    this.issue = issueV0.viewedIssue(state);
    this.issueRef = issueV0.viewedIssueRef(state);
    this.fieldDefs = projectV0.fieldDefsForPhases(state);
    this._updatingIssue = issueV0.requests(state).update.requesting;
    this._updateIssueError = issueV0.requests(state).update.error;
    this._fieldValueMap = issueV0.fieldValueMap(state);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('issue')) {
      this.reset();
    }
    if (changedProperties.has('_updatingIssue')) {
      if (!this._updatingIssue && !this._updateIssueError) {
        // Close phase edit modal only after a request finishes without errors.
        this.cancel();
      }
    }

    if (!this._isFetchingMilestone) {
      const milestoneToFetch = this._milestoneToFetch;
      if (milestoneToFetch && this._fetchedMilestone !== milestoneToFetch) {
        this._fetchMilestoneComplete = this.fetchMilestoneData(
            milestoneToFetch);
      }
    }
  }

  /**
   * Makes an XHR request to Chromium Dash to find Chrome-specific launch data.
   * eg. when certain Chrome milestones are planned for release.
   * @param {string} milestone A string containing a Chrome milestone number.
   * @return {Promise<void>}
   */
  async fetchMilestoneData(milestone) {
    this._isFetchingMilestone = true;

    try {
      const resp = await window.fetch(
          `https://chromiumdash.appspot.com/fetch_milestone_schedule?mstone=${
            milestone}`);
      this._milestoneData = await resp.json();
    } catch (error) {
      console.error(`Error when fetching milestone data: ${error}`);
    }
    this._fetchedMilestone = milestone;
    this._isFetchingMilestone = false;
  }

  /**
   * Opens the phase editing dialog when the user clicks the edit button.
   */
  edit() {
    this.reset();
    this.shadowRoot.querySelector('#editPhase').open();
  }

  /**
   * Stops editing the phase.
   */
  cancel() {
    this.shadowRoot.querySelector('#editPhase').close();
    this.reset();
  }

  /**
   * Resets the edit form to its default values.
   */
  reset() {
    const form = this.shadowRoot.querySelector('#metadataForm');
    form.reset();
  }

  /**
   * Saves the changes the user has made.
   */
  save() {
    const form = this.shadowRoot.querySelector('#metadataForm');
    const delta = form.delta;

    if (delta.fieldValsAdd) {
      delta.fieldValsAdd = delta.fieldValsAdd.map(
          (fv) => Object.assign({phaseRef: {phaseName: this.phaseName}}, fv));
    }
    if (delta.fieldValsRemove) {
      delta.fieldValsRemove = delta.fieldValsRemove.map(
          (fv) => Object.assign({phaseRef: {phaseName: this.phaseName}}, fv));
    }

    const message = {
      issueRef: this.issueRef,
      delta: delta,
      sendEmail: form.sendEmail,
      commentContent: form.getCommentContent(),
    };

    if (message.commentContent || message.delta) {
      store.dispatch(issueV0.update(message));
    }
  }

  /**
   * Shows the next relevant Chrome Milestone date for this phase. Depending
   * on the M-Target, M-Approved, or M-Launched values, this date means
   * different things.
   * @return {number} Unix timestamp in seconds.
   * @private
   */
  get _nextDate() {
    const phaseName = this.phaseName;
    const status = this._status;
    let data = this._milestoneData && this._milestoneData.mstones;
    // Data pulled from https://chromiumdash.appspot.com/fetch_milestone_schedule?mstone=xx
    if (!phaseName || !status || !data || !data.length) return 0;
    data = data[0];

    let key = TARGET_PHASE_MILESTONE_MAP[phaseName];
    if (['Approved', 'Launched'].includes(status)) {
      const osValues = this._fieldValueMap.get('OS');
      // If iOS is the only OS and the phase is one where iOS has unique
      // milestones, the only date we show should be this._nextUniqueiOSDate.
      if (osValues && osValues.every((os) => {
        return os === 'iOS';
      }) && phaseName in IOS_APPROVED_PHASE_MILESTONE_MAP) {
        return 0;
      }
      key = APPROVED_PHASE_MILESTONE_MAP[phaseName];
    }
    if (!key || !(key in data)) return 0;
    return Math.floor((new Date(data[key])).getTime() / 1000);
  }

  /**
   * For issues where iOS is the OS, this function finds the relevant iOS
   * launch date.
   * @return {number} Unix timestamp in seconds.
   * @private
   */
  get _nextUniqueiOSDate() {
    const phaseName = this.phaseName;
    const status = this._status;
    let data = this._milestoneData && this._milestoneData.mstones;
    // Data pull from https://chromiumdash.appspot.com/fetch_milestone_schedule?mstone=xx
    if (!phaseName || !status || !data || !data.length) return 0;
    data = data[0];

    const osValues = this._fieldValueMap.get('OS');
    if (['Approved', 'Launched'].includes(status) &&
        osValues && osValues.includes('iOS')) {
      const key = IOS_APPROVED_PHASE_MILESTONE_MAP[phaseName];
      if (key) {
        return Math.floor((new Date(data[key])).getTime() / 1000);
      }
    }
    return 0;
  }

  /**
   * Depending on what kind of date we're showing, we want to include
   * different text to describe the date.
   * @return {string}
   * @private
   */
  get _dateDescriptor() {
    const status = this._status;
    if (status === 'Approved') {
      return 'Launching on ';
    } else if (status === 'Launched') {
      return 'Launched on ';
    }
    return 'Due by ';
  }

  /**
   * The Chrome-specific status of a gate, computed from M-Approved,
   * M-Launched, and M-Target fields.
   * @return {string}
   * @private
   */
  get _status() {
    const target = this._targetMilestone;
    const approved = this._approvedMilestone;
    const launched = this._launchedMilestone;
    if (approved >= target) {
      if (launched >= approved) {
        return 'Launched';
      }
      return 'Approved';
    }
    return 'Target';
  }

  /**
   * The Chrome Milestone that this phase was approved for.
   * @return {string}
   * @private
   */
  get _approvedMilestone() {
    return valueForField(this._fieldValueMap, 'M-Approved', this.phaseName);
  }

  /**
   * The Chrome Milestone that this phase was launched on.
   * @return {string}
   * @private
   */
  get _launchedMilestone() {
    return valueForField(this._fieldValueMap, 'M-Launched', this.phaseName);
  }

  /**
   * The Chrome Milestone that this phase is targeting.
   * @return {string}
   * @private
   */
  get _targetMilestone() {
    return valueForField(this._fieldValueMap, 'M-Target', this.phaseName);
  }

  /**
   * The Chrome Milestone that's used to decide what date to show the user.
   * @return {string}
   * @private
   */
  get _milestoneToFetch() {
    const target = Number.parseInt(this._targetMilestone) || 0;
    const approved = Number.parseInt(this._approvedMilestone) || 0;
    const launched = Number.parseInt(this._launchedMilestone) || 0;

    const latestMilestone = Math.max(target, approved, launched);
    return latestMilestone > 0 ? `${latestMilestone}` : '';
  }
}


customElements.define('mr-phase', MrPhase);
