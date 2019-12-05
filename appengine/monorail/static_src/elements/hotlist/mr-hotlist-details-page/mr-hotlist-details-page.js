// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {connectStore} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';

/** Hotlist Details page */
export class MrHotlistDetailsPage extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
        padding: 0.5em 8px;
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
    if (!this.hotlist) {
      return html`Loading...`;
    }

    return html`
      <section>
        <h1>Hotlist Settings</h1>
        <dl>
          <dt>Name</dt>
          <dd>${this.hotlist.name}</dd>
          <dt>Summary</dt>
          <dd>${this.hotlist.summary}</dd>
          <dt>Description</dt>
          <dd>${this.hotlist.description}</dd>
        </dl>
      </section>

      <section>
        <h1>Hotlist Defaults</h1>
        <dl>
          <dt>Default columns shown in list view</dt>
          <dd></dd>
        </dl>
      </section>

      <section>
        <h1>Hotlist Access</h1>
        <dl>
          <dt>Who can view this hotlist</dt>
          <dd></dd>
        </dl>
        <p>Individual issues in the list can only be seen by users who can
          normally see them. The privacy status of an issue is considered
          when it is being displayed (or not displayed) in a hotlist.
      </section>
    `;
  }

  /** @override */
  static get properties() {
    return {
      hotlist: {type: Object},
    };
  }

  /** @override */
  constructor() {
    super();
    this.hotlist = null;
  }

  /** @override */
  stateChanged(state) {
    this.hotlist = hotlist.viewedHotlist(state);
  }
};

customElements.define('mr-hotlist-details-page', MrHotlistDetailsPage);
