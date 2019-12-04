// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import page from 'page';
import {connectStore} from 'reducers/base.js';
import * as issue from 'reducers/issue.js';
import * as project from 'reducers/project.js';
import 'elements/chops/chops-choice-buttons/chops-choice-buttons.js';
import '../mr-mode-selector/mr-mode-selector.js';
import './mr-grid-dropdown.js';
import {getAvailableGridFields} from './extract-grid-data.js';
import {urlWithNewParams} from 'shared/helpers.js';

export class MrGridControls extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: flex;
        justify-content: space-between;
        margin-top: 20px;
        box-sizing: border-box;
        padding: 0 20px;
      }
      mr-grid-dropdown {
        padding-right: 20px;
      }
      .left-controls {
        display: flex;
        align-items: center;
        justify-content: flex-start;
        flex-grow: 0;
      }
      .right-controls {
        display: flex;
        align-items: center;
        flex-grow: 0;
      }
      .issue-count {
        display: inline-block;
        padding-right: 20px;
      }
    `;
  };

  /** @override */
  render() {
    const hideCounts = this.totalIssues === 0;
    return html`
      <div class="left-controls">
        <mr-grid-dropdown
          class="row-selector"
          .text=${'Rows'}
          .items=${this.gridOptions}
          .selection=${this.queryParams.y}
          @change=${this._rowChanged}>
        </mr-grid-dropdown>
        <mr-grid-dropdown
          class="col-selector"
          .text=${'Cols'}
          .items=${this.gridOptions}
          .selection=${this.queryParams.x}
          @change=${this._colChanged}>
        </mr-grid-dropdown>
        <chops-choice-buttons
          class="cell-selector"
          .options=${this.cellOptions}
          .value=${this.cellType}>
        </chops-choice-buttons>
      </div>
      <div class="right-controls">
        ${hideCounts ? '' : html`
          <div class="issue-count">
            ${this.issueCount}
            of
            ${this.totalIssues}
            ${this.totalIssues === 1 ? html`
              issue `: html`
              issues `} shown
          </div>
        `}
        <mr-mode-selector
          .projectName=${this.projectName}
          .queryParams=${this.queryParams}
          value="grid"
        ></mr-mode-selector>
      </div>
    `;
  }

  /** @override */
  constructor() {
    super();
    this.gridOptions = getAvailableGridFields();
    this.queryParams = {};

    this.totalIssues = 0;
    this._fieldDefs = [];
    this._labelPrefixFields = [];

    this._page = page;
  };

  /** @override */
  static get properties() {
    return {
      gridOptions: {type: Array},
      projectName: {tupe: String},
      queryParams: {type: Object},
      issueCount: {type: Number},
      totalIssues: {type: Number},
      _fieldDefs: {type: Array},
      _labelPrefixFields: {type: Object},
    };
  };

  /** @override */
  stateChanged(state) {
    this.totalIssues = (issue.totalIssues(state) || 0);
    this._fieldDefs = project.fieldDefs(state) || [];
    this._labelPrefixFields = project.labelPrefixFields(state) || [];
  }

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('_fieldDefs') ||
        changedProperties.has('_labelPrefixFields')) {
      this.gridOptions = getAvailableGridFields(
          this._fieldDefs, this._labelPrefixFields);
    }
    super.update(changedProperties);
  }

  get cellType() {
    const cells = this.queryParams.cells;
    return cells || 'tiles';
  }

  get cellOptions() {
    return [
      {text: 'Tile', value: 'tiles',
        url: this._updatedGridViewUrl({}, ['cells'])},
      {text: 'IDs', value: 'ids',
        url: this._updatedGridViewUrl({cells: 'ids'})},
      {text: 'Counts', value: 'counts',
        url: this._updatedGridViewUrl({cells: 'counts'})},
    ];
  }

  _rowChanged(e) {
    const y = e.target.selection;
    let deletedParams;
    if (y === 'None') {
      deletedParams = ['y'];
    }
    this._changeUrlParams({y}, deletedParams);
  }

  _colChanged(e) {
    const x = e.target.selection;
    let deletedParams;
    if (x === 'None') {
      deletedParams = ['x'];
    }
    this._changeUrlParams({x}, deletedParams);
  }

  _changeUrlParams(newParams, deletedParams) {
    const newUrl = this._updatedGridViewUrl(newParams, deletedParams);
    this._page(newUrl);
  }

  _updatedGridViewUrl(newParams, deletedParams) {
    // TODO(zhangtiff): Replace /list_new with /list when switching the new grid
    // view to default.
    return urlWithNewParams(`/p/${this.projectName}/issues/list_new`,
        this.queryParams, newParams, deletedParams);
  }
};

customElements.define('mr-grid-controls', MrGridControls);
