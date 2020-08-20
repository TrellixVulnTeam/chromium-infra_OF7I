// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-top-app-bar-fixed';
import '@material/mwc-icon-button';
import '@chopsui/chops-signin';
import '@chopsui/chops-signin-aware/chops-signin-aware';

import {customElement, html, LitElement} from 'lit-element';
import {connect} from 'pwa-helpers';

import {receiveUser} from '../state/actions';
import {store} from '../state/store';


@customElement('top-bar') export class TopBar extends connect
(store)(LitElement) {
  render() {
    return html`
      <mwc-top-app-bar-fixed>
        <div slot="title">Manual Repair Records</div>
        <chops-signin
          slot="actionItems"
          client-id="974141142451-7804s4o2kugouvi6vndg3pm91jqfmmh1.apps.googleusercontent.com">
        </chops-signin>
        <chops-signin-aware
          id="csa"
          @user-update="${this.handleUserUpdate}">
        </chops-signin-aware>
      </mwc-top-app-bar-fixed>
    `;
  }

  handleUserUpdate(e) {
    const user = {
      signedIn: e.srcElement.signedIn,
      profile: e.srcElement.profile,
      authHeaders: e.srcElement.authHeaders
    };
    store.dispatch(receiveUser(user));
  }
}
