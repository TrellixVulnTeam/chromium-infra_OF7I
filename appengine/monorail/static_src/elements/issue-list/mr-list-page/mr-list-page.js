// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import page from 'page';
import qs from 'qs';
import {store, connectStore} from 'reducers/base.js';
import * as issue from 'reducers/issue.js';
import * as project from 'reducers/project.js';
import * as user from 'reducers/user.js';
import * as sitewide from 'reducers/sitewide.js';
import * as ui from 'reducers/ui.js';
import {prpcClient} from 'prpc-client-instance.js';
import {parseColSpec} from 'shared/issue-fields.js';
import {urlWithNewParams, userIsMember} from 'shared/helpers.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';
import 'elements/framework/mr-dropdown/mr-dropdown.js';
import 'elements/framework/mr-issue-list/mr-issue-list.js';
// eslint-disable-next-line max-len
import 'elements/issue-detail/dialogs/mr-update-issue-hotlists/mr-update-issue-hotlists.js';
import '../dialogs/mr-change-columns/mr-change-columns.js';
import '../mr-mode-selector/mr-mode-selector.js';

export const DEFAULT_ISSUES_PER_PAGE = 100;
const PARAMS_THAT_TRIGGER_REFRESH = ['sort', 'groupby', 'num',
  'start'];
const SNACKBAR_LOADING = 'Loading issues...';

/**
 * `<mr-list-page>`
 *
 * Container page for the list view
 */
