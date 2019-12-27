// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';
import {NON_EDITING_KEY_EVENTS} from 'shared/dom-helpers.js';

/**
 * @type {RegExp} Autocomplete options are matched at word boundaries. This
 *   Regex specifies what counts as a boundary between words.
 */
const DELIMITER_REGEX = /[^a-z0-9]+/i;

/**
 * A function to specify what happens to the input element an autocomplete
 * instance is attached to when a user selects an autocomplete option. This
 * constant specifies the default behavior where a form's entire value is
 * replaced with the selected value.
 * @param {HTMLInputElement} input An input element.
 * @param {string} value The value of the selected autocomplete option.
 */
const DEFAULT_REPLACER = (input, value) => {
  input.value = value;
};

/**
 * @type {number} The default maximum of completions to render at a time.
 */
const DEFAULT_MAX_COMPLETIONS = 200;

/**
 * @type {number} Globally shared counter for autocomplete instances to help
 *   ensure that no two <chops-autocomplete> options have the same ID.
 */
let idCount = 1;

/**
 * `<chops-autocomplete>` shared autocomplete UI code that inter-ops with
 * other code.
 *
 * chops-autocomplete inter-ops with any input element, whether custom or
 * native that can receive change handlers and has a 'value' property which
 * can be read and set.
 *
 * NOTE: This element disables ShadowDOM for accessibility reasons: to allow
 * aria attributes from the outside to reference features in this element.
 *
 * @customElement chops-autocomplete
 */
export class ChopsAutocomplete extends LitElement {
  /** @override */
  render() {
    const completions = this.completions;
    const currentValue = this._prefix.trim().toLowerCase();
    const index = this._selectedIndex;
    const currentCompletion = index >= 0 &&
      index < completions.length ? completions[index] : '';

    return html`
      <style>
        /*
         * Really specific class names are necessary because ShadowDOM
         * is disabled for this component.
         */
        .chops-autocomplete-container {
          position: relative;
        }
        .chops-autocomplete-container table {
          padding: 0;
          font-size: var(--chops-main-font-size);
          color: var(--chops-link-color);
          position: absolute;
          background: white;
          border: var(--chops-accessible-border);
          z-index: 999;
          box-shadow: 2px 3px 8px 0px hsla(0, 0%, 0%, 0.3);
          border-spacing: 0;
          border-collapse: collapse;
          /* In the case when the autocomplete extends the
           * height of the viewport, we want to make sure
           * there's spacing. */
          margin-bottom: 1em;
        }
        .chops-autocomplete-container tbody {
          display: block;
          min-width: 100px;
          max-height: 500px;
          overflow: auto;
        }
        .chops-autocomplete-container tr {
          cursor: pointer;
          transition: background 0.2s ease-in-out;
        }
        .chops-autocomplete-container tr[data-selected] {
          background: var(--chops-active-choice-bg);
          text-decoration: underline;
        }
        .chops-autocomplete-container td {
          padding: 0.25em 8px;
          white-space: nowrap;
        }
        .screenreader-hidden {
          clip: rect(1px, 1px, 1px, 1px);
          height: 1px;
          overflow: hidden;
          position: absolute;
          white-space: nowrap;
          width: 1px;
        }
      </style>
      <div class="chops-autocomplete-container">
        <span class="screenreader-hidden" aria-live="polite">
          ${currentCompletion}
        </span>
        <table
          ?hidden=${!completions.length}
        >
          <tbody>
            ${completions.map((completion, i) => html`
              <tr
                id=${completionId(this.id, i)}
                ?data-selected=${i === index}
                data-index=${i}
                data-value=${completion}
                @mouseover=${this._hoverCompletion}
                @mousedown=${this._clickCompletion}
                role="option"
                aria-selected=${completion.toLowerCase() ===
                  currentValue ? 'true' : 'false'}
              >
                <td class="completion">
                  ${this._renderCompletion(completion)}
                </td>
                <td class="docstring">
                  ${this._renderDocstring(completion)}
                </td>
              </tr>
            `)}
          </tbody>
        </table>
      </div>
    `;
  }

  /**
   * Renders a single autocomplete result.
   * @param {string} completion The string for the currently selected
   *   autocomplete value.
   * @return {TemplateResult}
   */
  _renderCompletion(completion) {
    const matchDict = this._matchDict;

    if (!(completion in matchDict)) return completion;

    const {index, matchesDoc} = matchDict[completion];

    if (matchesDoc) return completion;

    const prefix = this._prefix;
    const start = completion.substr(0, index);
    const middle = completion.substr(index, prefix.length);
    const end = completion.substr(index + prefix.length);

    return html`${start}<b>${middle}</b>${end}`;
  }

