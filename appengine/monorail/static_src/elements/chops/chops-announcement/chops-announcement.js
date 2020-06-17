// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

/**
 * `<chops-announcement>` displays a ChopsDash message when there's an outage
 * or other important announcement.
 *
 * @customElement chops-announcement
 */
export class ChopsAnnouncement extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: none;
      }
    `;
  }
  /** @override */
  render() {
    return html``;
  }

  /** @override */
  static get properties() {
    return {
    };
  }
}
customElements.define('chops-announcement', ChopsAnnouncement);
