// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import 'elements/framework/mr-issue-list/mr-issue-list.js';

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
    const issues = this.hotlistItems.map((hotlistItem) => hotlistItem.issue);

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
      ></mr-issue-list>
    `;
  }

  /** @override */
  static get properties() {
    return {
      hotlist: {type: Object},
      hotlistItems: {type: Array},
    };
  };

  /** @override */
  constructor() {
    super();
    this.hotlist = null;
    this.hotlistItems = [];
  }

  /** @override */
  stateChanged(state) {
    this.hotlist = hotlist.viewedHotlist(state);
    this.hotlistItems = hotlist.viewedHotlistItems(state);
  }
};

customElements.define('mr-hotlist-issues-page', MrHotlistIssuesPage);
