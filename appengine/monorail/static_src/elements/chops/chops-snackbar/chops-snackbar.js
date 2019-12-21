// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';


/**
 * `<chops-snackbar>`
 *
 * A container for showing messages in a snackbar.
 *
 */
export class ChopsSnackbar extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        align-items: center;
        background-color: #333;
        border-radius: 6px;
        bottom: 1em;
        left: 1em;
        color: hsla(0, 0%, 100%, .87);
        display: flex;
        font-size: var(--chops-large-font-size);
        padding: 16px;
        position: fixed;
        z-index: 1000;
      }
      button {
        background: none;
        border: none;
        color: inherit;
        cursor: pointer;
        margin: 0;
        margin-left: 8px;
        padding: 0;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <slot></slot>
      <button @click=${this.close}>
        <i class="material-icons">close</i>
      </button>
    `;
  }

  /**
   * Closes the snackbar.
   */
  close() {
    this.dispatchEvent(new CustomEvent('close'));
  }
}

customElements.define('chops-snackbar', ChopsSnackbar);
