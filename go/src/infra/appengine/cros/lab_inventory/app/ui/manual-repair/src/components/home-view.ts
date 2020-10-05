// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.


import {css, customElement, html, LitElement} from 'lit-element';
import {connect} from 'pwa-helpers';

import {SHARED_STYLES} from '../shared/shared-styles';
import {store} from '../state/store';


@customElement('home-view') export class HomeView extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        h1 {
          margin-bottom: 0.5em;
        }

        h2 {
          margin-bottom: 0.3em;
        }

        #disclosure-container {
          text-align: center;
          display: flex;
          justify-content: center;
        }

        #disclosure-container a {
          margin: 0 0.3em;
          font-size: 0.8em;
        }
      `,
    ]
  }

  render() {
    return html`
      <h1>Manual Repair UI</h1>
      <p>The Manual Repair UI provides an interface to create and submit manual repair records for DUTs. Log in using your Google account to use the tool.</p>
      <br>
      <h2>Instructions</h2>
      <p>To create or update a manual repair record for a DUT or a Labstation,</p>
      <ol>
        <li>Enter the hostname into the search bar above and hit 'Enter'.</li>
        <li>Fill out the form with necessary details.</li>
        <li>Hit the appropriate button in the lower right hand corner.</li>
      </ol>
      <br>
      <h2>Links</h2>
      <ul>
        <li>G3 Docs - <a target="_blank" href="http://go/manual-repair-docs">
          go/manual-repair-docs
        </a></li>
        <li>Chatroom - <a target="_blank" href="http://go/manual-repair-chat">
          go/manual-repair-chat
        </a></li>
      </ul>
      <br>
      <div id="disclosure-container">
        <a target="_blank" href="https://www.google.com/policies/privacy/">
          Privacy
        </a>
        <a target="_blank" href="https://www.google.com/policies/terms/">
          Terms
        </a>
      </div>
    `;
  }
}
