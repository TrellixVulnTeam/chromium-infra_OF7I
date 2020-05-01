// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {connectStore} from 'reducers/base.js';
import * as issueV0 from 'reducers/issueV0.js';
import * as projectV0 from 'reducers/projectV0.js';
import * as userV0 from 'reducers/userV0.js';
import 'elements/framework/mr-star-button/mr-star-button.js';
import 'elements/framework/links/mr-user-link/mr-user-link.js';
import 'elements/framework/links/mr-hotlist-link/mr-hotlist-link.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';
import {pluralize} from 'shared/helpers.js';
import './mr-metadata.js';


/**
 * `<mr-issue-metadata>`
 *
 * The metadata view for a single issue. Contains information such as the owner.
 *
 */
export class MrIssueMetadata extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          box-sizing: border-box;
          padding: 0.25em 8px;
          max-width: 100%;
          display: block;
        }
        h3 {
          display: block;
          font-size: var(--chops-main-font-size);
          margin: 0;
          line-height: 160%;
          width: 40%;
          height: 100%;
          overflow: ellipsis;
          flex-grow: 0;
          flex-shrink: 0;
        }
        a.label {
          color: hsl(120, 100%, 25%);
          text-decoration: none;
        }
        a.label[data-derived] {
          font-style: italic;
        }
        button.linkify {
          display: flex;
          align-items: center;
          text-decoration: none;
          padding: 0.25em 0;
        }
        button.linkify i.material-icons {
          margin-right: 4px;
          font-size: var(--chops-icon-font-size);
        }
        mr-hotlist-link {
          text-overflow: ellipsis;
          overflow: hidden;
          display: block;
          width: 100%;
        }
        .bottom-section-cell, .labels-container {
          padding: 0.5em 4px;
          width: 100%;
          box-sizing: border-box;
        }
        .bottom-section-cell {
          display: flex;
          flex-direction: row;
          flex-wrap: nowrap;
          align-items: flex-start;
        }
        .bottom-section-content {
          max-width: 60%;
        }
        .star-line {
          width: 100%;
          text-align: center;
          display: flex;
          align-items: center;
          justify-content: center;
        }
        mr-star-button {
          margin-right: 4px;
          padding-bottom: 2px;
        }
      `,
    ];
  }

  /** @override */
  render() {
    const hotlistsByRole = this._hotlistsByRole;
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <div class="star-line">
        <mr-star-button
          .issueRef=${this.issueRef}
        ></mr-star-button>
        Starred by ${this.issue.starCount || 0} ${pluralize(this.issue.starCount, 'user')}
      </div>
      <mr-metadata
        aria-label="Issue Metadata"
        .owner=${this.issue.ownerRef}
        .cc=${this.issue.ccRefs}
        .issueStatus=${this.issue.statusRef}
        .components=${this._components}
        .fieldDefs=${this._fieldDefs}
        .mergedInto=${this.mergedInto}
        .modifiedTimestamp=${this.issue.modifiedTimestamp}
      ></mr-metadata>

      <div class="labels-container">
        ${this.issue.labelRefs && this.issue.labelRefs.map((label) => html`
          <a
            title="${_labelTitle(this.labelDefMap, label)}"
            href="/p/${this.issueRef.projectName}/issues/list?q=label:${label.label}"
            class="label"
            ?data-derived=${label.isDerived}
          >${label.label}</a>
          <br>
        `)}
      </div>

      ${this.sortedBlockedOn.length ? html`
        <div class="bottom-section-cell">
          <h3>BlockedOn:</h3>
            <div class="bottom-section-content">
            ${this.sortedBlockedOn.map((issue) => html`
              <mr-issue-link
                .projectName=${this.issueRef.projectName}
                .issue=${issue}
              >
              </mr-issue-link>
              <br />
            `)}
            <button
              class="linkify"
              @click=${this.openViewBlockedOn}
            >
              <i class="material-icons" role="presentation">list</i>
              View details
            </button>
          </div>
        </div>
      `: ''}

      ${this.blocking.length ? html`
        <div class="bottom-section-cell">
          <h3>Blocking:</h3>
          <div class="bottom-section-content">
            ${this.blocking.map((issue) => html`
              <mr-issue-link
                .projectName=${this.issueRef.projectName}
                .issue=${issue}
              >
              </mr-issue-link>
              <br />
            `)}
          </div>
        </div>
      `: ''}

      ${this._userId ? html`
        <div class="bottom-section-cell">
          <h3>Your Hotlists:</h3>
          <div class="bottom-section-content" id="user-hotlists">
            ${this._renderHotlists(hotlistsByRole.user)}
            <button
              class="linkify"
              @click=${this.openUpdateHotlists}
            >
              <i class="material-icons" role="presentation">create</i> Update your hotlists
            </button>
          </div>
        </div>
      `: ''}

      ${hotlistsByRole.participants.length ? html`
        <div class="bottom-section-cell">
          <h3>Participant's Hotlists:</h3>
          <div class="bottom-section-content">
            ${this._renderHotlists(hotlistsByRole.participants)}
          </div>
        </div>
      ` : ''}

      ${hotlistsByRole.others.length ? html`
        <div class="bottom-section-cell">
          <h3>Other Hotlists:</h3>
          <div class="bottom-section-content">
            ${this._renderHotlists(hotlistsByRole.others)}
          </div>
        </div>
      ` : ''}
    `;
  }

  /**
   * Helper to render hotlists.
   * @param {Array<Hotlist>} hotlists
   * @return {Array<TemplateResult>}
   * @private
   */
  _renderHotlists(hotlists) {
    return hotlists.map((hotlist) => html`
      <mr-hotlist-link .hotlist=${hotlist}></mr-hotlist-link>
    `);
  }

  /** @override */
  static get properties() {
    return {
      issue: {type: Object},
      issueRef: {type: Object},
      projectConfig: String,
      user: {type: Object},
      issueHotlists: {type: Array},
      blocking: {type: Array},
      sortedBlockedOn: {type: Array},
      relatedIssues: {type: Object},
      labelDefMap: {type: Object},
      _components: {type: Array},
      _fieldDefs: {type: Array},
      _type: {type: String},
    };
  }

  /** @override */
  stateChanged(state) {
    this.issue = issueV0.viewedIssue(state);
    this.issueRef = issueV0.viewedIssueRef(state);
    this.user = userV0.currentUser(state);
    this.projectConfig = projectV0.viewedConfig(state);
    this.blocking = issueV0.blockingIssues(state);
    this.sortedBlockedOn = issueV0.sortedBlockedOn(state);
    this.mergedInto = issueV0.mergedInto(state);
    this.relatedIssues = issueV0.relatedIssues(state);
    this.issueHotlists = issueV0.hotlists(state);
    this.labelDefMap = projectV0.labelDefMap(state);
    this._components = issueV0.components(state);
    this._fieldDefs = issueV0.fieldDefs(state);
    this._type = issueV0.type(state);
  }

  /**
   * @return {string|number} The current user's userId.
   * @private
   */
  get _userId() {
    return this.user && this.user.userId;
  }

  /**
   * @return {Object.<string, Array<Hotlist>>}
   * @private
   */
  get _hotlistsByRole() {
    const issueHotlists = this.issueHotlists;
    const owner = this.issue && this.issue.ownerRef;
    const cc = this.issue && this.issue.ccRefs;

    const hotlists = {
      user: [],
      participants: [],
      others: [],
    };
    (issueHotlists || []).forEach((hotlist) => {
      if (hotlist.ownerRef.userId === this._userId) {
        hotlists.user.push(hotlist);
      } else if (_userIsParticipant(hotlist.ownerRef, owner, cc)) {
        hotlists.participants.push(hotlist);
      } else {
        hotlists.others.push(hotlist);
      }
    });
    return hotlists;
  }

  /**
   * Opens dialog for updating ths issue's hotlists.
   * @fires CustomEvent#open-dialog
   */
  openUpdateHotlists() {
    this.dispatchEvent(new CustomEvent('open-dialog', {
      bubbles: true,
      composed: true,
      detail: {
        dialogId: 'update-issue-hotlists',
      },
    }));
  }

  /**
   * Opens dialog with detailed view of blocked on issues.
   * @fires CustomEvent#open-dialog
   */
  openViewBlockedOn() {
    this.dispatchEvent(new CustomEvent('open-dialog', {
      bubbles: true,
      composed: true,
      detail: {
        dialogId: 'reorder-related-issues',
      },
    }));
  }
}

/**
 * @param {UserRef} user
 * @param {UserRef} owner
 * @param {Array<UserRef>} cc
 * @return {boolean} Whether a given user is a participant of
 *   a given hotlist attached to an issue. Used to sort hotlists into
 *   "My hotlists" and "Other hotlists".
 * @private
 */
function _userIsParticipant(user, owner, cc) {
  if (owner && owner.userId === user.userId) {
    return true;
  }
  return cc && cc.some((ccUser) => ccUser && ccUser.userId === user.userId);
}

/**
 * @param {Map.<string, LabelDef>} labelDefMap
 * @param {LabelDef} label
 * @return {string} Tooltip shown to the user when hovering over a
 *   given label.
 * @private
 */
function _labelTitle(labelDefMap, label) {
  if (!label) return '';
  let docstring = '';
  const key = label.label.toLowerCase();
  if (labelDefMap && labelDefMap.has(key)) {
    docstring = labelDefMap.get(key).docstring;
  }
  return (label.isDerived ? 'Derived: ' : '') + label.label +
    (docstring ? ` = ${docstring}` : '');
}

customElements.define('mr-issue-metadata', MrIssueMetadata);
