// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {relativeTime}
  from 'elements/chops/chops-timestamp/chops-timestamp-helpers.js';
import {DEFAULT_ISSUE_FIELD_LIST} from 'shared/issue-fields.js';

import {store, connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as issue from 'reducers/issue.js';
import * as project from 'reducers/project.js';
import * as sitewide from 'reducers/sitewide.js';

import 'elements/chops/chops-filter-chips/chops-filter-chips.js';
import 'elements/framework/mr-issue-list/mr-issue-list.js';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/**
 * A HotlistItem with the Issue flattened into the top-level,
 * containing the intersection of the fields of HotlistItem and Issue.
 *
 * @typedef {Issue & HotlistItemV3} HotlistIssue
 */

const DEFAULT_HOTLIST_FIELDS = Object.freeze([
  ...DEFAULT_ISSUE_FIELD_LIST,
  'Added',
  'Adder',
  'Rank',
]);

/** Hotlist Issues page */
export class _MrHotlistIssuesPage extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
      }
      p, div {
        margin: 16px 24px;
      }
      div {
        display: flex;
        align-items: center;
      }
      chops-filter-chips {
        margin-left: 6px;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <mr-hotlist-header selected=0></mr-hotlist-header>
      ${this._hotlist ? this._renderPage() : 'Loading...'}
    `;
  }

  /**
   * @return {TemplateResult}
   */
  _renderPage() {
    const issues = this._prepareIssues(this._hotlistItems);

    const allProjectNamesEqual = issues.length && issues.every(
        (issue) => issue.projectName === issues[0].projectName);
    const projectName = allProjectNamesEqual ? issues[0].projectName : null;

    return html`
      <p>${this._hotlist.summary}</p>

      <div>
        Filter by Status
        <chops-filter-chips
            .options=${['Open', 'Closed']}
            .selected=${this._filter}
            @change=${this._onFilterChange}
        ></chops-filter-chips>
      </div>

      <mr-issue-list
        .issues=${issues}
        .projectName=${projectName}
        .columns=${this._columns}
        .defaultFields=${DEFAULT_HOTLIST_FIELDS}
        .extractFieldValues=${this._extractFieldValues.bind(this)}
        .rerank=${this._rerank.bind(this)}
        ?selectionEnabled=${true}
      ></mr-issue-list>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _hotlist: {type: Object},
      _hotlistItems: {type: Array},
      _columns: {type: Array},
      _issue: {type: Object},
      _extractFieldValuesFromIssue: {type: Object},
      _filter: {type: Object},
    };
  };

  /** @override */
  constructor() {
    super();
    /** @type {?HotlistV3} */
    this._hotlist = null;
    /** @type {Array<HotlistItemV3>} */
    this._hotlistItems = [];
    /** @type {Array<string>} */
    this._columns = [];
    /**
     * @param {string} _name
     * @return {?Issue}
     */
    this._issue = (_name) => null;
    /**
     * @param {Issue} _issue
     * @param {string} _fieldName
     * @return {Array<string>}
     */
    this._extractFieldValuesFromIssue = (_issue, _fieldName) => [];
    /** @type {Object<string, boolean>} */
    this._filter = {Open: true};
  }

  /**
   * @param {Array<HotlistItemV3>} items
   * @return {Array<HotlistIssue>}
   */
  _prepareIssues(items) {
    // Filter out issues that haven't been fetched yet or failed to fetch.
    // Example: if the user doesn't have permissions to view the issue.
    // <mr-issue-list> assumes that certain fields are included in each Issue.
    const itemsWithData = items.filter((item) => this._issue(item.issue));

    const filteredItems = itemsWithData.filter(this._isShown.bind(this));

    return filteredItems.map((item) => ({
      ...this._issue(item.issue),
      name: item.name,
      rank: item.rank || 0,
      adder: item.adder, // TODO(dtu): Fetch the User's displayName.
      createTime: item.createTime,
    }));
  }

  /**
   * @param {HotlistIssue} hotlistIssue
   * @param {string} fieldName
   * @return {Array<string>}
   */
  _extractFieldValues(hotlistIssue, fieldName) {
    switch (fieldName) {
      case 'Added':
        return [relativeTime(new Date(hotlistIssue.createTime))];
      case 'Adder':
        return [hotlistIssue.adder.displayName];
      case 'Rank':
        return [String(hotlistIssue.rank + 1)];
      default:
        return this._extractFieldValuesFromIssue(hotlistIssue, fieldName);
    }
  }

  /**
   * @param {Event} e A change event fired by <chops-filter-chips>.
   */
  _onFilterChange(e) {
    this._filter = e.target.selected;
  }

  /**
   * Returns true iff the current filter includes the given HotlistItem.
   * @param {HotlistItemV3} item A HotlistItem in the current Hotlist.
   * @return {boolean}
   */
  _isShown(item) {
    return this._filter.Open && this._issue(item.issue).statusRef.meansOpen ||
        this._filter.Closed && !this._issue(item.issue).statusRef.meansOpen;
  }

  /**
   * Reranks items in the hotlist, dispatching the action to the Redux store.
   * @param {Array<String>} items The names of the HotlistItems to move.
   * @param {number} index The index to insert the moved items.
   * @return {Promise<void>}
   */
  async _rerank(items, index) {}
};

/** Redux-connected version of _MrHotlistIssuesPage. */
export class MrHotlistIssuesPage extends connectStore(_MrHotlistIssuesPage) {
  /** @override */
  stateChanged(state) {
    this._hotlist = hotlist.viewedHotlist(state);
    this._hotlistItems = hotlist.viewedHotlistItems(state);

    const hotlistColumns =
        this._hotlist && this._hotlist.defaultColumns.map((col) => col.column);
    this._columns = sitewide.currentColumns(state) || hotlistColumns;

    this._issue = issue.issue(state);
    this._extractFieldValuesFromIssue =
      project.extractFieldValuesFromIssue(state);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('_hotlist') && this._hotlist) {
      const pageTitle = `Issues - ${this._hotlist.displayName}`;
      store.dispatch(sitewide.setPageTitle(pageTitle));
      const headerTitle = `Hotlist ${this._hotlist.displayName}`;
      store.dispatch(sitewide.setHeaderTitle(headerTitle));
    }
  }

  /** @override */
  async _rerank(items, index) {
    // The index given from <mr-issue-list> includes only the items shown in
    // the list and excludes the items that are being moved. So, we need to
    // count the hidden items.
    let shownItems = 0;
    let hiddenItems = 0;
    for (let i = 0; shownItems < index && i < this._hotlistItems.length; ++i) {
      const item = this._hotlistItems[i];
      if (!this._isShown(item)) ++hiddenItems;
      if (this._isShown(item) && !items.includes(item.name)) ++shownItems;
    }

    await store.dispatch(hotlist.rerankItems(
        this._hotlist.name, items, index + hiddenItems));
  }
};

customElements.define('mr-hotlist-issues-page-base', _MrHotlistIssuesPage);
customElements.define('mr-hotlist-issues-page', MrHotlistIssuesPage);
