// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {userV3ToRef} from 'shared/convertersV0.js';

import {store, connectStore} from 'reducers/base.js';
import * as hotlists from 'reducers/hotlists.js';
import * as sitewide from 'reducers/sitewide.js';

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
      .placeholder {
        animation: pulse 1s infinite ease-in-out;
        background-clip: content-box;
        height: .8em;
        padding: .3em 0;
        width: 160px;
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
      <mr-hotlist-header selected=1></mr-hotlist-header>

      <section>
        <h2>Owner</h2>
        <p>
          ${this._renderUserLink(this._owner)}
        </p>
      </section>

      <section>
        <h2>Editors</h2>
        ${this._editors ? html`
          ${this._editors.length ? html`
            <ul>
              ${this._editors.map((user) => html`
                <li>${this._renderUserLink(user)}</li>
              `)}
            </ul>
          ` : html`<p>No editors.</p>`}
        ` : html`<div class="placeholder"></div>`}
      </section>
    `;
  }

  /**
   *
   * @param {User} user
   * @return {TemplateResult}
   */
  _renderUserLink(user) {
    if (!user) return html`<div class="placeholder"></div>`;
    return html`<mr-user-link .userRef=${userV3ToRef(user)}></mr-user-link>`;
  }

  /** @override */
  static get properties() {
    return {
      _hotlist: {type: Object},
      _owner: {type: Object},
      _editors: {type: Array},
    };
  }

  /** @override */
  constructor() {
    super();

    /** @type {?Hotlist} */
    this._hotlist = null;
    /** @type {?User} */
    this._owner = null;
    /** @type {?Array<User>} */
    this._editors = null;
  }
};

/** Redux-connected version of _MrHotlistPeoplePage. */
export class MrHotlistPeoplePage extends connectStore(_MrHotlistPeoplePage) {
  /** @override */
  stateChanged(state) {
    this._hotlist = hotlists.viewedHotlist(state);
    this._owner = hotlists.viewedHotlistOwner(state);
    this._editors = hotlists.viewedHotlistEditors(state);
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
}

customElements.define('mr-hotlist-people-page-base', _MrHotlistPeoplePage);
customElements.define('mr-hotlist-people-page', MrHotlistPeoplePage);