  /**
   * Finds the docstring for a given autocomplete result and renders it.
   * @param {string} completion The autocomplete result rendered.
   * @return {TemplateResult}
   */
  _renderDocstring(completion) {
    const matchDict = this._matchDict;
    const docDict = this.docDict;

    if (!completion in docDict) return '';

    const doc = docDict[completion];

    if (!(completion in matchDict)) return doc;

    const {index, matchesDoc} = matchDict[completion];

    if (!matchesDoc) return doc;

    const prefix = this._prefix;
    const start = doc.substr(0, index);
    const middle = doc.substr(index, prefix.length);
    const end = doc.substr(index + prefix.length);

    return html`${start}<b>${middle}</b>${end}`;
  }

  /** @override */
  static get properties() {
    return {
      /**
       * The input this element is for.
       */
      for: {type: String},
      /**
       * Generated id for the element.
       */
      id: {
        type: String,
        reflect: true,
      },
      /**
       * The role attribute, set for accessibility.
       */
      role: {
        type: String,
        reflect: true,
      },
      /**
       * Array of strings for possible autocompletion values.
       */
      strings: {type: Array},
      /**
       * A dictionary containing optional doc strings for each autocomplete
       * string.
       */
      docDict: {type: Object},
      /**
       * An optional function to compute what happens when the user selects
       * a value.
       */
      replacer: {type: Object},
      /**
       * An Array of the currently suggested autcomplte values.
       */
      completions: {type: Array},
      /**
       * Maximum number of completion values that can display at once.
       */
      max: {type: Number},
      /**
       * Dict of locations of matched substrings. Value format:
       * {index, matchesDoc}.
       */
      _matchDict: {type: Object},
      _selectedIndex: {type: Number},
      _prefix: {type: String},
      _forRef: {type: Object},
      _boundToggleCompletionsOnFocus: {type: Object},
      _boundNavigateCompletions: {type: Object},
      _boundUpdateCompletions: {type: Object},
      _oldAttributes: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();

    this.strings = [];
    this.docDict = {};
    this.completions = [];
    this.max = DEFAULT_MAX_COMPLETIONS;

    this.role = 'listbox';
    this.id = `chops-autocomplete-${idCount++}`;

    this._matchDict = {};
    this._selectedIndex = -1;
    this._prefix = '';
    this._boundToggleCompletionsOnFocus =
      this._toggleCompletionsOnFocus.bind(this);
    this._boundUpdateCompletions = this._updateCompletions.bind(this);
    this._boundNavigateCompletions = this._navigateCompletions.bind(this);
    this._oldAttributes = {};
  }

  // Disable shadow DOM to allow aria attributes to propagate.
  /** @override */
  createRenderRoot() {
    return this;
  }

  /** @override */
  disconnectedCallback() {
    super.disconnectedCallback();

    this._disconnectAutocomplete(this._forRef);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('for')) {
      const forRef = this.getRootNode().querySelector('#' + this.for);

      // TODO(zhangtiff): Make this element work with custom input components
      // in the future as well.
      this._forRef = (forRef.tagName || '').toUpperCase() === 'INPUT' ?
        forRef : undefined;
      this._connectAutocomplete(this._forRef);
    }
    if (this._forRef) {
      if (changedProperties.has('id')) {
        this._forRef.setAttribute('aria-owns', this.id);
      }
      if (changedProperties.has('completions')) {
        // a11y. Tell screenreaders whether the autocomplete is expanded.
        this._forRef.setAttribute('aria-expanded',
          this.completions.length ? 'true' : 'false');
      }

      if (changedProperties.has('_selectedIndex') ||
          changedProperties.has('completions')) {
        this._updateAriaActiveDescendant(this._forRef);

        this._scrollCompletionIntoView(this._selectedIndex);
      }
    }
  }

  /**
   * Sets the aria-activedescendant attribute of the element (ie: an input form)
   * that the autocomplete is attached to, in order to tell screenreaders about
   * which autocomplete option is currently selected.
   * @param {HTMLInputElement} element
   */
  _updateAriaActiveDescendant(element) {
    const i = this._selectedIndex;

    if (i >= 0 && i < this.completions.length) {
      const selectedId = completionId(this.id, i);

      // a11y. Set the ID of the currently selected element.
      element.setAttribute('aria-activedescendant', selectedId);

      // Scroll the container to make sure the selected element is in view.
    } else {
      element.setAttribute('aria-activedescendant', '');
    }
  }

