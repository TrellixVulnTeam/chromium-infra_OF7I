// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import 'elements/chops/chops-snackbar/chops-snackbar.js';
import {store, connectStore} from 'reducers/base.js';
import * as issue from 'reducers/issue.js';
import * as project from 'reducers/project.js';
import * as ui from 'reducers/ui.js';
import {arrayToEnglish} from 'shared/helpers.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';
import './mr-edit-metadata.js';
import 'shared/typedef.js';

import ClientLogger from 'monitoring/client-logger.js';

/**
 * `<mr-edit-issue>`
 *
 * Edit form for a single issue. Wraps <mr-edit-metadata>.
 *
 */
export class MrEditIssue extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
    ];
  }

  /** @override */
  render() {
    const issue = this.issue || {};
    let blockedOnRefs = issue.blockedOnIssueRefs || [];
    if (issue.danglingBlockedOnRefs && issue.danglingBlockedOnRefs.length) {
      blockedOnRefs = blockedOnRefs.concat(issue.danglingBlockedOnRefs);
    }

    let blockingRefs = issue.blockingIssueRefs || [];
    if (issue.danglingBlockingRefs && issue.danglingBlockingRefs.length) {
      blockingRefs = blockingRefs.concat(issue.danglingBlockingRefs);
    }

    return html`
      <h2 id="makechanges" class="medium-heading">
        <a href="#makechanges">Add a comment and make changes</a>
      </h2>
      <chops-snackbar ?hidden=${!this._issueUpdated}>
        Your comment was added.
      </chops-snackbar>
      <mr-edit-metadata
        formName="Issue Edit"
        .ownerName=${this._ownerDisplayName(this.issue.ownerRef)}
        .cc=${issue.ccRefs}
        .status=${issue.statusRef && issue.statusRef.status}
        .statuses=${this._availableStatuses(this.projectConfig.statusDefs, this.issue.statusRef)}
        .summary=${issue.summary}
        .components=${issue.componentRefs}
        .fieldDefs=${this._fieldDefs}
        .fieldValues=${issue.fieldValues}
        .blockedOn=${blockedOnRefs}
        .blocking=${blockingRefs}
        .mergedInto=${issue.mergedIntoIssueRef}
        .labelNames=${this._labelNames}
        .derivedLabels=${this._derivedLabels}
        .error=${this.updateError && (this.updateError.description || this.updateError.message)}
        ?saving=${this.updatingIssue}
        @save=${this.save}
        @discard=${this.reset}
        @change=${this._onChange}
      ></mr-edit-metadata>
    `;
  }

  /** @override */
  static get properties() {
    return {
      /**
       * All comments, including descriptions.
       */
      comments: {
        type: Array,
      },
      /**
       * The issue being updated.
       */
      issue: {
        type: Object,
      },
      /**
       * The issueRef for the currently viewed issue.
       */
      issueRef: {
        type: Object,
      },
      /**
       * The config of the currently viewed project.
       */
      projectConfig: {
        type: Object,
      },
      /**
       * Whether the issue is currently being updated.
       */
      updatingIssue: {
        type: Boolean,
      },
      /**
       * An error response, if one exists.
       */
      updateError: {
        type: Object,
      },
      /**
       * Hash from the URL, used to support the 'r' hot key for making changes.
       */
      focusId: {
        type: String,
      },
      _issueUpdated: {
        type: Boolean,
      },
      _awaitingSave: {
        type: Boolean,
      },
      _fieldDefs: {
        type: Array,
      },
    };
  }

  /** @override */
  constructor() {
    super();

    this.clientLogger = new ClientLogger('issues');
  }

  /** @override */
  stateChanged(state) {
    this.issue = issue.issue(state);
    this.issueRef = issue.issueRef(state);
    this.comments = issue.comments(state);
    this.projectConfig = project.project(state).config;
    this.updatingIssue = issue.requests(state).update.requesting;
    this.updateError = issue.requests(state).update.error;
    this.focusId = ui.focusId(state);
    this._fieldDefs = issue.fieldDefs(state);
  }

  /** @override */
  updated(changedProperties) {
    if (this.focusId && changedProperties.has('focusId')) {
      // TODO(zhangtiff): Generalize logic to focus elements based on ID
      // to a reuseable class mixin.
      if (this.focusId.toLowerCase() === 'makechanges') {
        this.focus();
      }
    }

    if (this.issue && changedProperties.has('issue')) {
      if (this._awaitingSave) {
        // This is the first update cycle after an issue update has finished
        // saving.
        this._awaitingSave = false;
        this._issueUpdated = true;
        this.reset();
      }
    }

    if (changedProperties.has('_issueUpdated') && this._issueUpdated) {
      // This case runs after the update cycle when the snackbar telling the
      // user their issue has been updates shows up.
      if (this.clientLogger.started('issue-update')) {
        this.clientLogger.logEnd('issue-update', 'computer-time', 120 * 1000);
      }
    }
  }

  /**
   * Resets all form fields to their initial values.
   */
  reset() {
    const form = this.shadowRoot.querySelector('mr-edit-metadata');
    if (!form) return;
    form.reset();
  }

  /**
   * Dispatches an action to save issue changes on the server.
   */
  async save() {
    const form = this.shadowRoot.querySelector('mr-edit-metadata');
    if (!form) return;

    const delta = form.delta;
    if (!allowRemovedRestrictions(delta.labelRefsRemove)) {
      return;
    }

    const message = {
      issueRef: this.issueRef,
      delta: delta,
      commentContent: form.getCommentContent(),
      sendEmail: form.sendEmail,
    };

    // Add files to message.
    const uploads = await form.getAttachments();

    if (uploads && uploads.length) {
      message.uploads = uploads;
    }

    if (message.commentContent || message.delta || message.uploads) {
      this.clientLogger.logStart('issue-update', 'computer-time');

      this._issueUpdated = false;
      this._awaitingSave = true;

      store.dispatch(issue.update(message));
    }
  }

  /**
   * Focuses the edit form in response to the 'r' hotkey.
   */
  focus() {
    const editHeader = this.shadowRoot.querySelector('#makechanges');
    editHeader.scrollIntoView();

    const editForm = this.shadowRoot.querySelector('mr-edit-metadata');
    editForm.focus();
  }

  /**
   * Concatenates all comment text into one long string to be sent to the
   * component prediction API.
   * @return {string} All comments in an issue with their text content
   *   concatenated.
   */
  get _commentsText() {
    return (this.comments || []).map(
        (comment) => comment.content).join('\n').trim();
  }

  /**
   * Turns all LabelRef Objects attached to an issue into an Array of strings
   * containing only the names of those labels that aren't derived.
   * @return {Array<string>} Array of label names.
   */
  get _labelNames() {
    if (!this.issue || !this.issue.labelRefs) return [];
    const labels = this.issue.labelRefs;
    return labels.filter((l) => !l.isDerived).map((l) => l.label);
  }

  /**
   * Finds only the derived labels attached to an issue and returns only
   * their names.
   * @return {Array<string>} Array of label names.
   */
  get _derivedLabels() {
    if (!this.issue || !this.issue.labelRefs) return [];
    const labels = this.issue.labelRefs;
    return labels.filter((l) => l.isDerived).map((l) => l.label);
  }

  /**
   * Gets the displayName of the owner. Only uses the displayName if a
   * userId also exists in the ref.
   * @param {UserRef} ownerRef The owner of the issue.
   * @return {string} The name of the owner for the edited issue.
   */
  _ownerDisplayName(ownerRef) {
    return (ownerRef && ownerRef.userId) ? ownerRef.displayName : '';
  }

  /**
   * Dispatches an action against the server to run "issue presubmit", a feature
   * that warns the user about issue changes that violate configured rules.
   * @param {Object} issueDelta Changes currently present in the edit form.
   */
  _presubmitIssue(issueDelta) {
    if (Object.keys(issueDelta).length) {
      const message = {
        issueDelta,
        issueRef: this.issueRef,
      };
      store.dispatch(issue.presubmit(message));
    }
  }

  /**
   * Dispatches an action to predict a component based on comment text.
   * @param {Object} issueDelta Changes currently present in the edit form.
   * @param {string} newCommentContent The text for the comment the user
   *   typed into the edit form.
   */
  _predictComponent(issueDelta, newCommentContent) {
    // Component prediction is only enabled on Chromium issues.
    if (this.issueRef.projectName !== 'chromium') return;

    let text = this._commentsText;
    if (issueDelta.summary) {
      text += '\n' + issueDelta.summary;
    } else if (this.issue.summary) {
      text += '\n' + this.issue.summary;
    }
    if (newCommentContent) {
      text += '\n' + newCommentContent.trim();
    }

    store.dispatch(issue.predictComponent(this.issueRef.projectName, text));
  }

  /**
   * Form change handler that runs presubmit and component prediction on the
   * form.
   * @param {Event} evt
   */
  _onChange(evt) {
    this._presubmitIssue(evt.detail.delta);
    this._predictComponent(evt.detail.delta, evt.detail.commentContent);
  }

  /**
   * Creates the list of statuses that the user sees in the status dropdown.
   * @param {Array<StatusDef>} statusDefsArg The project configured StatusDefs.
   * @param {StatusRef} currentStatusRef The status that the issue currently
   *   uses. Note that Monorail supports free text statuses that do not exist in
   *   a project config. Because of this, currentStatusRef may not exist in
   *   statusDefsArg.
   * @return {Array<StatusRef|StatusDef>} Array of statuses a user can edit this
   *   issue to have.
   */
  _availableStatuses(statusDefsArg, currentStatusRef) {
    let statusDefs = statusDefsArg || [];
    statusDefs = statusDefs.filter((status) => !status.deprecated);
    if (!currentStatusRef || statusDefs.find(
        (status) => status.status === currentStatusRef.status)) {
      return statusDefs;
    }
    return [currentStatusRef, ...statusDefs];
  }
}

/**
 * Asks the user for confirmation when they try to remove retriction labels.
 * eg. Restrict-View-Google.
 * @param {Array<LabelRef>} labelRefsRemoved The labels a user is removing
 *   from this issue.
 * @return {boolean} Whether removing these labels is okay. ie: true if there
 *   are either no restrictions being removed or if the user approved the
 *   removal of the restrictions.
 */
export function allowRemovedRestrictions(labelRefsRemoved) {
  if (!labelRefsRemoved) return true;
  const removedRestrictions = labelRefsRemoved
      .map(({label}) => label)
      .filter((label) => label.toLowerCase().startsWith('restrict-'));
  const removeRestrictionsMessage =
    'You are removing these restrictions:\n' +
    arrayToEnglish(removedRestrictions) + '\n' +
    'This might allow more people to access this issue. Are you sure?';
  return !removedRestrictions.length || confirm(removeRestrictionsMessage);
}

customElements.define('mr-edit-issue', MrEditIssue);
