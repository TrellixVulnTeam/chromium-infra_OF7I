// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import page from 'page';
import 'shared/typedef.js';
import {store, connectStore} from 'reducers/base.js';
import {hotlists} from 'reducers/hotlists.js';
import * as sitewide from 'reducers/sitewide.js';
import * as ui from 'reducers/ui.js';
import * as userV0 from 'reducers/userV0.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';

import 'elements/chops/chops-button/chops-button.js';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/**
 * Supported Hotlist privacy options from feature_objects.proto.
 * @enum {string}
 */
const HotlistPrivacy = {
  PRIVATE: 'PRIVATE',
  PUBLIC: 'PUBLIC',
};

/** Hotlist Settings page */
class _MrHotlistSettingsPage extends LitElement {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          display: block;
        }
        section {
          margin: 16px 24px;
        }
        h2 {
          font-weight: normal;
        }
        dl,
        form {
          margin: 16px 24px;
        }
        dt {
          font-weight: bold;
          text-align: right;
          word-wrap: break-word;
        }
        dd {
          margin-left: 0;
        }
        label {
          display: flex;
          flex-direction: column;
        }
        form input,
        form select {
          /* Match minimum size of header. */
          min-width: 250px;
        }
        /* https://material.io/design/layout/responsive-layout-grid.html#breakpoints */
        @media (min-width: 1024px) {
          input,
          select,
          p,
          dd {
            max-width: 750px;
          }
        }
        #save-hotlist {
          background: var(--chops-primary-button-bg);
          color: var(--chops-primary-button-color);
        }
     `,
    ];
  }

  /** @override */
  render() {
    return html`
      <mr-hotlist-header selected=2></mr-hotlist-header>
      ${this._hotlist ? this._renderPage() : 'Loading...'}
    `;
  }

  /**
   * @return {TemplateResult}
   */
  _renderPage() {
    const defaultColumns = this._hotlist.defaultColumns
        .map((col) => col.column).join(' ');
    if (this._permissions.includes(hotlists.ADMINISTER)) {
      return this._renderEditableForm(defaultColumns);
    }
    return this._renderViewOnly(defaultColumns);
  }

  /**
   * Render the editable form Settings page.
   * @param {string} defaultColumns The default columns to be shown.
   * @return {TemplateResult}
   */
  _renderEditableForm(defaultColumns) {
    return html`
      <form id="settingsForm" class="input-grid"
        @change=${this._handleFormChange}>
        <label>Name</label>
        <input id="displayName" class="path"
            value="${this._hotlist.displayName}">
        <label>Summary</label>
        <input id="summary" class="path" value="${this._hotlist.summary}">
        <label>Default Issues columns</label>
        <input id="defaultColumns" class="path" value="${defaultColumns}">
        <label>Who can view this hotlist</label>
        <select id="hotlistPrivacy" class="path">
          <option
            value="${HotlistPrivacy.PUBLIC}"
            ?selected="${this._hotlist.hotlistPrivacy ===
                        HotlistPrivacy.PUBLIC}">
            Anyone on the Internet
          </option>
          <option
            value="${HotlistPrivacy.PRIVATE}"
            ?selected="${this._hotlist.hotlistPrivacy ===
                        HotlistPrivacy.PRIVATE}">
            Members only
          </option>
        </select>
        <span><!-- grid spacer --></span>
        <p>
          Individual issues in the list can only be seen by users who can
          normally see them. The privacy status of an issue is considered
          when it is being displayed (or not displayed) in a hotlist.
        </p>
        <span><!-- grid spacer --></span>
        <div>
          <chops-button @click=${this._save} id="save-hotlist" disabled>
            Save hotlist
          </chops-button>
          <chops-button @click=${this._delete} id="delete-hotlist">
            Delete hotlist
          </chops-button>
        </div>
      </form>
    `;
  }

  /**
   * Render the view-only Settings page.
   * @param {string} defaultColumns The default columns to be shown.
   * @return {TemplateResult}
   */
  _renderViewOnly(defaultColumns) {
    return html`
      <dl class="input-grid">
        <dt>Name</dt>
        <dd>${this._hotlist.displayName}</dd>
        <dt>Summary</dt>
        <dd>${this._hotlist.summary}</dd>
        <dt>Default Issues columns</dt>
        <dd>${defaultColumns}</dd>
        <dt>Who can view this hotlist</dt>
        <dd>
          ${this._hotlist.hotlistPrivacy &&
            this._hotlist.hotlistPrivacy === HotlistPrivacy.PUBLIC ?
            'Anyone on the Internet' : 'Members only'}
        </dd>
        <dt></dt>
        <dd>
          Individual issues in the list can only be seen by users who can
          normally see them. The privacy status of an issue is considered
          when it is being displayed (or not displayed) in a hotlist.
        </dd>
      </dl>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _hotlist: {type: Object},
      _permissions: {type: Array},
      _currentUser: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();
    /** @type {?Hotlist} */
    this._hotlist = null;

    /** @type {Array<Permission>} */
    this._permissions = [];

    /** @type {UserRef} */
    this._currentUser = null;

    // Expose page.js for test stubbing.
    this.page = page;
  }

  /**
   * Handles changes to the editable form.
   * @param {Event} e
   */
  _handleFormChange() {
    const saveButton = this.shadowRoot.getElementById('save-hotlist');
    if (saveButton.disabled) {
      saveButton.disabled = false;
    }
  }

  /** Saves the hotlist, dispatching an action to Redux. */
  async _save() {}

  /** Deletes the hotlist, dispatching an action to Redux. */
  async _delete() {}
};