  /**
   * When a user moves up or down from an autocomplete option that's at the top
   * or bottom of the autocomplete option container, we must scroll the
   * container to make sure the user always sees the option they've selected.
   * @param {number} i The index of the autocomplete option to put into view.
   */
  _scrollCompletionIntoView(i) {
    const selectedId = completionId(this.id, i);

    const container = this.querySelector('tbody');
    const completion = this.querySelector(`#${selectedId}`);

    if (!completion) return;

    const distanceFromTop = completion.offsetTop - container.scrollTop;

    // If the completion is above the viewport for the container.
    if (distanceFromTop < 0) {
      // Position the completion at the top of the container.
      container.scrollTop = completion.offsetTop;
    }

    // If the compltion is below the viewport for the container.
    if (distanceFromTop > (container.offsetHeight - completion.offsetHeight)) {
      // Position the compltion at the bottom of the container.
      container.scrollTop = completion.offsetTop - (container.offsetHeight -
        completion.offsetHeight);
    }
  }

  /**
   * Changes the input's value according to the rules of the replacer function.
   * @param {string} value - the value to swap in.
   * @return {undefined}
   */
  completeValue(value) {
    if (!this._forRef) return;

    const replacer = this.replacer || DEFAULT_REPLACER;
    replacer(this._forRef, value);

    this.hideCompletions();
  }

  /**
   * Computes autocomplete values matching the current input in the field.
   * @return {Boolean} Whether any completions were found.
   */
  showCompletions() {
    if (!this._forRef) {
      this.hideCompletions();
      return false;
    }
    this._prefix = this._forRef.value.trim().toLowerCase();
    // Always select the first completion by default when recomputing
    // completions.
    this._selectedIndex = 0;

    const matchDict = {};
    const accepted = [];
    matchDict;
    for (let i = 0; i < this.strings.length &&
        accepted.length < this.max; i++) {
      const s = this.strings[i];
      let matchIndex = this._matchIndex(this._prefix, s);
      let matches = matchIndex >= 0;
      if (matches) {
        matchDict[s] = {index: matchIndex, matchesDoc: false};
      } else if (s in this.docDict) {
        matchIndex = this._matchIndex(this._prefix, this.docDict[s]);
        matches = matchIndex >= 0;
        if (matches) {
          matchDict[s] = {index: matchIndex, matchesDoc: true};
        }
      }
      if (matches) {
        accepted.push(s);
      }
    }

    this._matchDict = matchDict;

    this.completions = accepted;

    return !!this.completions.length;
  }

  /**
   * Finds where a given user input matches an autocomplete option. Note that
   * a match is only found if the substring is at either the beginning of the
   * string or the beginning of a delimited section of the string. Hence, we
   * refer to the "needle" in this function a "prefix".
   * @param {string} prefix The value that the user inputed into the form.
   * @param {string} s The autocomplete option that's being compared.
   * @return {number} An integer for what index the substring is found in the
   *   autocomplete option. Returns -1 if no match.
   */
  _matchIndex(prefix, s) {
    const matchStart = s.toLowerCase().indexOf(prefix.toLocaleLowerCase());
    if (matchStart === 0 ||
        (matchStart > 0 && s[matchStart - 1].match(DELIMITER_REGEX))) {
      return matchStart;
    }
    return -1;
  }

  /**
   * Hides autocomplete options.
   */
  hideCompletions() {
    this.completions = [];
    this._prefix = '';
    this._selectedIndex = -1;
  }

  /**
   * Sets an autocomplete option that a user hovers over as the selected option.
   * @param {MouseEvent} e
   */
  _hoverCompletion(e) {
    const target = e.currentTarget;

    if (!target.dataset || !target.dataset.index) return;

    const index = Number.parseInt(target.dataset.index);
    if (index >= 0 && index < this.completions.length) {
      this._selectedIndex = index;
    }
  }

  /**
   * Sets the value of the form input that the user is editing to the
   * autocomplete option that the user just clicked.
   * @param {MouseEvent} e
   */
  _clickCompletion(e) {
    e.preventDefault();
    const target = e.currentTarget;
    if (!target.dataset || !target.dataset.value) return;

    this.completeValue(target.dataset.value);
  }

  /**
   * Hides and shows the autocomplete completions when a user focuses and
   * unfocuses a form.
   * @param {FocusEvent} e
   */
  _toggleCompletionsOnFocus(e) {
    const target = e.target;

    // Check if the input is focused or not.
    if (target.matches(':focus')) {
      this.showCompletions();
    } else {
      this.hideCompletions();
    }
  }

