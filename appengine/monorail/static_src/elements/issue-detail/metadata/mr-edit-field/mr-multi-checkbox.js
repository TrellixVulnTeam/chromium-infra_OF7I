// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

/**
 * `<mr-multi-checkbox>`
 *
 * A web component for managing values in a set of checkboxes.
 *
 */
export class MrMultiCheckbox extends LitElement {
  /** @override */
  static get styles() {
    return css`
      input[type="checkbox"] {
        width: auto;
        height: auto;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      ${this.options.map((option) => html`
        <label title=${option.docstring}>
          <input
            type="checkbox"
            name=${this.name}
            value=${option.optionName}
            ?checked=${this.values.includes(option.optionName)}
            @change=${this._changeHandler}
          />
          ${option.optionName}
        </label>
      `)}
    `;
  }

  /** @override */
  static get properties() {
    return {
      values: {type: Array},
      options: {type: Array},
      _inputRefs: {type: Object},
    };
  }


  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('options')) {
      this._inputRefs = this.shadowRoot.querySelectorAll('input');
    }

    if (changedProperties.has('values')) {
      this.reset();
    }
  }

  reset() {
    this.setValues(this.values);
  }

  getValues() {
    if (!this._inputRefs) return;
    const valueList = [];
    this._inputRefs.forEach((c) => {
      if (c.checked) {
        valueList.push(c.value.trim());
      }
    });
    return valueList;
  }

  setValues(values) {
    if (!this._inputRefs) return;
    this._inputRefs.forEach(
        (checkbox) => {
          checkbox.checked = values.includes(checkbox.value);
        },
    );
  }

  _changeHandler() {
    this.dispatchEvent(new CustomEvent('change'));
  }
}

customElements.define('mr-multi-checkbox', MrMultiCheckbox);