export class MrListPage extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          display: block;
          box-sizing: border-box;
          width: 100%;
          padding: 0.5em 8px;
        }
        .container-loading,
        .container-no-issues {
          width: 100%;
          box-sizing: border-box;
          padding: 0 8px;
          font-size: var(--chops-main-font-size);
        }
        .container-no-issues {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
        }
        .container-no-issues p {
          margin: 0.5em;
        }
        .no-issues-block {
          display: block;
          padding: 1em 16px;
          margin-top: 1em;
          flex-grow: 1;
          width: 300px;
          max-width: 100%;
          text-align: center;
          background: var(--chops-primary-accent-bg);
          border-radius: 8px;
          border-bottom: var(--chops-normal-border);
        }
        .no-issues-block[hidden] {
          display: none;
        }
        .list-controls {
          display: flex;
          align-items: center;
          justify-content: space-between;
          width: 100%;
          padding: 0.5em 0;
        }
        .edit-actions {
          flex-grow: 0;
          box-sizing: border-box;
          display: flex;
          align-items: center;
        }
        .edit-actions button {
          height: 100%;
          background: none;
          display: flex;
          align-items: center;
          justify-content: center;
          border: none;
          border-right: var(--chops-normal-border);
          font-size: var(--chops-normal-font-size);
          cursor: pointer;
          transition: 0.2s background ease-in-out;
          color: var(--chops-link-color);
          font-weight: var(--chops-link-font-weight);
          line-height: 160%;
          padding: 0.25em 8px;
        }
        .edit-actions button:hover {
          background: var(--chops-active-choice-bg);
        }
        .right-controls {
          flex-grow: 0;
          display: flex;
          align-items: center;
          justify-content: flex-end;
        }
        .next-link, .prev-link {
          display: inline-block;
          margin: 0 8px;
        }
        mr-mode-selector {
          margin-left: 8px;
        }
        .testing-notice {
          box-sizing: border-box;
          padding: 4px 0.5em;
          text-align: center;
          background: var(--chops-notice-bubble-bg);
          border: var(--chops-normal-border);
          width: 100%;
        }
      `,
    ];
  }

  /** @override */
  render() {
    const selectedRefs = this.selectedIssues.map(
        ({localId, projectName}) => ({localId, projectName}));

    // eslint-disable-next-line
    const feedbackUrl = `https://bugs.chromium.org/p/monorail/issues/entry?labels=UI-Refresh-Feedback&cc=zhangtiff@chromium.org&summary=Feedback+on+the+new+Monorail+UI&components=UI`;
    return html`
      <div class="testing-notice">
        Thanks for trying out the new list view! If you encounter any issues,
        please <a href=${feedbackUrl}>file feedback</a>.
      </div>
      ${this._renderControls()}
      ${this._renderListBody()}
      <mr-update-issue-hotlists
        .issueRefs=${selectedRefs}
        @saveSuccess=${this._showHotlistSaveSnackbar}
      ></mr-update-issue-hotlists>
      <mr-change-columns
        .columns=${this.columns}
        .queryParams=${this._queryParams}
      ></mr-change-columns>
    `;
  }

  /**
   * @return {TemplateResult}
   */
  _renderListBody() {
    if (this.fetchingIssueList && !this.totalIssues) {
      return html`
        <div class="container-loading">
          Loading...
        </div>
      `;
    } else if (!this.totalIssues) {
      return html`
        <div class="container-no-issues">
          <p>
            The search query:
          </p>
          <strong>${this._queryParams.q}</strong>
          <p>
            did not generate any results.
          </p>
          <div class="no-issues-block">
            Type a new query in the search box above
          </div>
          <a
            href=${this._urlWithNewParams({can: 2, q: ''})}
            class="no-issues-block view-all-open"
          >
            View all open issues
          </a>
          <a
            href=${this._urlWithNewParams({can: 1})}
            class="no-issues-block consider-closed"
            ?hidden=${this._queryParams.can === '1'}
          >
            Consider closed issues
          </a>
        </div>
      `;
    }

    return html`
      <mr-issue-list
        .issues=${this.issues}
        .projectName=${this.projectName}
        .queryParams=${this._queryParams}
        .initialCursor=${this._queryParams.cursor}
        .currentQuery=${this.currentQuery}
        .currentCan=${this.currentCan}
        .columns=${this.columns}
        .groups=${this.groups}
        ?selectionEnabled=${this.editingEnabled}
        ?starringEnabled=${this.starringEnabled}
        @selectionChange=${this._setSelectedIssues}
      ></mr-issue-list>
    `;
  }

  /**
   * @return {TemplateResult}
   */
  _renderControls() {
    const maxItems = this.maxItems;
    const startIndex = this.startIndex;
    const end = Math.min(startIndex + maxItems, this.totalIssues);
    const hasNext = end < this.totalIssues;
    const hasPrev = startIndex > 0;

    return html`
      <div class="list-controls">
        <div class="edit-actions">
          ${this.editingEnabled ? html`
            <button
              class="bulk-edit-button"
              @click=${this.bulkEdit}
            >
              Bulk edit
            </button>
            <button
              class="add-to-hotlist-button"
              @click=${this.addToHotlist}
            >
              Add to hotlist
            </button>
            <button
              class="change-columns-button"
              @click=${this.changeColumns}
            >
              Change columns
            </button>
            <mr-dropdown
              icon="more_vert"
              menuAlignment="left"
              label="More actions..."
              .items=${this._moreActions}
            ></mr-dropdown>
          ` : ''}
        </div>

        <div class="right-controls">
          ${hasPrev ? html`
            <a
              href=${this._urlWithNewParams({start: startIndex - maxItems})}
              class="prev-link"
            >
              &lsaquo; Prev
            </a>
          ` : ''}
          <div class="issue-count" ?hidden=${!this.totalIssues}>
            ${startIndex + 1} - ${end} of ${this.totalIssues}
          </div>
          ${hasNext ? html`
            <a
              href=${this._urlWithNewParams({start: startIndex + maxItems})}
              class="next-link"
            >
              Next &rsaquo;
            </a>
          ` : ''}
          <mr-mode-selector
            .projectName=${this.projectName}
            .queryParams=${this._queryParams}
            value="list"
          ></mr-mode-selector>
        </div>
      </div>
    `;
  }

  /** @override */
  static get properties() {
    return {
      issues: {type: Array},
      totalIssues: {type: Number},
      /** @private {Object} */
      _queryParams: {type: Object},
      projectName: {type: String},
      fetchingIssueList: {type: Boolean},
      selectedIssues: {type: Array},
      columns: {type: Array},
      userDisplayName: {type: String},
      /**
       * The current search string the user is querying for.
       */
      currentQuery: {type: String},
      /**
       * The current canned query the user is searching for.
       */
      currentCan: {type: String},
      _isLoggedIn: {type: Boolean},
      _currentUser: {type: Object},
      _usersProjects: {type: Object},
      _fetchIssueListError: {type: String},
    };
  };

  /** @override */
  constructor() {
    super();
    this.issues = [];
    this.fetchingIssueList = false;
    this.selectedIssues = [];
    this._queryParams = {};
    this.columns = [];
    this._usersProjects = new Map();

    this._boundRefresh = this.refresh.bind(this);

    this._moreActions = [
      {
        text: 'Flag as spam',
        handler: () => this._flagIssues(true),
      },
      {
        text: 'Un-flag as spam',
        handler: () => this._flagIssues(false),
      },
    ];

    // Expose page.js for test stubbing.
    this.page = page;
  };

  /** @override */
  connectedCallback() {
    super.connectedCallback();

    window.addEventListener('refreshList', this._boundRefresh);

    // TODO(zhangtiff): Consider if we can make this page title more useful for
    // the list view.
    store.dispatch(sitewide.setPageTitle('Issues'));
  }

  /** @override */
  disconnectedCallback() {
    super.disconnectedCallback();

    window.removeEventListener('refreshList', this._boundRefresh);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('projectName') ||
        changedProperties.has('currentQuery') ||
        changedProperties.has('currentCan')) {
      this.refresh();
    } else if (changedProperties.has('_queryParams')) {
      const oldParams = changedProperties.get('_queryParams') || {};

      const shouldRefresh = PARAMS_THAT_TRIGGER_REFRESH.some((param) => {
        const oldValue = oldParams[param];
        const newValue = this._queryParams[param];
        return oldValue !== newValue;
      });

      if (shouldRefresh) {
        this.refresh();
      }
    }
    if (changedProperties.has('userDisplayName')) {
      store.dispatch(issue.fetchStarredIssues());
    }

    if (changedProperties.has('fetchingIssueList')) {
      const wasFetching = changedProperties.get('fetchingIssueList');
      const isFetching = this.fetchingIssueList;
      // Show a snackbar if waiting for issues to load but only when there's
      // already a different, non-empty issue list loaded. This approach avoids
      // clearing the issue list for a loading screen.
      if (isFetching && this.totalIssues > 0) {
        this._showIssueLoadingSnackbar();
      }
      if (wasFetching && !isFetching) {
        this._hideIssueLoadingSnackbar();
      }
    }

    if (changedProperties.has('_fetchIssueListError') &&
        this._fetchIssueListError) {
      this._showIssueErrorSnackbar(this._fetchIssueListError);
    }
  }

  // TODO(crbug.com/monorail/6933): Remove the need for this wrapper.
  /** Dispatches a Redux action to show an issues loading snackbar.  */
  _showIssueLoadingSnackbar() {
    store.dispatch(ui.showSnackbar(ui.snackbarNames.FETCH_ISSUE_LIST,
        SNACKBAR_LOADING, 0));
  }

  /** Dispatches a Redux action to hide the issue loading snackbar.  */
  _hideIssueLoadingSnackbar() {
    store.dispatch(ui.hideSnackbar(ui.snackbarNames.FETCH_ISSUE_LIST));
  }

  /**
   * Shows a snackbar telling the user their issue loading failed.
   * @param {string} error The error to display.
   */
  _showIssueErrorSnackbar(error) {
    store.dispatch(ui.showSnackbar(ui.snackbarNames.FETCH_ISSUE_LIST_ERROR,
        error));
  }

  /**
   * Refreshes the list of issues show.
   */
  refresh() {
    store.dispatch(issue.fetchIssueList(
        {...this._queryParams, q: this.currentQuery, can: this.currentCan},
        this.projectName,
        {maxItems: this.maxItems, start: this.startIndex}));
  }

  /** @override */
  stateChanged(state) {
    this.projectName = project.viewedProjectName(state);
    this._isLoggedIn = user.isLoggedIn(state);
    this._currentUser = user.user(state);
    this._usersProjects = user.projectsPerUser(state);

    this.issues = (issue.issueList(state) || []);
    this.totalIssues = (issue.totalIssues(state) || 0);
    this.fetchingIssueList = issue.requests(state).fetchIssueList.requesting;

    const error = issue.requests(state).fetchIssueList.error;
    this._fetchIssueListError = error ? error.message : '';

    this.currentQuery = sitewide.currentQuery(state);
    this.currentCan = sitewide.currentCan(state);
    this.columns = sitewide.currentColumns(state);

    this._queryParams = sitewide.queryParams(state);
  }

  /**
   * @return {boolean} Whether the user is able to star the issues in the list.
   */
  get starringEnabled() {
    return this._isLoggedIn;
  }

  /**
   * @return {boolean} Whether the user has permissions to edit the issues in
   *   the list.
   */
  get editingEnabled() {
    return this._isLoggedIn && (userIsMember(this._currentUser,
        this.projectName, this._usersProjects) ||
        this._currentUser.isSiteAdmin);
  }

  /**
   * @return {Array<string>} Array of columns to group by.
   */
  get groups() {
    return parseColSpec(this._queryParams.groupby);
  }

  /**
   * @return {number} Maximum number of issues to load for this query.
   */
  get maxItems() {
    return Number.parseInt(this._queryParams.num) || DEFAULT_ISSUES_PER_PAGE;
  }

  /**
   * @return {number} Number of issues to offset by, based on pagination.
   */
  get startIndex() {
    const num = Number.parseInt(this._queryParams.start) || 0;
    return Math.max(0, num);
  }

  /**
   * Computes the current URL of the page with updated queryParams.
   *
   * @param {Object} newParams keys and values to override existing parameters.
   * @return {string} the new URL.
   */
  _urlWithNewParams(newParams) {
    // TODO(zhangtiff): replace list_new with list when switching over.
    const baseUrl = `/p/${this.projectName}/issues/list_new`;
    return urlWithNewParams(baseUrl, this._queryParams, newParams);
  }

  /**
   * Shows the user an alert telling them their action won't work.
   * @param {string} action Text describing what you're trying to do.
   */
  noneSelectedAlert(action) {
    // TODO(zhangtiff): Replace this with a modal for a more modern feel.
    alert(`Please select some issues to ${action}.`);
  }

  /**
   * Opens the the column selector.
   */
  changeColumns() {
    this.shadowRoot.querySelector('mr-change-columns').open();
  }

  /**
   * Opens a modal to add the selected issues to a hotlist.
   */
  addToHotlist() {
    const issues = this.selectedIssues;
    if (!issues || !issues.length) {
      this.noneSelectedAlert('add to hotlists');
      return;
    }
    this.shadowRoot.querySelector('mr-update-issue-hotlists').open();
  }

  /**
   * Redirects the user to the bulk edit page for the issues they've selected.
   */
  bulkEdit() {
    const issues = this.selectedIssues;
    if (!issues || !issues.length) {
      this.noneSelectedAlert('edit');
      return;
    }
    const params = {
      ids: issues.map((issue) => issue.localId).join(','),
      q: this._queryParams && this._queryParams.q,
    };
    this.page(`/p/${this.projectName}/issues/bulkedit?${qs.stringify(params)}`);
  }

  /** Shows user confirmation that their hotlist changes were saved. */
  _showHotlistSaveSnackbar() {
    store.dispatch(ui.showSnackbar(ui.snackbarNames.UPDATE_HOTLISTS_SUCCESS,
        'Hotlists updated.'));
  }

  /**
   * Flags the selected issues as spam.
   * @param {boolean} flagAsSpam If true, flag as spam. If false, unflag
   *   as spam.
   */
  async _flagIssues(flagAsSpam = true) {
    const issues = this.selectedIssues;
    if (!issues || !issues.length) {
      return this.noneSelectedAlert(
          `${flagAsSpam ? 'flag' : 'un-flag'} as spam`);
    }
    const refs = issues.map((issue) => ({
      localId: issue.localId,
      projectName: issue.projectName,
    }));

    // TODO(zhangtiff): Refactor this into a shared action creator and
    // display the error on the frontend.
    try {
      await prpcClient.call('monorail.Issues', 'FlagIssues', {
        issueRefs: refs,
        flag: flagAsSpam,
      });
      this.refresh();
    } catch (e) {
      console.error(e);
    }
  }

  /**
   * Syncs this component's selected issues with the child component's selected
   * issues.
   */
  _setSelectedIssues() {
    const issueListRef = this.shadowRoot.querySelector('mr-issue-list');
    if (!issueListRef) return;

    this.selectedIssues = issueListRef.selectedIssues;
  }
};
customElements.define('mr-list-page', MrListPage);
