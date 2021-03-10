// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-top-app-bar-fixed';
import '@material/mwc-icon-button';
import '@chopsui/chops-signin';
import '@chopsui/chops-signin-aware/chops-signin-aware';

import {Drawer} from '@material/mwc-drawer';
import {customElement, html, LitElement} from 'lit-element';
import {connect} from 'pwa-helpers';

import {oauthClientID} from '../shared/oauth';
import {SHARED_STYLES} from '../shared/shared-styles';
import {receiveUser} from '../state/reducers/user';
import {store} from '../state/store';


@customElement('top-bar') export class TopBar extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
    ]
  }

  render() {
    return html`
      <mwc-top-app-bar-fixed>
        <mwc-icon-button slot="navigationIcon" icon="menu" @click=${
        this.toggleMenu}></mwc-icon-button>
        <h3 slot="title">Manual Repair Records</h3>
        <chops-signin
          slot="actionItems"
          client-id="${oauthClientID}">
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

  toggleMenu() {
    let menu = <Drawer>(<Element>this.getRootNode()).querySelector('#menu');
    menu.open = !menu.open;
  }
}
