// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import page from 'page';
import {LitElement, html, css} from 'lit-element';

/**
 * `<mr-issue-entry-page>`
 *
 * This is the main details section for a given issue.
 *
 */
export class MrIssueEntryPage extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        margin: 0;
      }
    `;
  }

  /** @override */
  static get properties() {
    return {
      userDisplayName: {type: String},
      loginUrl: {type: String},
    };
  }

  /** @override */
  constructor() {
    super();

    /* dependency injection for testing purpose */
    this._page = page;
  }

  /** @override */
  connectedCallback() {
    super.connectedCallback();
    if (!this.userDisplayName) {
      this._page(this.loginUrl);
    }
  }

  /** @override */
  render() {
    return html`
      <div>SPA issue entry page place holder</div>
    `;
  }
}

customElements.define('mr-issue-entry-page', MrIssueEntryPage);
