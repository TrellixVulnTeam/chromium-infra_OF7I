// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';
import 'elements/hotlist/mr-hotlist-header/mr-hotlist-header.js';

/** Hotlist People page */
export class MrHotlistPeoplePage extends LitElement {
  /** @override */
  render() {
    return html`
      <mr-hotlist-header name="Name" selected=1>
      </mr-hotlist-header>
    `;
  };


  /** @override */
  static get properties() {
    return {};
  };
};

customElements.define('mr-hotlist-people-page', MrHotlistPeoplePage);
