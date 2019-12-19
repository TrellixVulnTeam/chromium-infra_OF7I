// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {connectStore} from 'reducers/base.js';
import * as project from 'reducers/project.js';
import * as sitewide from 'reducers/sitewide.js';
import '../mr-mode-selector/mr-mode-selector.js';
import '../mr-chart/mr-chart.js';

/**
 * <mr-chart-page>
 *
 * Chart page view containing mr-mode-selector and mr-chart.
 * @extends {LitElement}
 */
export class MrChartPage extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
        box-sizing: border-box;
        width: 100%;
        padding: 0.5em 8px;
      }
      h2 {
        font-size: 1.2em;
        margin: 0 0 0.5em;
      }
      .list-controls {
        display: flex;
        align-items: center;
        justify-content: flex-end;
        width: 100%;
        padding: 0.5em 0;
      }
      .help {
        padding: 1em;
        background-color: rgb(227, 242, 253);
        width: 44em;
        font-size: 92%;
        margin: 5px;
        padding: 6px;
        border-radius: 6px;
      }
      .monospace {
        font-family: monospace;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <div class="list-controls">
        <mr-mode-selector
          .projectName=${this._projectName}
          .queryParams=${this._queryParams}
          .value=${'chart'}
        ></mr-mode-selector>
      </div>
      <mr-chart
        .projectName=${this._projectName}
        .queryParams=${this._queryParams}
      ></mr-chart>

      <div>
        <div class="help">
          <h2>Supported query parameters:</h2>
          <span class="monospace">
            cc, component, hotlist, label, owner, reporter, status
          </span>
          <br /><br />
          <a href="https://bugs.chromium.org/p/monorail/issues/entry?labels=Feature-Charts">
            Please file feedback here.
          </a>
        </div>
      </div>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _projectName: {type: String},
      /** @private {Object} */
      _queryParams: {type: Object},
    };
  }

  /** @override */
  stateChanged(state) {
    this._projectName = project.viewedProjectName(state);
    this._queryParams = sitewide.queryParams(state);
  }
};
customElements.define('mr-chart-page', MrChartPage);
