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
        padding: 0.5em 8px;
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
    const issues = prepareIssues(this.hotlistItems);

    const allProjectNamesEqual = issues.length && issues.every(
        (issue) => issue.projectName === issues[0].projectName);
    const projectName = allProjectNamesEqual ? issues[0].projectName : null;

    return html`
      <h1>Hotlist ${this.hotlist.name}</h1>
      <dl>
        <dt>Summary</dt>
        <dd>${this.hotlist.summary}</dd>
        <dt>Description</dt>
        <dd>${this.hotlist.description}</dd>
      </dl>
      <mr-issue-list
        .issues=${issues}
        .projectName=${projectName}
        .columns=${this.hotlist.defaultColSpec.split(' ')}
        .defaultFields=${DEFAULT_HOTLIST_FIELDS}
        .extractFieldValues=${this._extractFieldValues.bind(this)}
      ></mr-issue-list>
    `;
  }

  /** @override */
  static get properties() {
    return {
      hotlist: {type: Object},
      hotlistItems: {type: Array},
      _extractFieldValuesFromIssue: {type: Object},
    };
  };

  /** @override */
  constructor() {
    super();
    /** @type {Hotlist=} */
    this.hotlist = null;
    /** @type {Array<HotlistItem>} */
    this.hotlistItems = [];
    /**
     * @param {Issue} _issue
     * @param {string} _fieldName
     * @return {Array<string>}
     */
    this._extractFieldValuesFromIssue = (_issue, _fieldName) => [];
  }

  /** @override */
  stateChanged(state) {
    this.hotlist = hotlist.viewedHotlist(state);
    this.hotlistItems = hotlist.viewedHotlistItems(state);
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
