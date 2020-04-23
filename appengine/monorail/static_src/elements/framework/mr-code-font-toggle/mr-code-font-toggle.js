// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';

import {store, connectStore} from 'reducers/base.js';
import * as userV0 from 'reducers/userV0.js';
import 'elements/chops/chops-toggle/chops-toggle.js';

/**
 * `<mr-code-font-toggle>`
 *
 * Code font toggle button for the issue detail page.  Pressing it
 * causes issue description and comment text to switch to monospace
 * font and the setting is saved in the user's preferences.
 */
export class MrCodeFontToggle extends connectStore(LitElement) {
  /** @override */
  render() {
    return html`
      <chops-toggle
        ?checked=${this._codeFont}
        ?disabled=${this._prefsInFlight}
        @checked-change=${this._toggleFont}
        title="Code font"
       >Code</chops-toggle>
    `;
  }

  /** @override */
  static get properties() {
    return {
      prefs: {type: Object},
      userDisplayName: {type: String},
      initialValue: {type: Boolean},
      _prefsInFlight: {type: Boolean},
    };
  }

  /** @override */
  stateChanged(state) {
    this.prefs = userV0.prefs(state);
    this._prefsInFlight = userV0.requests(state).fetchPrefs.requesting ||
      userV0.requests(state).setPrefs.requesting;
  }

  /** @override */
  constructor() {
    super();
    this.initialValue = false;
    this.userDisplayName = '';
  }

  // Used by the legacy EZT page to interact with Redux.
  fetchPrefs() {
    store.dispatch(userV0.fetchPrefs());
  }

  get _codeFont() {
    const {prefs, initialValue} = this;
    if (!prefs) return initialValue;
    return prefs.get('code_font') === 'true';
  }

  /**
   * Toggles the code font in response to the user activating the button.
   * @param {Event} e
   * @fires CustomEvent#font-toggle
   * @private
   */
  _toggleFont(e) {
    const checked = e.detail.checked;
    this.dispatchEvent(new CustomEvent('font-toggle', {detail: {checked}}));

    const newPrefs = [{name: 'code_font', value: '' + checked}];
    store.dispatch(userV0.setPrefs(newPrefs, !!this.userDisplayName));
  }
}
customElements.define('mr-code-font-toggle', MrCodeFontToggle);
