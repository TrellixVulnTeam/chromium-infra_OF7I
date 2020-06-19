// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import debounce from 'debounce';
import {LitElement, html, css} from 'lit-element';

import {userV3ToRef} from 'shared/convertersV0.js';

import {store, connectStore} from 'reducers/base.js';
import {hotlists} from 'reducers/hotlists.js';
import * as sitewide from 'reducers/sitewide.js';
import * as users from 'reducers/users.js';

import 'elements/framework/links/mr-user-link/mr-user-link.js';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/** Hotlist People page */
class _MrHotlistPeoplePage extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
      }
      section {
        margin: 16px 24px;
      }
      h2 {
        font-weight: normal;
      }

      ul {
        padding: 0;
      }
      li {
        list-style-type: none;
      }
      p, li, form {
        display: flex;
      }
      p, ul, li, form {
        margin: 12px 0;
      }

      input {
        margin-left: -6px;
        padding: 4px;
        width: 320px;
      }

      button {
        align-items: center;
        background-color: transparent;
        border: 0;
        cursor: pointer;
        display: inline-flex;
        margin: 0 4px;
        padding: 0;
      }
      .material-icons {
        font-size: 18px;
      }

      .placeholder::before {
        animation: pulse 1s infinite ease-in-out;
        border-radius: 3px;
        content: " ";
        height: 10px;
        margin: 4px 0;
        width: 200px;
      }
      @keyframes pulse {
        0% {background-color: var(--chops-blue-50);}
        50% {background-color: var(--chops-blue-75);}
        100% {background-color: var(--chops-blue-50);}
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <mr-hotlist-header selected=1></mr-hotlist-header>

      <section>
        <h2>Owner</h2>
        ${this._renderOwner(this._owner)}
      </section>

      <section>
        <h2>Editors</h2>
        ${this._renderEditors(this._editors)}

        ${this._permissions.includes(hotlists.ADMINISTER) ? html`
          <form @submit=${this._onAddEditors}>
            <input id="add" placeholder="List of email addresses"></input>
            <button><i class="material-icons">add</i></button>
          </form>
        ` : html``}
      </section>
    `;
  }

  /**
   * @param {?User} owner
   * @return {TemplateResult}
   */
  _renderOwner(owner) {
    if (!owner) return html`<p class="placeholder"></p>`;
    return html`
      <p><mr-user-link .userRef=${userV3ToRef(owner)}></mr-user-link></p>
    `;
  }

  /**
   * @param {?Array<User>} editors
   * @return {TemplateResult}
   */
  _renderEditors(editors) {
    if (!editors) return html`<p class="placeholder"></p>`;
    if (!editors.length) return html`<p>No editors.</p>`;

    return html`
      <ul>${editors.map((editor) => this._renderEditor(editor))}</ul>
    `;
  }

  /**
   * @param {?User} editor
   * @return {TemplateResult}
   */
  _renderEditor(editor) {
    if (!editor) return html`<li class="placeholder"></li>`;

    const canRemove = this._permissions.includes(hotlists.ADMINISTER) ||
        editor.name === this._currentUserName;

    return html`
      <li>
        <mr-user-link .userRef=${userV3ToRef(editor)}></mr-user-link>
        ${canRemove ? html`
          <button @click=${this._removeEditor.bind(this, editor.name)}>
            <i class="material-icons">clear</i>
          </button>
        ` : html``}
      </li>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _hotlist: {type: Object},
      _owner: {type: Object},
      _editors: {type: Array},
      _permissions: {type: Array},
      _currentUserName: {type: String},
    };
  }

  /** @override */
  constructor() {
    super();

    /** @type {?Hotlist} */ this._hotlist = null;
    /** @type {?User} */ this._owner = null;
    /** @type {Array<User>} */ this._editors = null;
    /** @type {Array<Permission>} */ this._permissions = [];
    /** @type {?String} */ this._currentUserName = null;

    this._debouncedAddEditors = debounce(this._addEditors, 400, true);
  }

  /** Adds hotlist editors.
   * @param {Event} event
   */
  async _onAddEditors(event) {
    event.preventDefault();

    const input =
      /** @type {HTMLInputElement} */ (this.shadowRoot.getElementById('add'));
    const emails = input.value.split(/[\s,;]/).filter((e) => e);
    if (!emails.length) return;
    const editors = emails.map((email) => 'users/' + email);
    try {
      await this._debouncedAddEditors(editors);
      input.value = '';
    } catch (error) {
      // The `hotlists.update()` call shows a snackbar on errors.
    }
  }

  /** Adds hotlist editors.
   * @param {Array<string>} editors An Array of User resource names.
   */
  async _addEditors(editors) {}

  /**
   * Removes a hotlist editor.
   * @param {string} name A User resource name.
  */
  async _removeEditor(name) {}
};

/** Redux-connected version of _MrHotlistPeoplePage. */
export class MrHotlistPeoplePage extends connectStore(_MrHotlistPeoplePage) {
  /** @override */
  stateChanged(state) {
    this._hotlist = hotlists.viewedHotlist(state);
    this._owner = hotlists.viewedHotlistOwner(state);
    this._editors = hotlists.viewedHotlistEditors(state);
    this._permissions = hotlists.viewedHotlistPermissions(state);
    this._currentUserName = users.currentUserName(state);
  }

  /** @override */
  updated(changedProperties) {
    super.updated(changedProperties);

    if (changedProperties.has('_hotlist') && this._hotlist) {
      const pageTitle = 'People - ' + this._hotlist.displayName;
      store.dispatch(sitewide.setPageTitle(pageTitle));
      const headerTitle = 'Hotlist ' + this._hotlist.displayName;
      store.dispatch(sitewide.setHeaderTitle(headerTitle));
    }
  }

  /** @override */
  async _addEditors(editors) {
    await store.dispatch(hotlists.update(this._hotlist.name, {editors}));
  }

  /** @override */
  async _removeEditor(name) {
    await store.dispatch(hotlists.removeEditors(this._hotlist.name, [name]));
  }
}

customElements.define('mr-hotlist-people-page-base', _MrHotlistPeoplePage);
customElements.define('mr-hotlist-people-page', MrHotlistPeoplePage);
