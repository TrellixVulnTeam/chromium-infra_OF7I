// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import 'elements/chops/chops-chip/chops-chip.js';

/**
 * `<chops-filter-chips>` displays a set of filter chips.
 * https://material.io/components/chips/#filter-chips
 */
export class ChopsFilterChips extends LitElement {
  /** @override */
  static get properties() {
    return {
      options: {type: Array},
      selected: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();
    /** @type {Array<string>} */
    this.options = [];
    /** @type {Object<string, boolean>} */
    this.selected = {};
  }

  /** @override */
  static get styles() {
    return css`
      :host {
        display: inline-flex;
      }
    `;
  }

  /** @override */
  render() {
    return html`${this.options.map((option) => this._renderChip(option))}`;
  }

  /**
   * Render a single chip.
   * @param {string} option The text on the chip.
   * @return {TemplateResult}
   */
  _renderChip(option) {
    return html`
      <chops-chip
          @click=${this.select.bind(this, option)}
          class=${this.selected[option] ? 'selected' : ''}
          .thumbnail=${this.selected[option] ? 'check' : ''}>
        ${option}
      </chops-chip>
    `;
  }

  /**
   * Selects or unselects an option.
   * @param {string} option The option to select or unselect.
   * @fires Event#change
   */
  select(option) {
    this.selected = {...this.selected, [option]: !this.selected[option]};
    this.dispatchEvent(new Event('change'));
  }
}
customElements.define('chops-filter-chips', ChopsFilterChips);
