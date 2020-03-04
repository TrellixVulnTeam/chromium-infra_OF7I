// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {store, connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as sitewide from 'reducers/sitewide.js';

import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/** Hotlist Settings page */
class _MrHotlistSettingsPage extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
      }
      section {
        margin: 16px 24px;
      }
      dt {
        font-weight: bold;
      }
      dd {
        margin: 0;
      }
    `;
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
    return html`
      <section>
        <h1>Hotlist Settings</h1>
        <dl>
          <dt>Name</dt>
          <dd>${this._hotlist.displayName}</dd>
          <dt>Summary</dt>
          <dd>${this._hotlist.summary}</dd>
          <dt>Description</dt>
          <dd>${this._hotlist.description}</dd>
        </dl>
      </section>

      <section>
        <h1>Hotlist Defaults</h1>
        <dl>
          <dt>Default columns shown in list view</dt>
          <dd>${defaultColumns}</dd>
        </dl>
      </section>

      <section>
        <h1>Hotlist Access</h1>
        <dl>
          <dt>Who can view this hotlist</dt>
          <dd>
            ${this._hotlist.hotlistPrivacy ?
              'Anyone on the internet' : 'Members only'}
          </dd>
        </dl>
        <p>
          Individual issues in the list can only be seen by users who can
          normally see them. The privacy status of an issue is considered
          when it is being displayed (or not displayed) in a hotlist.
      </section>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _hotlist: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();
    this._hotlist = null;
  }
};

/** Redux-connected version of _MrHotlistSettingsPage. */
export class MrHotlistSettingsPage extends connectStore(
    _MrHotlistSettingsPage) {
  /** @override */
  stateChanged(state) {
    this._hotlist = hotlist.viewedHotlist(state);
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
}

customElements.define('mr-hotlist-settings-page-base', _MrHotlistSettingsPage);
customElements.define('mr-hotlist-settings-page', MrHotlistSettingsPage);
