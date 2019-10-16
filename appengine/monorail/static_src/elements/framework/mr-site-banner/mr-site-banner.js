// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import 'elements/chops/chops-timestamp/chops-timestamp.js';
import {connectStore} from 'reducers/base.js';
import * as sitewide from 'reducers/sitewide.js';

export class MrSiteBanner extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host([hidden]) {
        display: none;
      }
      :host {
        display: block;
        font-weight: bold;
        color: var(--chops-field-error-color);
        background: var(--chops-orange-50);
        padding: 5px;
        text-align: center;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      ${this.bannerMessage}
      ${this.bannerTime ? html`
        <chops-timestamp
          .timestamp=${this.bannerTime}
        ></chops-timestamp>
      ` : ''}
    `;
  }

  /** @override */
  static get properties() {
    return {
      hidden: {
        type: Boolean,
        reflect: true,
      },
      bannerMessage: {type: String},
      bannerTime: {type: Number},
    };
  }

  /** @override */
  constructor() {
    super();
    this.bannerMessage = '';
    this.bannerTime = 0;
    this.hidden = false;
  }

  /** @override */
  stateChanged(state) {
    this.bannerMessage = sitewide.bannerMessage(state);
    this.bannerTime = sitewide.bannerTime(state);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('bannerMessage')) {
      this.hidden = !this.bannerMessage;
    }
  }
}

customElements.define('mr-site-banner', MrSiteBanner);
