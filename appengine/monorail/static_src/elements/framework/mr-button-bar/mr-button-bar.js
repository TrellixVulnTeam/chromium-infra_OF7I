// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import 'elements/framework/mr-dropdown/mr-dropdown.js';

import 'shared/typedef.js';

/** Button bar containing table controls. */
export class MrButtonBar extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: flex;
      }
      button {
        background: none;
        color: var(--chops-link-color);
        cursor: pointer;
        font-size: var(--chops-normal-font-size);
        font-weight: var(--chops-link-font-weight);

        line-height: 24px;
        padding: 4px 16px;

        border: none;

        align-items: center;
        display: inline-flex;
      }
      button:hover {
        background: var(--chops-active-choice-bg);
      }
      i.material-icons {
        font-size: 20px;
        margin-right: 4px;
        vertical-align: middle;
      }
      mr-dropdown {
        --mr-dropdown-anchor-padding: 6px 4px;
        --mr-dropdown-icon-color: var(--chops-link-color);
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      ${this.items.map(_renderItem)}
    `;
  }

  /** @override */
  static get properties() {
    return {
      items: {type: Array},
    };
  };

  /** @override */
  constructor() {
    super();

    /** @type {Array<MenuItem>} */
    this.items = [];
  }
};

/**
 * Renders one item.
 * @param {MenuItem} item
 * @return {TemplateResult}
 */
function _renderItem(item) {
  if (item.items) {
    return html`
      <mr-dropdown
        icon=${item.icon}
        menuAlignment="left"
        label=${item.text}
        .items=${item.items}
      ></mr-dropdown>
    `;
  } else {
    return html`
      <button @click=${item.handler}>
        <i class="material-icons" ?hidden=${!item.icon}>
          ${item.icon}
        </i>
        ${item.text}
      </button>
    `;
  }
}

customElements.define('mr-button-bar', MrButtonBar);
