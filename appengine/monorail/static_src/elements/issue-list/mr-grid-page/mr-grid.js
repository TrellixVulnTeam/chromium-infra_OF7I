// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {connectStore} from 'reducers/base.js';
import * as project from 'reducers/project.js';
import {extractGridData, makeGridCellKey} from './extract-grid-data.js';
import {EMPTY_FIELD_VALUE} from 'shared/issue-fields.js';
import {issueRefToUrl} from 'shared/converters.js';
import './mr-grid-tile.js';
import qs from 'qs';

/**
 * <mr-grid>
 *
 * A grid of issues grouped optionally horizontally and vertically.
 *
 * Throughout the file 'x' corresponds to column headers and 'y' corresponds to
 * row headers.
 *
 * @extends {LitElement}
 */
export class MrGrid extends connectStore(LitElement) {
  /** @override */
  render() {
    return html`
      <table>
        <tr>
          <th>&nbsp</th>
          ${this._xHeadings.map((heading) => html`
              <th>${heading}</th>`)}
        </tr>
        ${this._yHeadings.map((yHeading) => html`
          <tr>
            <th>${yHeading}</th>
            ${this._xHeadings.map((xHeading) => html`
                ${this._renderCell(xHeading, yHeading)}`)}
          </tr>
        `)}
      </table>
    `;
  }
  /**
   *
   * @param {string} xHeading
   * @param {string} yHeading
   * @return {TemplateResult}
   */
  _renderCell(xHeading, yHeading) {
    const cell = this._groupedIssues.get(makeGridCellKey(xHeading, yHeading));
    if (!cell) {
      return html`<td></td>`;
    }

    const cellMode = this.cellMode.toLowerCase();
    let content;
    if (cellMode === 'ids') {
      content = html`
        ${cell.map((issue) => html`
          <mr-issue-link
            .projectName=${this.projectName}
            .issue=${issue}
            .text=${issue.localId}
            .queryParams=${this.queryParams}
          ></mr-issue-link>
        `)}
      `;
    } else if (cellMode === 'counts') {
      const itemCount = cell.length;
      if (itemCount === 1) {
        const issue = cell[0];
        content = html`
          <a href=${issueRefToUrl(issue, this.queryParams)} class="counts">
            1 item
          </a>
        `;
      } else {
        content = html`
          <a href=${this._formatListUrl(xHeading, yHeading)} class="counts">
            ${itemCount} items
          </a>
        `;
      }
    } else {
      // Default to tiles.
      content = html`
        ${cell.map((issue) => html`
          <mr-grid-tile
            .issue=${issue}
            .queryParams=${this.queryParams}
          ></mr-grid-tile>
          `)}
        `;
    }
    return html`<td>${content}</td>`;
  }

  /**
   * Creates a URL to the list view for the group of issues corresponding to
   * the given headings.
   *
   * @param {string} xHeading
   * @param {string} yHeading
   * @return {string}
   */
  _formatListUrl(xHeading, yHeading) {
    let url = 'list?';
    const params = Object.assign({}, this.queryParams);
    params.mode = '';

    params.q = this._addHeadingToQuery(params.q, xHeading, this.xField);
    params.q = this._addHeadingToQuery(params.q, yHeading, this.yField);

    url += qs.stringify(params);

    return url;
  }

  /**
   * @param {string} query
   * @param {string} heading The value of field for the current group.
   * @param {string} field Field on which we're grouping the issue.
   * @return {string} The query with an additional clause if needed.
   */
  _addHeadingToQuery(query, heading, field) {
    if (field && field !== 'None') {
      if (heading === EMPTY_FIELD_VALUE) {
        query += ' -has:' + field;
      // The following two cases are to handle grouping issues by Blocked
      } else if (heading === 'No') {
        query += ' -is:' + field;
      } else if (heading === 'Yes') {
        query += ' is:' + field;
      } else {
        query += ' ' + field + '=' + heading;
      }
    }
    return query;
  }

  /** @override */
  static get properties() {
    return {
      xField: {type: String},
      yField: {type: String},
      issues: {type: Array},
      cellMode: {type: String},
      queryParams: {type: Object},
      projectName: {type: String},
      _fieldDefMap: {type: Object},
      _labelPrefixSet: {type: Object},
    };
  }

  /** @override */
  static get styles() {
    return css`
      table {
        table-layout: auto;
        border-collapse: collapse;
        width: 98%;
        margin: 0.5em 1%;
        text-align: left;
      }
      th {
        border: 1px solid white;
        padding: 5px;
        background-color: var(--chops-table-header-bg);
        white-space: nowrap;
      }
      td {
        border: var(--chops-table-divider);
        padding-left: 0.3em;
        background-color: white;
        vertical-align: top;
      }
      mr-issue-link {
        display: inline-block;
        margin-right: 8px;
      }
    `;
  }

  /** @override */
  constructor() {
    super();
    /** @type {string} */
    this.cellMode = 'tiles';
    /** @type {Array<Issue>} */
    this.issues = [];
    /** @type {string} */
    this.projectName;
    this.queryParams = {};

    /** @type {string} The issue field on which to group columns. */
    this.xField;

    /** @type {string} The issue field on which to group rows. */
    this.yField;

    /**
     * Grid cell key mapped to issues associated with that cell.
     * @type {Map<string, Array<Issue>>}
     */
    this._groupedIssues = new Map();

    /** @type {Array<string>} */
    this._xHeadings = [];

    /** @type {Array<string>} */
    this._yHeadings = [];

    /**
     * Lowercase field name -> FieldDef for the project
     * @type {Map<string, FieldDef>}
     */
    this._fieldDefMap = new Map();

    this._labelPrefixSet = new Set();
  }

  /** @override */
  stateChanged(state) {
    this._fieldDefMap = project.fieldDefMap(state);
    this._labelPrefixSet = project.labelPrefixSet(state);
  }

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('xField') || changedProperties.has('yField') ||
        changedProperties.has('issues') ||
        changedProperties.has('_fieldDefMap') ||
        changedProperties.has('_labelPrefixSet')) {
      const gridData = extractGridData(this.issues, this.xField, this.yField,
          this.projectName, this._fieldDefMap, this._labelPrefixSet);
      this._xHeadings = gridData.xHeadings;
      this._yHeadings = gridData.yHeadings;
      this._groupedIssues = gridData.groupedIssues;
    }

    super.update(changedProperties);
  }
};
customElements.define('mr-grid', MrGrid);
