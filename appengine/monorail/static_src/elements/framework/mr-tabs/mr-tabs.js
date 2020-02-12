// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import 'shared/typedef.js';

/**
 * `<mr-tabs>`
 *
 * A Material Design tabs strip. https://material.io/components/tabs/
 *
 */
export class MrTabs extends LitElement {
  /** @override */
  static get styles() {
    return css`
      ul {
        display: flex;
        list-style: none;
        margin: 0;
        padding: 0;
      }
      li {
        color: var(--chops-choice-color);
      }
      li.selected {
        color: var(--chops-active-choice-color);
      }
      li:hover {
        background: var(--chops-primary-accent-bg);
        color: var(--chops-active-choice-color);
      }
      a {
        color: inherit;
        text-decoration: none;

        display: inline-block;
        line-height: 38px;
        padding: 0 24px;
      }
      li.selected a {
        border-bottom: solid 2px;
      }
      i.material-icons {
        vertical-align: middle;
        margin-right: 4px;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <ul>
        ${this.items.map(this._renderTab.bind(this))}
      </ul>
    `;
  }

  /**
   * Renders one tab.
   * @param {MenuItem} item
   * @param {number} index
   * @return {TemplateResult}
   */
  _renderTab(item, index) {
    return html`
      <li class=${index === this.selected ? 'selected' : ''}>
        <a href=${item.url}>
          <i class="material-icons" ?hidden=${!item.icon}>
            ${item.icon}
          </i>
          ${item.text}
        </a>
      </li>
    `;
  }

  /** @override */
  static get properties() {
    return {
      items: {type: Array},
      selected: {type: Number},
    };
  }

  /** @override */
  constructor() {
    super();

    /** @type {Array<MenuItem>} */
    this.items = [];
    this.selected = 0;
  }
}

customElements.define('mr-tabs', MrTabs);