/** Redux-connected version of _MrHotlistSettingsPage. */
export class MrHotlistSettingsPage
  extends connectStore(_MrHotlistSettingsPage) {
  /** @override */
  stateChanged(state) {
    this._hotlist = hotlists.viewedHotlist(state);
    this._permissions = hotlists.viewedHotlistPermissions(state);
    this._currentUser = userV0.currentUser(state);
  }

  /** @override */
  updated(changedProperties) {
    super.updated(changedProperties);

    if (changedProperties.has('_hotlist') && this._hotlist) {
      const pageTitle = 'Settings - ' + this._hotlist.displayName;
      store.dispatch(sitewide.setPageTitle(pageTitle));
      const headerTitle = 'Hotlist ' + this._hotlist.displayName;
      store.dispatch(sitewide.setHeaderTitle(headerTitle));
    }
  }

  /** @override */
  async _save() {
    const form = this.shadowRoot.getElementById('settingsForm');
    if (!form) return;

    // TODO(https://crbug.com/monorail/7475): Consider generalizing this logic.
    const updatedHotlist = /** @type {Hotlist} */({});
    // These are is an input or select elements.
    const pathInputs = form.querySelectorAll('.path');
    pathInputs.forEach((input) => {
      const path = input.id;
      const value = /** @type {HTMLInputElement} */(input).value;
      switch (path) {
        case 'defaultColumns':
          const columnsValue = [];
          value.trim().split(' ').forEach((column) => {
            if (column) columnsValue.push({column});
          });
          if (JSON.stringify(columnsValue) !==
              JSON.stringify(this._hotlist.defaultColumns)) {
            updatedHotlist.defaultColumns = columnsValue;
          }
          break;
        default:
          if (value !== this._hotlist[path]) updatedHotlist[path] = value;
          break;
      };
    });

    const action = hotlists.update(this._hotlist.name, updatedHotlist);
    await store.dispatch(action);
    this._showHotlistSavedSnackbar();
  }

  /**
   * Shows a snackbar informing the user about their save request.
   */
  async _showHotlistSavedSnackbar() {
    await store.dispatch(ui.showSnackbar(
        'SNACKBAR_ID_HOTLIST_SETTINGS_UPDATED', 'Hotlist Updated.'));
  }

  /** @override */
  async _delete() {
    if (confirm(
        'Are you sure you want to delete this hotlist? This cannot be undone.')
    ) {
      const action = hotlists.deleteHotlist(this._hotlist.name);
      await store.dispatch(action);

      // TODO(crbug/monorail/7430): Handle an error and add <chops-snackbar>.
      // Note that this will redirect regardless of an error.
      this.page(`/u/${this._currentUser.displayName}/hotlists`);
    }
  }
}

customElements.define('mr-hotlist-settings-page-base', _MrHotlistSettingsPage);
customElements.define('mr-hotlist-settings-page', MrHotlistSettingsPage);
