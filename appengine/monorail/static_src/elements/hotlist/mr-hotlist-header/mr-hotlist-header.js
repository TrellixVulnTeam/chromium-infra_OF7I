// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import 'elements/framework/mr-tabs/mr-tabs.js';

/** @type {readonly MenuItem[]} */
const _MENU_ITEMS = Object.freeze([
  {
    icon: 'list',
    text: 'Issues',
    url: 'issues',
  },
  {
    icon: 'people',
    text: 'People',
    url: 'people',
  },
  {
    icon: 'settings',
    text: 'Settings',
    url: 'settings',
  },
]);

/** Hotlist Issues page */
export class MrHotlistHeader extends LitElement {
  /** @override */
  static get styles() {
    return css`
      h1 {
        font-size: 20px;
        font-weight: normal;
        margin: 16px 24px;
      }
      nav {
        border-bottom: solid #ddd 1px;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <h1>Hotlist ${this.name}</h1>
      <nav>
        <mr-tabs .items=${_MENU_ITEMS} .selected=${this.selected}></mr-tabs>
      </nav>
    `;
  }

  /** @override */
  static get properties() {
    return {
      name: {type: String},
      selected: {type: Number},
    };
  };

  /** @override */
  constructor() {
    super();
    /** @type {string} */
    this.name = '';
    /** @type {number} */
    this.selected = 0;
  }
}

customElements.define('mr-hotlist-header', MrHotlistHeader);
