// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// TODO(juliacordero): Handle pRPC errors with a FE page

import {LitElement, html, css} from 'lit-element';
import {store, connectStore} from 'reducers/base.js';
import {shouldWaitForDefaultQuery} from 'shared/helpers.js';
import * as issueV0 from 'reducers/issueV0.js';
import * as projectV0 from 'reducers/projectV0.js';
import * as sitewide from 'reducers/sitewide.js';
import 'elements/framework/links/mr-issue-link/mr-issue-link.js';
import './mr-grid-controls.js';
import './mr-grid.js';

/**
 * <mr-grid-page>
 *
 * Grid page view containing mr-grid and mr-grid-controls.
 * @extends {LitElement}
 */
export class MrGridPage extends connectStore(LitElement) {
  /** @override */
  render() {
    const displayedProgress = this.progress || 0.02;
    const doneLoading = this.progress === 1;
    const noMatches = this.totalIssues === 0 && doneLoading;
    return html`
      <div id="grid-area">
        <mr-grid-controls
          .projectName=${this.projectName}
          .queryParams=${this._queryParams}
          .issueCount=${this.issues.length}>
        </mr-grid-controls>
        ${noMatches ? html`
          <div class="empty-search">
            Your search did not generate any results.
          </div>` : html`
          <progress
            title="${Math.round(displayedProgress * 100)}%"
            value=${displayedProgress}
            ?hidden=${doneLoading}
          ></progress>`}
        <br>
        <mr-grid
          .issues=${this.issues}
          .xField=${this._queryParams.x}
          .yField=${this._queryParams.y}
          .cellMode=${this._queryParams.cells ? this._queryParams.cells : 'tiles'}
          .queryParams=${this._queryParams}
          .projectName=${this.projectName}
        ></mr-grid>
      </div>
    `;
  }

  /** @override */
  static get properties() {
    return {
      projectName: {type: String},
      _queryParams: {type: Object},
      userDisplayName: {type: String},
      issues: {type: Array},
      fields: {type: Array},
      progress: {type: Number},
      totalIssues: {type: Number},
      _presentationConfigLoaded: {type: Boolean},
      /**
       * The current search string the user is querying for.
       * Project default if not specified.
       */
      _currentQuery: {type: String},
      /**
       * The current canned query the user is searching for.
       * Project default if not specified.
       */
      _currentCan: {type: String},
    };
  };

  /** @override */
  constructor() {
    super();
    this.issues = [];
    this.progress = 0;
    /** @type {string} */
    this.projectName;
    this._queryParams = {};
    this._presentationConfigLoaded = false;
  };

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('userDisplayName')) {
      store.dispatch(issueV0.fetchStarredIssues());
    }
    // TODO(zosha): Abort sets of calls to ListIssues when
    // queryParams.q is changed.
    if (this._shouldFetchMatchingIssues(changedProperties)) {
      this._fetchMatchingIssues();
    }
  }

  /**
   * Computes whether to fetch matching issues based on changedProperties
   * @param {Map} changedProperties
   * @return {boolean}
   */
  _shouldFetchMatchingIssues(changedProperties) {
    const wait = shouldWaitForDefaultQuery(this._queryParams);
    if (wait && !this._presentationConfigLoaded) {
      return false;
    } else if (wait && this._presentationConfigLoaded &&
        changedProperties.has('_presentationConfigLoaded')) {
      return true;
    } else if (changedProperties.has('projectName') ||
        changedProperties.has('_currentQuery') ||
        changedProperties.has('_currentCan')) {
      return true;
    }
    return false;
  }

  /** @private */
  _fetchMatchingIssues() {
    store.dispatch(issueV0.fetchIssueList(this.projectName, {
      ...this._queryParams,
      q: this._currentQuery,
      can: this._currentCan,
      maxItems: 500, // 500 items * 12 calls = max of 6,000 issues.
      maxCalls: 12,
    }));
  }

  /** @override */
  stateChanged(state) {
    this.projectName = projectV0.viewedProjectName(state);
    this.issues = (issueV0.issueList(state) || []);
    this.progress = (issueV0.issueListProgress(state) || 0);
    this.totalIssues = (issueV0.totalIssues(state) || 0);
    this._queryParams = sitewide.queryParams(state);
    this._currentQuery = sitewide.currentQuery(state);
    this._currentCan = sitewide.currentCan(state);
    this._presentationConfigLoaded =
      projectV0.viewedPresentationConfigLoaded(state);
  }

  /** @override */
  static get styles() {
    return css `
      progress {
        background-color: white;
        border: 1px solid var(--chops-gray-500);
        width: 40%;
        margin-left: 1%;
        margin-top: 0.5em;
        visibility: visible;
      }
      ::-webkit-progress-bar {
        background-color: white;
      }
      progress::-webkit-progress-value {
        transition: width 1s;
        background-color: var(--chops-blue-700);
      }
      .empty-search {
        text-align: center;
        padding-top: 2em;
      }
    `;
  }
};
customElements.define('mr-grid-page', MrGridPage);
