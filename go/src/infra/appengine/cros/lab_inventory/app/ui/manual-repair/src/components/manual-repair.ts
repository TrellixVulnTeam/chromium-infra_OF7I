// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-list';
import '@material/mwc-list/mwc-list-item';
import '@material/mwc-drawer';
import './repair-form/repair-form';
import './top-bar';
import './message-display';
import './home-view';
import './dashboard-view';

import {Drawer} from '@material/mwc-drawer';
import {css, customElement, html, LitElement, property} from 'lit-element';
import {TemplateResult} from 'lit-html';
import {isEmpty} from 'lodash';
import {connect} from 'pwa-helpers';

import {parseQueryStringToDict} from '../shared/helpers/query-store-helpers';
import {router} from '../shared/router';
import {SHARED_STYLES} from '../shared/shared-styles';
import {receiveQueryStore} from '../state/reducers/query';
import {store, thunkDispatch} from '../state/store';


@customElement('manual-repair')
export default class ManualRepair extends connect
(store)(LitElement) {
  static get styles() {
    return [
      SHARED_STYLES,
      css`
      #app-body {
        width: 75%;
        height: calc(100vh - 64px);
        margin: auto;
        padding: 1.8em 0 3em 0;
        display: flex;
        flex-direction: column;
        overflow-y: hidden;
      }

      .page-link {
        text-decoration: none;
      }
    `,
    ];
  }

  @property({type: Object}) user;
  // this.route will be used to determine which view should be displayed based
  // on the router path.
  @property({type: Object}) route;

  constructor() {
    super();

    // Router routes defined here.
    router
        .on('dashboard',
            () => {this.route = html`<dashboard-view></dashboard-view>`})
        .on('repairs',
            (_, query) => {
              this.route = html`<repair-form></repair-form>`;
              thunkDispatch(receiveQueryStore(parseQueryStringToDict(query)));
            })
        .on('*', () => {this.route = html`<home-view></home-view>`});
    router.resolve();
  }

  stateChanged(state) {
    this.user = state.user;
  }

  static AppLinks: Map<string, {[key: string]: string}> = new Map([
    [
      'Home', {
        link: '/#/home',
        target: '',
        icon: 'home',
      }
    ],
    [
      'Dashboard', {
        link: '/#/dashboard',
        target: '',
        icon: 'dashboard',
      }
    ],
    [
      'Repair Form', {
        link: '/#/repairs',
        target: '',
        icon: 'assignment',
      }
    ],
  ]);

  static MiscLinks: Map<string, {[key: string]: string}> = new Map([
    [
      'Feature Requests', {
        link: 'http://go/mrfeatures',
        target: '_blank',
        icon: 'feedback',
      }
    ],
    [
      'Report Blockers', {
        link: 'http://go/mr-must-fix',
        target: '_blank',
        icon: 'warning',
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
        isEmpty(this.user.profile) ? 'Welcome' : this.user.profile.getName();
    const drawerSubtitle = isEmpty(this.user.profile) ?
        'Please log in!' :
        this.user.profile.getEmail();

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
            ${this.route}
          </div>
          <message-display></message-display>
        </div>
      </mwc-drawer>
    `;
  }
}
