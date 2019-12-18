// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import 'elements/chops/chops-button/chops-button.js';

/**
 * @typedef {Object} ChoiceOption
 * @property {string=} value a unique string identifier for this option.
 * @property {string=} text the text displayed to the user for this option.
 * @property {string=} url the url this option navigates to.
 */

/**
 * Shared component for rendering a set of choice chips.
 * @extends {LitElement}
 */
export class ChopsChoiceButtons extends LitElement {
  /** @override */
  render() {
    return html`
      ${(this.options).map((option) => this._renderOption(option))}
    `;
  }

  /**
   * Rendering helper for rendering a single option.
   * @param {ChoiceOption} option
   * @return {TemplateResult}
   */
  _renderOption(option) {
    const isSelected = this.value === option.value;
    if (option.url) {
      return html`
        <a
          ?selected=${isSelected}
          aria-current=${isSelected ? 'true' : 'false'}
          href=${option.url}
        >${option.text}</a>
      `;
    }
    return html`
      <button
        ?selected=${isSelected}
        aria-current=${isSelected ? 'true' : 'false'}
        @click=${this._setValue}
        value=${option.value}
      >${option.text}</button>
    `;
  }

  /** @override */
  static get properties() {
    return {
      /**
       * Array of options where each option is an Object with keys:
       * {value, text, url}
       */
      options: {type: Array},
      /**
       * Which button is currently selected.
       */
      value: {type: String},
    };
  };

  /** @override */
  constructor() {
    super();

    /**
     * @type {Array<ChoiceOption>}
     */
    this.options = [];
    this.value = '';
  };

  /** @override */
  static get styles() {
    return css`
      :host {
        display: grid;
        grid-auto-flow: column;
        grid-template-columns: auto;
      }
      button, a {
        display: block;
        cursor: pointer;
        border: 0;
        color: var(--chops-gray-700);
        font-size: var(--chops-normal-font-size);
        margin: 0.1em 4px;
        padding: 4px 10px;
        line-height: 1.4;
        background: var(--chops-choice-bg);
        text-decoration: none;
        border-radius: 16px;
      }
      button[selected], a[selected] {
        background: var(--chops-blue-50);
        color: var(--chops-blue-900);
        border-radius: 16px;
      }
    `;
  };

  /**
   * Public method for allowing parents to change the value of this component.
   * @param {string} newValue
   */
  setValue(newValue) {
    if (newValue !== this.value) {
      this.value = newValue;
      this.dispatchEvent(new CustomEvent('change'));
    }
  }

  /**
   * Private setter for updating the value of the component based on an internal
   * click event.
   * @param {MouseEvent} e
   */
  _setValue(e) {
    this.setValue(e.target.getAttribute('value'));
  }
};

customElements.define('chops-choice-buttons', ChopsChoiceButtons);
