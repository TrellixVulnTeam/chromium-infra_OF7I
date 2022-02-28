// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import { customElement, html, LitElement } from 'lit-element';

import '../../shared_elements/title_bar'
import { BeforeEnterObserver, RouterLocation } from '@vaadin/router';
import { ifDefined } from 'lit/directives/if-defined.js';

declare global {
    interface Window {
        email: string;
        logoutUrl: string;
    }
}

@customElement("base-view")
export class BaseView extends LitElement implements BeforeEnterObserver {

    private project: string | null = null;

    onBeforeEnter(location: RouterLocation) {
        if(location.params.project !== undefined) {
            this.project = typeof location.params.project == 'string' ? location.params.project : location.params.project[0];
        }
    }

    render() {
        return html`
          <title-bar email="${window.email}" logouturl="${window.logoutUrl}" project=${ifDefined(this.project === null ? undefined : this.project)}></title-bar>
          <slot></slot>
        `;
    }

}