  /**
   * Implements hotkeys to allow the user to navigate autocomplete options with
   * their keyboard. ie: pressing up and down to select options or Esc to close
   * the form.
   * @param {KeyboardEvent} e
   */
  _navigateCompletions(e) {
    const completions = this.completions;
    if (!completions.length) return;

    switch (e.key) {
      // TODO(zhangtiff): Throttle or control keyboard navigation so the user
      // can't navigate faster than they can can perceive.
      case 'ArrowUp':
        e.preventDefault();
        this._navigateUp();
        break;
      case 'ArrowDown':
        e.preventDefault();
        this._navigateDown();
        break;
      case 'Enter':
      // TODO(zhangtiff): Add Tab to this case as well once all issue detail
      // inputs use chops-autocomplete.
        e.preventDefault();
        if (this._selectedIndex >= 0 &&
            this._selectedIndex <= completions.length) {
          this.completeValue(completions[this._selectedIndex]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        this.hideCompletions();
        break;
    }
  }

  /**
   * Selects the completion option above the current one.
   */
  _navigateUp() {
    const completions = this.completions;
    this._selectedIndex -= 1;
    if (this._selectedIndex < 0) {
      this._selectedIndex = completions.length - 1;
    }
  }

  /**
   * Selects the completion option below the current one.
   */
  _navigateDown() {
    const completions = this.completions;
    this._selectedIndex += 1;
    if (this._selectedIndex >= completions.length) {
      this._selectedIndex = 0;
    }
  }

  /**
   * Recomputes autocomplete completions when the user types a new input.
   * Ignores KeyboardEvents that don't change the input value of the form
   * to prevent excess recomputations.
   * @param {KeyboardEvent} e
   */
  _updateCompletions(e) {
    if (NON_EDITING_KEY_EVENTS.has(e.key)) return;
    this.showCompletions();
  }

  /**
   * Initializes the input element that this autocomplete instance is
   * attached to with aria attributes required for accessibility.
   * @param {HTMLInputElement} node The input element that the autocomplete is
   *   attached to.
   */
  _connectAutocomplete(node) {
    if (!node) return;

    node.addEventListener('keyup', this._boundUpdateCompletions);
    node.addEventListener('keydown', this._boundNavigateCompletions);
    node.addEventListener('focus', this._boundToggleCompletionsOnFocus);
    node.addEventListener('blur', this._boundToggleCompletionsOnFocus);

    this._oldAttributes = {
      'aria-owns': node.getAttribute('aria-owns'),
      'aria-autocomplete': node.getAttribute('aria-autocomplete'),
      'aria-expanded': node.getAttribute('aria-expanded'),
      'aria-haspopup': node.getAttribute('aria-haspopup'),
      'aria-activedescendant': node.getAttribute('aria-activedescendant'),
    };
    node.setAttribute('aria-owns', this.id);
    node.setAttribute('aria-autocomplete', 'both');
    node.setAttribute('aria-expanded', 'false');
    node.setAttribute('aria-haspopup', 'listbox');
    node.setAttribute('aria-activedescendant', '');
  }

  /**
   * When <chops-autocomplete> is disconnected or moved to a difference form,
   * this function removes the side effects added by <chops-autocomplete> on the
   * input element that <chops-autocomplete> is attached to.
   * @param {HTMLInputElement} node The input element that the autocomplete is
   *   attached to.
   */
  _disconnectAutocomplete(node) {
    if (!node) return;

    node.removeEventListener('keyup', this._boundUpdateCompletions);
    node.removeEventListener('keydown', this._boundNavigateCompletions);
    node.removeEventListener('focus', this._boundToggleCompletionsOnFocus);
    node.removeEventListener('blur', this._boundToggleCompletionsOnFocus);

    for (const key of Object.keys(this._oldAttributes)) {
      node.setAttribute(key, this._oldAttributes[key]);
    }
    this._oldAttributes = {};
  }
}

/**
 * Generates a unique HTML ID for a given autocomplete option, for use by
 * aria-activedescendant. Note that because the autocomplete element has
 * ShadowDOM disabled, we need to make sure the ID is specific enough to be
 * globally unique across the entire application.
 * @param {string} prefix A unique prefix to differentiate this autocomplete
 *   instance from other autocomplete instances.
 * @param {number} i The index of the autocomplete option.
 * @return {string} A unique HTML ID for a given autocomplete option.
 */
function completionId(prefix, i) {
  return `${prefix}-option-${i}`;
}

customElements.define('chops-autocomplete', ChopsAutocomplete);
