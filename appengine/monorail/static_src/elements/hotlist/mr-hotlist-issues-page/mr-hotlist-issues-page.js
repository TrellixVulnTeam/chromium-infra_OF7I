// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {relativeTime}
  from 'elements/chops/chops-timestamp/chops-timestamp-helpers.js';
import {connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as issue from 'reducers/issue.js';
import * as project from 'reducers/project.js';
import {DEFAULT_ISSUE_FIELD_LIST} from 'shared/issue-fields.js';
import 'elements/framework/mr-issue-list/mr-issue-list.js';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/**
 * A HotlistItem with the Issue flattened into the top-level,
 * containing the intersection of the fields of HotlistItem and Issue.
 *
 * @typedef {Issue & HotlistItemV1} HotlistIssue
 */

const DEFAULT_HOTLIST_FIELDS = Object.freeze([
  ...DEFAULT_ISSUE_FIELD_LIST,
  'Added',
  'Adder',
  'Note',
  'Rank',
]);

/** Hotlist Issues page */
export class MrHotlistIssuesPage extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
      }
      dl {
        margin: 16px 24px;
      }
      dt {
        font-weight: bold;
      }
      dd {
        margin: 0;
      }
    `;
  }

  /** @override */
  render() {
    if (!this._hotlist) {
      return html`Loading...`;
    }

    const issues = this.prepareIssues(this._hotlistItems);

    const allProjectNamesEqual = issues.length && issues.every(
        (issue) => issue.projectName === issues[0].projectName);
    const projectName = allProjectNamesEqual ? issues[0].projectName : null;

    return html`
      <mr-hotlist-header .name=${this._hotlist.displayName} selected=0>
      </mr-hotlist-header>

      <dl>
        <dt>Summary</dt>
        <dd>${this._hotlist.summary}</dd>
        <dt>Description</dt>
        <dd>${this._hotlist.description}</dd>
      </dl>

      <mr-issue-list
        .issues=${issues}
        .projectName=${projectName}
        .columns=${this._hotlist.defaultColumns.map((col) => col.column)}
        .defaultFields=${DEFAULT_HOTLIST_FIELDS}
        .extractFieldValues=${this._extractFieldValues.bind(this)}
        .rerank=${hotlist.rerankItems.bind(null, this._hotlist.name)}
        ?selectionEnabled=${true}
      ></mr-issue-list>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _hotlist: {type: Object},
      _hotlistItems: {type: Array},
      _issue: {type: Object},
      _extractFieldValuesFromIssue: {type: Object},
    };
  };

  /** @override */
  constructor() {
    super();
    /** @type {?HotlistV1} */
    this._hotlist = null;
    /** @type {Array<HotlistItemV1>} */
    this._hotlistItems = [];
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
  }

  /** @override */
  stateChanged(state) {
    this._hotlist = hotlist.viewedHotlist(state);
    this._hotlistItems = hotlist.viewedHotlistItems(state);
    this._issue = issue.issue(state);
    this._extractFieldValuesFromIssue =
      project.extractFieldValuesFromIssue(state);
  }

  /**
   * @param {Array<HotlistItemV1>} items
   * @return {Array<HotlistIssue>}
   */
  prepareIssues(items) {
    // Filter out issues that haven't been fetched yet or failed to fetch.
    // Example: if the user doesn't have permissions to view the issue.
    // <mr-issue-list> assumes that certain fields are included in each Issue.
    const itemsWithData = items.filter((item) => this._issue(item.issue));

    return itemsWithData.map((item) => ({
      ...this._issue(item.issue),
      name: item.name,
      rank: item.rank || 0,
      adder: item.adder, // TODO(dtu): Fetch the User's displayName.
      createTime: item.createTime,
      note: item.note,
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
        return [hotlistIssue.adder];
      case 'Note':
        return [hotlistIssue.note];
      case 'Rank':
        return [String(hotlistIssue.rank + 1)];
      default:
        return this._extractFieldValuesFromIssue(hotlistIssue, fieldName);
    }
  }
};

customElements.define('mr-hotlist-issues-page', MrHotlistIssuesPage);
