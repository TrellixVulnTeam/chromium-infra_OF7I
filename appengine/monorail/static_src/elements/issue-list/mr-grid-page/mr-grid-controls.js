// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import page from 'page';
import {connectStore} from 'reducers/base.js';
import * as issue from 'reducers/issue.js';
import 'elements/chops/chops-choice-buttons/chops-choice-buttons.js';
import '../mr-mode-selector/mr-mode-selector.js';
import './mr-grid-dropdown.js';
import {urlWithNewParams} from 'shared/helpers.js';
import {fieldsForIssue} from 'shared/issue-fields.js';

// A list of the valid default field names available in an issue grid.
// High cardinality fields must be excluded, so the grid only includes a subset
// of AVAILABLE FIELDS.
export const DEFAULT_GRID_FIELDS = Object.freeze([
  'Project',
  'Attachments',
  'Blocked',
  'BlockedOn',
  'Blocking',
  'Component',
  'MergedInto',
  'Reporter',
  'Stars',
  'Status',
  'Type',
  'Owner',
]);

/**
 * Component for displaying the controls shown on the Monorail issue grid page.
 * @extends {LitElement}
 */
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
    this.gridOptions = this._computeGridOptions([]);
    this.queryParams = {};

    this.totalIssues = 0;

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
      _issues: {type: Array},
    };
  };

  /** @override */
  stateChanged(state) {
    this.totalIssues = issue.totalIssues(state) || 0;
    this._issues = issue.issueList(state) || [];
  }

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('_issues')) {
      this.gridOptions = this._computeGridOptions(this._issues);
    }
    super.update(changedProperties);
  }

  /**
   * Gets what issue filtering options exist on the grid view.
   * @param {Array<Issue>} issues The issues to find values on.
   * @param {Array<string>=} defaultFields Available built in fields.
   * @return {Array<string>} Array of names of fields you can filter by.
   */
  _computeGridOptions(issues, defaultFields = DEFAULT_GRID_FIELDS) {
    const availableFields = new Set(defaultFields);
    issues.forEach((issue) => {
      fieldsForIssue(issue, true).forEach((field) => {
        availableFields.add(field);
      });
    });
    const options = [...availableFields].sort();
    options.unshift('None');
    return options;
  }

  /**
   * @return {string} What cell mode the user has selected.
   * ie: Tiles, IDs, Counts
   */
  get cellType() {
    const cells = this.queryParams.cells;
    return cells || 'tiles';
  }

  /**
   * @return {Array<Object>} Cell options available to the user, formatted for
   *   <mr-mode-selector>
   */
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

  /**
   * Changes the URL parameters on the page in response to a user changing
   * their row setting.
   * @param {Event} e 'change' event fired by <mr-grid-dropdown>
   */
  _rowChanged(e) {
    const y = e.target.selection;
    let deletedParams;
    if (y === 'None') {
      deletedParams = ['y'];
    }
    this._changeUrlParams({y}, deletedParams);
  }

  /**
   * Changes the URL parameters on the page in response to a user changing
   * their col setting.
   * @param {Event} e 'change' event fired by <mr-grid-dropdown>
   */
  _colChanged(e) {
    const x = e.target.selection;
    let deletedParams;
    if (x === 'None') {
      deletedParams = ['x'];
    }
    this._changeUrlParams({x}, deletedParams);
  }

  /**
   * Helper method to update URL params with a new grid view URL.
   * @param {Array<Object>} newParams
   * @param {Array<string>} deletedParams
   */
  _changeUrlParams(newParams, deletedParams) {
    const newUrl = this._updatedGridViewUrl(newParams, deletedParams);
    this._page(newUrl);
  }

  /**
   * Helper to generate a new grid view URL given a set of params.
   * @param {Array<Object>} newParams
   * @param {Array<string>} deletedParams
   * @return {string} The generated URL.
   */
  _updatedGridViewUrl(newParams, deletedParams) {
    return urlWithNewParams(`/p/${this.projectName}/issues/list`,
        this.queryParams, newParams, deletedParams);
  }
};

customElements.define('mr-grid-controls', MrGridControls);
