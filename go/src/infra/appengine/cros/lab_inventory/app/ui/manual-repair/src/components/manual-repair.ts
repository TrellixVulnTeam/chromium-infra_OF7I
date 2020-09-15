// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-list';
import '@material/mwc-list/mwc-list-item';
import '@material/mwc-drawer';
import './repair-form/repair-form';
import './search-hostname';
import './top-bar';
import './message-display';

import {Drawer} from '@material/mwc-drawer';
import {css, customElement, html, LitElement, property} from 'lit-element';
import {TemplateResult} from 'lit-html';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';
import {installRouter} from 'pwa-helpers/router.js';

import {SHARED_STYLES} from '../shared/shared-styles';
import {store} from '../state/store';


@customElement('manual-repair')
export default class ManualRepair extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
      #app-body {
        width: 70%;
        margin: auto;
        padding: 1.8em 0 3em 0;
        display: flex;
        flex-direction: column;
        justify-content: center;
      }

      .page-link {
        text-decoration: none;
      }
    `,
    ];
  }

  @property({type: String}) path = '';
  @property({type: Object}) user;

  constructor() {
    super();
    installRouter((loc) => {
      this.path = loc.pathname;
    });
  }

  stateChanged(state) {
    this.user = state.user;
  }

  static AppLinks: Map<string, {[key: string]: string}> = new Map([
    [
      'Home', {
        link: '/home',
        target: '',
        icon: 'home',
      }
    ],
    [
      'Repairs', {
        link: '/repairs',
        target: '',
        icon: 'view_list',
      }
    ],
  ]);

  static MiscLinks: Map<string, {[key: string]: string}> = new Map([
    [
      'Feedback', {
        link:
            'https://bugs.chromium.org/p/chromium/issues/entry?status=Unassigned&summary=Manual%20Repair%20App%20Feedback%20-%20Summarize%20issue%20here&labels=Type-Bug,Pri-2,AdminRepair&comment=What%20feedback%20would%20you%20like%20to%20provide?&components=Infra%3EFleet%3ESoftware%3EAutomation,%20Infra%3EFleet%3ESoftware%3EInventory',
        target: '_blank',
        icon: 'feedback',
      }
    ],
  ]);

  buildLinksHtml(linksMap: Map<string, {[key: string]: string}>) {
    const links: Array<TemplateResult> = [];
    for (const [key, obj] of linksMap.entries()) {
      links.push(html`
                <a href=${obj.link}
                  target=${obj.target}
                  class="page-link"
                  @click=${this.toggleMenu}>
                    <mwc-list-item graphic="icon">
                      <slot>${key}</slot>
                      <mwc-icon slot="graphic">${obj.icon}</mwc-icon>
                    </mwc-list-item>
                </a>
              `);
    }
    return html`
      ${links}
    `;
  }

  toggleMenu() {
    const menu = <Drawer>this.shadowRoot!.querySelector('#menu');
    menu.open = !menu.open;
  }

  render() {
    const drawerTitle =
        isEmpty(this.user.profile) ? 'Welcome' : this.user.profile.Ad;
    const drawerSubtitle =
        isEmpty(this.user.profile) ? 'Please log in!' : this.user.profile.$t;

    return html`
      <mwc-drawer hasHeader type="modal" id="menu">
        <span slot="title">${drawerTitle}</span>
        <span slot="subtitle">${drawerSubtitle}</span>
        <mwc-list activatable>
          <li divider role="separator"></li>
          ${this.buildLinksHtml(ManualRepair.AppLinks)}
          <li divider role="separator"></li>
          ${this.buildLinksHtml(ManualRepair.MiscLinks)}
        </mwc-list>
        <div slot="appContent">
          <top-bar></top-bar>
          <div id="app-body">
            <search-hostname></search-hostname>
            <repair-form></repair-form>
          </div>
          <message-display></message-display>
        </div>
      </mwc-drawer>
    `;
  }
}
