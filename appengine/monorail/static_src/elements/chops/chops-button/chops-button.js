// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

/**
 * `<chops-button>` displays a styled button component with a few niceties.
 *
 * @customElement chops-button
 * @demo /demo/chops-button_demo.html
 */
export class ChopsButton extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        --chops-button-padding: 0.5em 16px;
        background: hsla(0, 0%, 95%, 1);
        margin: 0.25em 4px;
        cursor: pointer;
        border-radius: 3px;
        text-align: center;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        user-select: none;
        transition: filter 0.3s ease-in-out, box-shadow 0.3s ease-in-out;
        font-family: var(--chops-font-family);
      }
      :host([hidden]) {
        display: none;
      }
      :host([raised]) {
        box-shadow: 0px 2px 8px -1px hsla(0, 0%, 0%, 0.5);
      }
      :host(:hover) {
        filter: brightness(95%);
      }
      :host(:active) {
        filter: brightness(115%);
      }
      :host([raised]:active) {
        box-shadow: 0px 1px 8px -1px hsla(0, 0%, 0%, 0.5);
      }
      :host([disabled]),
      :host([disabled]:hover) {
        filter: grayscale(30%);
        opacity: 0.4;
        background: hsla(0, 0%, 87%, 1);
        cursor: default;
        pointer-events: none;
        box-shadow: none;
      }
      button {
        background: none;
        width: 100%;
        height: 100%;
        border: 0;
        padding: var(--chops-button-padding);
        margin: 0;
        color: inherit;
        cursor: inherit;
        text-align: center;
        font-family: inherit;
        text-align: inherit;
        font-weight: inherit;
        font-size: inherit;
        line-height: inherit;
        border-radius: inherit;
        display: flex;
        align-items: center;
        justify-content: center;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <button ?disabled=${this.disabled}>
        <slot></slot>
      </button>
    `;
  }

  /** @override */
  static get properties() {
    return {
      /** Whether the button is available for input or not. */
      disabled: {
        type: Boolean,
        reflect: true,
      },
      /** Whether the button should have a shadow or not. */
      raised: {
        type: Boolean,
        value: false,
        reflect: true,
      },
    };
  }

  /** @override */
  constructor() {
    super();

    this.disabled = false;
    this.raised = false;
  }
}
customElements.define('chops-button', ChopsButton);
