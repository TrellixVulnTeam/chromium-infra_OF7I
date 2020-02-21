// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/** Hotlist Settings page */
export class MrHotlistSettingsPage extends connectStore(LitElement) {
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
    if (!this._hotlist) {
      return html`Loading...`;
    }

    const defaultColumns = this._hotlist.defaultColumns
        .map((col) => col.column).join(' ');
    return html`
      <mr-hotlist-header .name=${this._hotlist.displayName} selected=2>
      </mr-hotlist-header>

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

  /** @override */
  stateChanged(state) {
    this._hotlist = hotlist.viewedHotlist(state);
  }
};

customElements.define('mr-hotlist-settings-page', MrHotlistSettingsPage);
