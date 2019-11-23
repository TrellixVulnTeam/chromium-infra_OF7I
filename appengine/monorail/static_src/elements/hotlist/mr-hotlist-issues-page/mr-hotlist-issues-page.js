// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';

/** Hotlist Issues page */
export class MrHotlistIssuesPage extends LitElement {
  /** @override */
  render() {
    return html`Hotlist Issues`;
  }

  /** @override */
  static get properties() {
    return {};
  };
};

customElements.define('mr-hotlist-issues-page', MrHotlistIssuesPage);
