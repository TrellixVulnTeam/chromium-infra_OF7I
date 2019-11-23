// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';

/** Hotlist Details page */
export class MrHotlistDetailsPage extends LitElement {
  /** @override */
  render() {
    return html`Hotlist Details`;
  }

  /** @override */
  static get properties() {
    return {};
  };
};

customElements.define('mr-hotlist-details-page', MrHotlistDetailsPage);
