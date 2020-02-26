// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

/**
 * `<chops-chip>` displays a chip.
 * "Chips are compact elements that represent an input, attribute, or action."
 * https://material.io/components/chips/
 */
export class ChopsChip extends LitElement {
  /** @override */
  static get properties() {
    return {
      focusable: {type: Boolean, reflect: true},
      thumbnail: {type: String},
      buttonIcon: {type: String},
      buttonLabel: {type: String},
    };
  }

  /** @override */
  constructor() {
    super();

    /** @type {boolean} */
    this.focusable = false;

    /** @type {string} */
    this.thumbnail = '';

    /** @type {string} */
    this.buttonIcon = '';
    /** @type {string} */
    this.buttonLabel = '';
  }

  /** @override */
  static get styles() {
    return css`
      :host {
        --chops-chip-bg-color: var(--chops-blue-gray-50);
        display: inline-flex;
        padding: 0px 10px;
        line-height: 22px;
        margin: 0 2px;
        border-radius: 12px;
        background: var(--chops-chip-bg-color);
        align-items: center;
        font-size: var(--chops-main-font-size);
        box-sizing: border-box;
        border: 1px solid var(--chops-chip-bg-color);
      }
      :host(:focus), :host(.selected) {
        background: var(--chops-active-choice-bg);
        border: 1px solid var(--chops-light-accent-color);
      }
      :host([hidden]) {
        display: none;
      }
      i.left {
        margin: 0 4px 0 -6px;
      }
      button {
        border-radius: 50%;
        cursor: pointer;
        background: none;
        border: 0;
        padding: 0;
        margin: 0 -6px 0 4px;
        display: inline-flex;
        align-items: center;
        transition: background-color 0.2s ease-in-out;
      }
      button[hidden] {
        display: none;
      }
      button:hover {
        background: var(--chops-gray-300);
      }
      i.material-icons {
        color: var(--chops-primary-icon-color);
        font-size: 14px;
        user-select: none;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      ${this.thumbnail ? html`
        <i class="material-icons left">${this.thumbnail}</i>
      ` : ''}
      <slot></slot>
      ${this.buttonIcon ? html`
        <button @click=${this.clickButton} aria-label=${this.buttonLabel}>
          <i class="material-icons" aria-hidden="true"}>${this.buttonIcon}</i>
        </button>
      `: ''}
    `;
  }

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('focusable')) {
      this.tabIndex = this.focusable ? '0' : undefined;
    }
    super.update(changedProperties);
  }

  /**
   * @param {MouseEvent} e A click event.
   */
  clickButton(e) {
    this.dispatchEvent(new CustomEvent('click-button'));
  }
}
customElements.define('chops-chip', ChopsChip);
