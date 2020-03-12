// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {defaultMemoize} from 'reselect';

import {relativeTime}
  from 'elements/chops/chops-timestamp/chops-timestamp-helpers.js';
import {DEFAULT_ISSUE_FIELD_LIST} from 'shared/issue-fields.js';

import {store, connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as project from 'reducers/project.js';
import * as sitewide from 'reducers/sitewide.js';

import 'elements/chops/chops-filter-chips/chops-filter-chips.js';
import 'elements/framework/mr-issue-list/mr-issue-list.js';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

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
    // Memoize the issues passed to <mr-issue-list> so that
    // out property updates don't cause it to re-render.
    const items = _filterIssues(this._filter, this._items);

    const allProjectNamesEqual = items.length && items.every(
        (issue) => issue.projectName === items[0].projectName);
    const projectName = allProjectNamesEqual ? items[0].projectName : null;

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
        .issues=${items}
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
      _items: {type: Array},
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
    /** @type {Array<HotlistIssue>} */
    this._items = [];
    /** @type {Array<string>} */
    this._columns = [];
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
    this._items = hotlist.viewedHotlistIssues(state);

    const hotlistColumns =
        this._hotlist && this._hotlist.defaultColumns.map((col) => col.column);
    this._columns = sitewide.currentColumns(state) || hotlistColumns;

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
    for (let i = 0; shownItems < index && i < this._items.length; ++i) {
      const item = this._items[i];
      const isShown = _isShown(this._filter, item);
      if (!isShown) ++hiddenItems;
      if (isShown && !items.includes(item.name)) ++shownItems;
    }

    await store.dispatch(hotlist.rerankItems(
        this._hotlist.name, items, index + hiddenItems));
  }
};

const _filterIssues = defaultMemoize(
    /**
     * Filters an array of HotlistIssues based on a filter condition. Memoized.
     * @param {Object<string, boolean>} filter The types of issues to show.
     * @param {Array<HotlistIssue>} items A HotlistIssue to check.
     * @return {Array<HotlistIssue>}
     */
    (filter, items) => items.filter((item) => _isShown(filter, item)));

/**
 * Returns true iff the current filter includes the given HotlistIssue.
 * @param {Object<string, boolean>} filter The types of issues to show.
 * @param {HotlistIssue} item A HotlistIssue to check.
 * @return {boolean}
 */
function _isShown(filter, item) {
  return filter.Open && item.statusRef.meansOpen ||
      filter.Closed && !item.statusRef.meansOpen;
}

customElements.define('mr-hotlist-issues-page-base', _MrHotlistIssuesPage);
customElements.define('mr-hotlist-issues-page', MrHotlistIssuesPage);
