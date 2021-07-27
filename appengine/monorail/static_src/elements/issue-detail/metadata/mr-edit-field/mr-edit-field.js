// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';

import deepEqual from 'deep-equal';
import {fieldTypes, EMPTY_FIELD_VALUE} from 'shared/issue-fields.js';
import {arrayDifference, equalsIgnoreCase} from 'shared/helpers.js';
import {NON_EDITING_KEY_EVENTS} from 'shared/dom-helpers.js';

import './mr-multi-checkbox.js';
import 'react/mr-react-autocomplete.tsx';

const AUTOCOMPLETE_INPUT = 'AUTOCOMPLETE_INPUT';
const CHECKBOX_INPUT = 'CHECKBOX_INPUT';
const SELECT_INPUT = 'SELECT_INPUT';

/**
 * `<mr-edit-field>`
 *
 * A single edit input for a fieldDef + the values of the field.
 *
 */
export class MrEditField extends LitElement {
  /** @override */
  createRenderRoot() {
    return this;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons"
            rel="stylesheet">
      <style>
        mr-edit-field {
          display: block;
        }
        mr-edit-field[hidden] {
          display: none;
        }
        mr-edit-field input,
        mr-edit-field select {
          width: var(--mr-edit-field-width);
          padding: var(--mr-edit-field-padding);
        }
      </style>
      ${this._renderInput()}
    `;
  }

  /**
   * Renders a single input field.
   * @return {TemplateResult}
   */
  _renderInput() {
    switch (this._widgetType) {
      case CHECKBOX_INPUT:
        return html`
          <mr-multi-checkbox
            .options=${this.options}
            .values=${[...this.values]}
            @change=${this._changeHandler}
          ></mr-multi-checkbox>
        `;
      case SELECT_INPUT:
        return html`
          <select
            id="${this.label}"
            class="editSelect"
            aria-label=${this.name}
            @change=${this._changeHandler}
          >
            <option value="">${EMPTY_FIELD_VALUE}</option>
            ${this.options.map((option) => html`
              <option
                value=${option.optionName}
                .selected=${this.value === option.optionName}
              >
                ${option.optionName}
                ${option.docstring ? ' = ' + option.docstring : ''}
              </option>
            `)}
          </select>
        `;
      case AUTOCOMPLETE_INPUT:
        return html`
          <mr-react-autocomplete
            .label=${this.label}
            .vocabularyName=${this.acType || ''}
            .inputType=${this._html5InputType}
            .fixedValues=${this.derivedValues}
            .value=${this.multi ? this.values : this.value}
            .multiple=${this.multi}
            .onChange=${this._changeHandlerReact.bind(this)}
          ></mr-react-autocomplete>
        `;
      default:
        return '';
    }
  }


  /** @override */
  static get properties() {
    return {
      // TODO(zhangtiff): Redesign this a bit so we don't need two separate
      // ways of specifying "type" for a field. Right now, "type" is mapped to
      // the Monorail custom field types whereas "acType" includes additional
      // data types such as components, and labels.
      // String specifying what kind of autocomplete to add to this field.
      acType: {type: String},
      // "type" is based on the various custom field types available in
      // Monorail.
      type: {type: String},
      label: {type: String},
      multi: {type: Boolean},
      name: {type: String},
      // Only used for basic, non-repeated fields.
      placeholder: {type: String},
      initialValues: {
        type: Array,
        hasChanged(newVal, oldVal) {
          // Prevent extra recomputations of the same initial value causing
          // values to be reset.
          return !deepEqual(newVal, oldVal);
        },
      },
      // The current user-inputted values for a field.
      values: {type: Array},
      derivedValues: {type: Array},
      // For enum fields, the possible options that you have. Each entry is a
      // label type with an additional optionName field added.
      options: {type: Array},
    };
  }

  /** @override */
  constructor() {
    super();
    this.initialValues = [];
    this.values = [];
    this.derivedValues = [];
    this.options = [];
    this.multi = false;

    this.actType = '';
    this.placeholder = '';
    this.type = '';
  }

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('initialValues')) {
      // Assume we always want to reset the user's input when initial
      // values change.
      this.reset();
    }
    super.update(changedProperties);
  }

  /**
   * @return {string}
   */
  get value() {
    return _getSingleValue(this.values);
  }

  /**
   * @return {string}
   */
  get _widgetType() {
    const type = this.type;
    const multi = this.multi;
    if (type === fieldTypes.ENUM_TYPE) {
      if (multi) {
        return CHECKBOX_INPUT;
      }
      return SELECT_INPUT;
    } else {
      return AUTOCOMPLETE_INPUT;
    }
  }

  /**
   * @return {string} HTML type for the input.
   */
  get _html5InputType() {
    const type = this.type;
    if (type === fieldTypes.INT_TYPE) {
      return 'number';
    } else if (type === fieldTypes.DATE_TYPE) {
      return 'date';
    }
    return 'text';
  }

  /**
   * Reset form values to initial state.
   */
  reset() {
    this.values = _wrapInArray(this.initialValues);
  }

  /**
   * Return the values that the user added to this input.
   * @return {Array<string>}åß
   */
  getValuesAdded() {
    if (!this.values || !this.values.length) return [];
    return arrayDifference(
        this.values, this.initialValues, equalsIgnoreCase);
  }

  /**
   * Return the values that the userremoved from this input.
   * @return {Array<string>}
   */
  getValuesRemoved() {
    if (!this.multi && (!this.values || this.values.length > 0)) return [];
    return arrayDifference(
        this.initialValues, this.values, equalsIgnoreCase);
  }

  /**
   * Syncs form values and fires a change event as the user edits the form.
   * @param {Event} e
   * @fires Event#change
   * @private
   */
  _changeHandler(e) {
    if (e instanceof KeyboardEvent) {
      if (NON_EDITING_KEY_EVENTS.has(e.key)) return;
    }
    const input = e.target;

    if (input.getValues) {
      // <mr-multi-checkbox> support.
      this.values = input.getValues();
    } else {
      // Is a native input element.
      const value = input.value.trim();
      this.values = _wrapInArray(value);
    }

    this.dispatchEvent(new Event('change'));
  }

  /**
   * Syncs form values and fires a change event as the user edits the form.
   * @param {React.SyntheticEvent} _e
   * @param {string|Array<string>|null} value React autcoomplete form value.
   * @fires Event#change
   * @private
   */
  _changeHandlerReact(_e, value) {
    this.values = _wrapInArray(value);

    this.dispatchEvent(new Event('change'));
  }
}

/**
 * Returns the string value for a single field.
 * @param {Array<string>} arr
 * @return {string}
 */
function _getSingleValue(arr) {
  return (arr && arr.length) ? arr[0] : '';
}

/**
 * Returns the string value for a single field.
 * @param {Array<string>|string} v
 * @return {string}
 */
function _wrapInArray(v) {
  if (!v) return [];

  let values = v;
  if (!Array.isArray(v)) {
    values = !!v ? [v] : [];
  }
  return [...values];
}

customElements.define('mr-edit-field', MrEditField);
