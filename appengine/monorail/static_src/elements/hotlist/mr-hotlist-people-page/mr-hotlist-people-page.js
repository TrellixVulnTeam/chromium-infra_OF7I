// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {store, connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as sitewide from 'reducers/sitewide.js';

import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/** Hotlist People page */
class _MrHotlistPeoplePage extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
      }
      p {
        margin: 16px 24px;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <mr-hotlist-header selected=1></mr-hotlist-header>
      ${this._hotlist ? this._renderPage() : 'Loading...'}
    `;
  }

  /**
   * @return {TemplateResult}
   */
  _renderPage() {
    return html`
      <p>
        This page is not yet implemented.
        This will be a list of hotlist members.
      </p>
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

/** Redux-connected version of _MrHotlistPeoplePage. */
export class MrHotlistPeoplePage extends connectStore(_MrHotlistPeoplePage) {
  /** @override */
  stateChanged(state) {
    this._hotlist = hotlist.viewedHotlist(state);
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
