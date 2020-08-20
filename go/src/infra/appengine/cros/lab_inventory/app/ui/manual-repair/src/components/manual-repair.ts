// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './search-hostname';
import './top-bar';
import {css, customElement, html, LitElement} from 'lit-element';


@customElement('manual-repair')
export class ManualRepair extends LitElement {
  static get styles() {
    return [css`
      #app-body {
        width: 70%;
        margin: auto;
        padding: 20px 0;
        display: flex;
        justify-content: center;
      }
    `];
  }

  render() {
    return html`
      <div slot="appContent">
        <top-bar></top-bar>
        <div id="app-body">
          <search-hostname></search-hostname>
        </div>
      </div>
    `;
  }
}
