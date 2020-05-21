// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';


/**
 * `<mr-issue-slo>`
 *
 * A widget for showing the given issue's SLO status.
 */
export class MrIssueSlo extends LitElement {
  /** @override */
  static get styles() {
    return css``;
  }

  /** @override */
  render() {
    // TODO(crbug.com/monorail/7740): SLO rendering implementation.
    return html`N/A`;
  }

  /** @override */
  static get properties() {
    return {};
  }

  /** @override */
  constructor() {
    super();
  }
}
customElements.define('mr-issue-slo', MrIssueSlo);
