// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-list';
import '@material/mwc-list/mwc-list-item';
import '@material/mwc-drawer';
import './repair-form';
import './search-hostname';
import './top-bar';

import {Drawer} from '@material/mwc-drawer';
import {css, customElement, html, LitElement, property} from 'lit-element';
import {TemplateResult} from 'lit-html';
import {installRouter} from 'pwa-helpers/router.js';


@customElement('manual-repair')
export class ManualRepair extends LitElement {
  static get styles() {
    return [css`
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
    `];
  }

  static TITLE: Array<String> = ['Home', 'Repairs'];

  @property({type: String}) path = '';

  constructor() {
    super();
    installRouter((loc) => {
      this.path = loc.pathname;
    });
  }

  pageLinks() {
    let menu: Array<TemplateResult> = [];
    for (let i = 0; i < ManualRepair.TITLE.length; i++) {
      let link = '/' + ManualRepair.TITLE[i].toLowerCase();
      menu.push(html`
                <a href=${link} @click=${this.toggleMenu} class="page-link">
                    <mwc-list-item>${ManualRepair.TITLE[i]}</mwc-list-item>
                </a>
            `);
    }
    return html`<mwc-list activatable>${menu}</mwc-list>`;
  }

  toggleMenu() {
    let menu = <Drawer>this.shadowRoot!.querySelector('#menu');
    menu.open = !menu.open;
  }

  render() {
    return html`
      <mwc-drawer hasHeader type="modal" id="menu">
        <span slot="title">Menu</span>
        <span slot="subtitle">Welcome</span>
        ${this.pageLinks()}
        <div slot="appContent">
          <top-bar></top-bar>
          <div id="app-body">
            <search-hostname></search-hostname>
            <repair-form></repair-form>
          </div>
        </div>
      </mwc-drawer>
    `;
  }
}
