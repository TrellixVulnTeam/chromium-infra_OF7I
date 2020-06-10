// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import 'elements/chops/chops-timestamp/chops-timestamp.js';
import {determineSloStatus} from './slo-rules.js';

/** @typedef {import('./slo-rules.js').SloStatus} SloStatus */

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
    const sloStatus = this._determineSloStatus();
    if (!sloStatus) {
      return html`N/A`;
    }
    if (!sloStatus.target) {
      return html`Done`;
    }
    return html`
      <chops-timestamp .timestamp=${sloStatus.target} short></chops-timestamp>`;
  }

  /**
   * Wrapper around slo-rules.js determineSloStatus to allow tests to override
   * the return value.
   * @private
   * @return {SloStatus}
   */
  _determineSloStatus() {
    return this.issue ? determineSloStatus(this.issue) : null;
  }

  /** @override */
  static get properties() {
    return {
      issue: {type: Object},
    };
  }
  /** @override */
  constructor() {
    super();
    /** @type {Issue} */
    this.issue;
  }
}
customElements.define('mr-issue-slo', MrIssueSlo);
