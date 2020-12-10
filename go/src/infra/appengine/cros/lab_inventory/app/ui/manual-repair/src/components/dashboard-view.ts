// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@vaadin/vaadin-grid/theme/lumo/vaadin-grid.js';

import {css, customElement, html, LitElement, property} from 'lit-element';
import {connect} from 'pwa-helpers';

import {SHARED_STYLES} from '../shared/shared-styles';
import {store} from '../state/store';


@customElement('dashboard-view') export class DashboardView extends connect
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
      `,
    ]
  }

  @property({type: Object}) user;
  @property({type: Object}) data;

  render() {
    return html`
      <h1>Dashboard</h1>
      <div id="dashboard">
        <h2>Under Construction</h2>
      </div>
    `;
  }
}
