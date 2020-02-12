// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {relativeTime}
  from 'elements/chops/chops-timestamp/chops-timestamp-helpers.js';
import {connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as project from 'reducers/project.js';
import {DEFAULT_ISSUE_FIELD_LIST} from 'shared/issue-fields.js';
import 'elements/framework/mr-issue-list/mr-issue-list.js';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/**
 * A HotlistItem with the Issue flattened into the top-level,
 * containing the intersection of the fields of HotlistItem and Issue.
 *
 * @typedef {Issue & HotlistItem} HotlistIssue
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

    const issues = prepareIssues(this._hotlistItems);

    const allProjectNamesEqual = issues.length && issues.every(
        (issue) => issue.projectName === issues[0].projectName);
    const projectName = allProjectNamesEqual ? issues[0].projectName : null;

    return html`
      <mr-hotlist-header .name=${this._hotlist.name} selected=0>
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
        .columns=${this._hotlist.defaultColSpec.split(' ')}
        .defaultFields=${DEFAULT_HOTLIST_FIELDS}
        .extractFieldValues=${this._extractFieldValues.bind(this)}
        ?rerankEnabled=${true}
        ?selectionEnabled=${true}
      ></mr-issue-list>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _hotlist: {type: Object},
      _hotlistItems: {type: Array},
      _extractFieldValuesFromIssue: {type: Object},
    };
  };

  /** @override */
  constructor() {
    super();
    /** @type {Hotlist=} */
    this._hotlist = null;
    /** @type {Array<HotlistItem>} */
    this._hotlistItems = [];
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
    this._extractFieldValuesFromIssue =
      project.extractFieldValuesFromIssue(state);
  }

  /**
   * @param {HotlistIssue} hotlistIssue
   * @param {string} fieldName
   * @return {Array<string>}
   */
  _extractFieldValues(hotlistIssue, fieldName) {
    switch (fieldName) {
      case 'Added':
        return [relativeTime(new Date(hotlistIssue.addedTimestamp * 1000))];
      case 'Adder':
        return [hotlistIssue.adderRef.displayName];
      case 'Note':
        return [hotlistIssue.note];
      case 'Rank':
        return [String(hotlistIssue.rank)];
      default:
        return this._extractFieldValuesFromIssue(hotlistIssue, fieldName);
    }
  }
};

/**
 * @param {Array<HotlistItem>} hotlistItems
 * @return {Array<HotlistIssue>}
 */
export function prepareIssues(hotlistItems) {
  /** @type {Array<HotlistIssue>} */
  const issues = hotlistItems.map((hotlistItem) => {
    return {
      ...hotlistItem.issue,
      addedTimestamp: hotlistItem.addedTimestamp,
      adderRef: hotlistItem.adderRef,
      note: hotlistItem.note,
      rank: hotlistItem.rank,
    };
  });

  issues.sort((a, b) => a.rank - b.rank);
  return issues;
}

customElements.define('mr-hotlist-issues-page', MrHotlistIssuesPage);
