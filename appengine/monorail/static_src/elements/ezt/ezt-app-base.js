// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement} from 'lit-element';
import qs from 'qs';
import {store, connectStore} from 'reducers/base.js';
import * as userV0 from 'reducers/userV0.js';
import * as projectV0 from 'reducers/projectV0.js';
import * as sitewide from 'reducers/sitewide.js';

/**
 * `<ezt-app-base>`
 *
 * Base component meant to simulate a subset of the work mr-app does on
 * EZT pages in order to allow us to more easily glue web components
 * on EZT pages to SPA web components.
 *
 */
export class EztAppBase extends connectStore(LitElement) {
  /** @override */
  static get properties() {
    return {
      projectName: {type: String},
      userDisplayName: {type: String},
    };
  }

  /** @override */
  connectedCallback() {
    super.connectedCallback();

    this.mapUrlToQueryParams();
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('userDisplayName') && this.userDisplayName) {
      this.fetchUserData(this.userDisplayName);
    }

    if (changedProperties.has('projectName') && this.projectName) {
      this.fetchProjectData(this.projectName);
    }
  }

  fetchUserData(displayName) {
    store.dispatch(userV0.fetch(displayName));
  }

  fetchProjectData(projectName) {
    store.dispatch(projectV0.select(projectName));
    store.dispatch(projectV0.fetch(projectName));
  }

  mapUrlToQueryParams() {
    const params = qs.parse((window.location.search || '').substr(1));

    store.dispatch(sitewide.setQueryParams(params));
  }
}
customElements.define('ezt-app-base', EztAppBase);
