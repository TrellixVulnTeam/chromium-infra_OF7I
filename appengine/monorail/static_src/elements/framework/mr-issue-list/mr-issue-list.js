// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import page from 'page';
import {connectStore, store} from 'reducers/base.js';
import * as project from 'reducers/project.js';
import * as issue from 'reducers/issue.js';
import 'elements/framework/links/mr-issue-link/mr-issue-link.js';
import 'elements/framework/links/mr-crbug-link/mr-crbug-link.js';
import 'elements/framework/mr-dropdown/mr-dropdown.js';
import 'elements/framework/mr-star-button/mr-star-button.js';
import {issueRefToUrl, issueRefToString, issueStringToRef,
  issueToIssueRef, issueToIssueRefString,
  labelRefsToOneWordLabels} from 'shared/converters.js';
import {isTextInput, findDeepEventTarget} from 'shared/dom-helpers.js';
import {urlWithNewParams, pluralize, setHasAny,
  objectValuesForKeys} from 'shared/helpers.js';
import {parseColSpec,
  EMPTY_FIELD_VALUE} from 'shared/issue-fields.js';
import './mr-show-columns-dropdown.js';

const COLUMN_DISPLAY_NAMES = {
  'summary': 'Summary + Labels',
};

/** @type {Number} Button property value of DOM click event */
const PRIMARY_BUTTON = 0;
/** @type {Number} Button property value of DOM auxclick event */
const MIDDLE_BUTTON = 1;

/**
 * Really high cardinality attributes like ID and Summary are unlikely to be
 * useful if grouped, so it's better to just hide the option.
 */
const UNGROUPABLE_COLUMNS = new Set(['id', 'summary']);

/**
 * Converts input data array into csv formatted string
 * using already implemented data extractor
 * @param {Array<Issue>} data
 * @param {Array<string>} columns
 * @param {function(Issue, string): Array<string>} dataExtractor
 * @return {string}
 */
export const convertListToCsv = (data, columns, dataExtractor) => {
  // Returning sample data for now. To be replaced in follow up CL.
  return 'Feature,development\nin,progress.\nPlease,ignore.';
};

/** @type {String} CSV download link's data href prefix */
const CSV_DATA_HREF_PREFIX = 'data:attachment/csv;charset=utf-8,';

/**
 * Constructs download link url from csv string data.
 * @param {string} data CSV data
 * @return {string}
 */
export const constructHref = (data = '') => {
  return `${CSV_DATA_HREF_PREFIX}${encodeURIComponent(data)}`;
};

/**
 * `<mr-issue-list>`
 *
 * A list of issues intended to be used in multiple contexts.
 * @extends {LitElement}
 */
export class MrIssueList extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: table;
        width: 100%;
        font-size: var(--chops-main-font-size);
      }
      .edit-widget-container {
        display: flex;
        flex-wrap: no-wrap;
        align-items: center;
      }
      mr-star-button {
        --mr-star-button-size: 18px;
        margin-bottom: 1px;
        margin-left: 4px;
      }
      input[type="checkbox"] {
        cursor: pointer;
        margin: 0 4px;
        width: 16px;
        height: 16px;
        border-radius: 2px;
        box-sizing: border-box;
        appearance: none;
        -webkit-appearance: none;
        border: 2px solid var(--chops-gray-400);
        position: relative;
        background: white;
      }
      th input[type="checkbox"] {
        border-color: var(--chops-gray-500);
      }
      input[type="checkbox"]:checked {
        background: var(--chops-primary-accent-color);
        border-color: var(--chops-primary-accent-color);
      }
      input[type="checkbox"]:checked::after {
        left: 1px;
        top: 2px;
        position: absolute;
        content: "";
        width: 8px;
        height: 4px;
        border: 2px solid white;
        border-right: none;
        border-top: none;
        transform: rotate(-45deg);
      }
      td, th.group-header {
        padding: 4px 8px;
        text-overflow: ellipsis;
        border-bottom: var(--chops-normal-border);
        cursor: pointer;
        font-weight: normal;
      }
      .group-header-content {
        height: 100%;
        width: 100%;
        align-items: center;
        display: flex;
      }
      th.group-header i.material-icons {
        font-size: var(--chops-icon-font-size);
        color: var(--chops-primary-icon-color);
        margin-right: 4px;
      }
      td.ignore-navigation {
        cursor: default;
      }
      th {
        background: var(--chops-table-header-bg);
        white-space: nowrap;
        text-align: left;
        z-index: 10;
        border-bottom: var(--chops-normal-border);
      }
      th.first-column {
        padding: 3px 8px;
      }
      th > mr-dropdown, th > mr-show-columns-dropdown {
        font-weight: normal;
        color: var(--chops-link-color);
        --mr-dropdown-icon-color: var(--chops-link-color);
        --mr-dropdown-anchor-padding: 3px 8px;
        --mr-dropdown-anchor-font-weight: bold;
        --mr-dropdown-menu-min-width: 150px;
      }
      tr {
        padding: 0 8px;
      }
      tr[selected] {
        background: var(--chops-selected-bg);
      }
      .first-column {
        border-left: 4px solid transparent;
      }
      tr[cursored] > td.first-column {
        border-left: 4px solid var(--chops-blue-700);
      }
      mr-crbug-link {
        visibility: hidden;
      }
      td:hover > mr-crbug-link {
        visibility: visible;
      }
      .col-summary, .header-summary {
        /* Setting a table cell to 100% width makes it take up
         * all remaining space in the table, not the full width of
         * the table. */
        width: 100%;
      }
      .summary-label {
        display: inline-block;
        margin: 0 2px;
        color: var(--chops-green-800);
        text-decoration: none;
        font-size: 90%;
      }
      .summary-label:hover {
        text-decoration: underline;
      }

      .csv-download-container {
        border-bottom: none;
        text-align: end;
        cursor: default;
        /* Hiding until the function is ready */
        display: none;
      }

      #hidden-data-link {
        display: none;
      }

      @media (min-width: 1024px) {
        .first-row th {
          position: sticky;
          top: var(--monorail-header-height);
        }
      }
    `;
  }

  /** @override */
  render() {
    const selectAllChecked = this._selectedIssues.size > 0;

    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <tbody>
        <tr class="first-row">
          <th class="first-column">
            <div class="edit-widget-container">
              ${this.selectionEnabled ? html`
                <input
                  class="select-all"
                  .checked=${selectAllChecked}
                  type="checkbox"
                  aria-label="Select ${selectAllChecked ? 'All' : 'None'}"
                  @change=${this._selectAll}
                />
              ` : ''}
            </div>
          </th>
          ${this.columns.map((column, i) => this._renderHeader(column, i))}
          <th style="z-index: ${this.highestZIndex};">
            <mr-show-columns-dropdown
              title="Show columns"
              menuAlignment="right"
              .columns=${this.columns}
              .queryParams=${this.queryParams}
              .phaseNames=${this._phaseNames}
            ></mr-show-columns-dropdown>
          </th>
        </tr>
        ${this._renderIssues()}
      </tbody>
      <tfoot><tr><td colspan=999 class="csv-download-container">
        <a id="download-link" @click=${this._downloadCsv} href>CSV</a>
        <a id="hidden-data-link" download="monorail-issues.csv"
          href=${this._csvDataHref}></a>
      </td></tr></tfoot>
    `;
  }

  /**
   * @param {string} column
   * @param {number} i The index of the column in the table.
   * @return {TemplateResult} html for header for the i-th column.
   * @private
   */
  _renderHeader(column, i) {
    // zIndex is used to render the z-index property in descending order
    const zIndex = this.highestZIndex - i;
    const colKey = column.toLowerCase();
    const name = colKey in COLUMN_DISPLAY_NAMES ? COLUMN_DISPLAY_NAMES[colKey] :
      column;
    return html`
      <th style="z-index: ${zIndex};" class="header-${colKey}">
        <mr-dropdown
          class="dropdown-${colKey}"
          .text=${name}
          .items=${this._headerActions(column, i)}
          menuAlignment="left"
        ></mr-dropdown>
      </th>`;
  }

  /**
   * @param {string} column
   * @param {number} i The index of the column in the table.
   * @return {Array<Object>} Available actions for the column.
   * @private
   */
  _headerActions(column, i) {
    const columnKey = column.toLowerCase();

    const isGroupable = !UNGROUPABLE_COLUMNS.has(columnKey);

    let showOnly = [];
    if (isGroupable) {
      const values = [...this._uniqueValuesByColumn.get(columnKey)];
      if (values.length) {
        showOnly = [{
          text: 'Show only',
          items: values.map((v) => ({
            text: v,
            handler: () => this.showOnly(column, v),
          })),
        }];
      }
    }
    const actions = [
      {
        text: 'Sort up',
        handler: () => this.updateSortSpec(column),
      },
      {
        text: 'Sort down',
        handler: () => this.updateSortSpec(column, true),
      },
      ...showOnly,
      {
        text: 'Hide column',
        handler: () => this.removeColumn(i),
      },
    ];
    if (isGroupable) {
      actions.push({
        text: 'Group rows',
        handler: () => this.addGroupBy(i),
      });
    }
    return actions;
  }

  /**
   * @return {TemplateResult}
   */
  _renderIssues() {
    // Keep track of all the groups that we've seen so far to create
    // group headers as needed.
    const {issues, groupedIssues} = this;

    if (groupedIssues) {
      // Make sure issues in groups are rendered with unique indices across
      // groups to make sure hot keys and the like still work.
      let indexOffset = 0;
      return html`${groupedIssues.map(({groupName, issues}) => {
        const template = html`
          ${this._renderGroup(groupName, issues, indexOffset)}
        `;
        indexOffset += issues.length;
        return template;
      })}`;
    }

    return html`
      ${issues.map((issue, i) => this._renderRow(issue, i))}
    `;
  }

  /**
   * @param {string} groupName
   * @param {Array<Issue>} issues
   * @param {number} iOffset
   * @return {TemplateResult}
   * @private
   */
  _renderGroup(groupName, issues, iOffset) {
    if (!this.groups.length) return html``;

    const count = issues.length;
    const groupKey = groupName.toLowerCase();
    const isHidden = this._hiddenGroups.has(groupKey);

    return html`
      <tr>
        <th
          class="group-header"
          colspan="${this.numColumns}"
          @click=${() => this._toggleGroup(groupKey)}
          aria-expanded=${(!isHidden).toString()}
        >
          <div class="group-header-content">
            <i
              class="material-icons"
              title=${isHidden ? 'Show' : 'Hide'}
            >${isHidden ? 'add' : 'remove'}</i>
            ${count} ${pluralize(count, 'issue')}: ${groupName}
          </div>
        </th>
      </tr>
      ${issues.map((issue, i) => this._renderRow(issue, iOffset + i, isHidden))}
    `;
  }

  /**
   * @param {string} groupKey Lowercase group key.
   * @private
   */
  _toggleGroup(groupKey) {
    if (this._hiddenGroups.has(groupKey)) {
      this._hiddenGroups.delete(groupKey);
    } else {
      this._hiddenGroups.add(groupKey);
    }

    // Lit-element's default hasChanged check does not notice when Sets mutate.
    this.requestUpdate('_hiddenGroups');
  }

  /**
   * @param {Issue} issue
   * @param {number} i Index within the list of issues
   * @param {boolean} [isHidden]
   * @return {TemplateResult}
   */
  _renderRow(issue, i, isHidden = false) {
    const draggable = this.rerankEnabled && this.rerankEnabled(issue);
    const rowSelected = this._selectedIssues.has(issueRefToString(issue));
    const id = issueRefToString(issue);
    const cursorId = issueRefToString(this.cursor);
    const hasCursor = cursorId === id;

    return html`
      <tr
        class="row-${i} list-row ${i === this.srcIndex ? 'dragged' : ''}"
        ?selected=${rowSelected}
        ?cursored=${hasCursor}
        ?hidden=${isHidden}
        draggable=${draggable}
        data-issue-ref=${id}
        data-index=${i}
        @dragstart=${this._dragstart}
        @dragend=${this._dragend}
        @dragover=${this._dragover}
        @drop=${this._dragdrop}
        @focus=${this._setRowAsCursorOnFocus}
        @click=${this._clickIssueRow}
        @auxclick=${this._clickIssueRow}
        @keydown=${this._keydownIssueRow}
        tabindex="0"
      >
        <td class="first-column ignore-navigation">
          <div class="edit-widget-container">
            ${draggable ? html`
              <i
                class="material-icons draggable"
                title="Drag issue"
              >drag_indicator</i>
            ` : ''}
            ${this.selectionEnabled ? html`
              <input
                class="issue-checkbox"
                .value=${id}
                .checked=${rowSelected}
                type="checkbox"
                data-index=${i}
                aria-label="Select Issue ${issue.localId}"
                @change=${this._selectIssue}
                @click=${this._selectIssueRange}
              />
            ` : ''}
            ${this.starringEnabled ? html`
              <mr-star-button
                .issueRef=${issueToIssueRef(issue)}
              ></mr-star-button>
            ` : ''}
          </div>
        </td>

        ${this.columns.map((column) => html`
          <td class="col-${column.toLowerCase()}">
            ${this._renderCell(column, issue) || EMPTY_FIELD_VALUE}
          </td>
        `)}

        <td>
          <mr-crbug-link .issue=${issue}></mr-crbug-link>
        </td>
      </tr>
    `;
  }

  /**
   * @param {string} column
   * @param {Issue} issue
   * @return {TemplateResult} Html for the given column for the given issue.
   * @private
   */
  _renderCell(column, issue) {
    // Fields that need to render more than strings happen first.
    switch (column.toLowerCase()) {
      case 'id':
        return html`
           <mr-issue-link
            .projectName=${this.projectName}
            .issue=${issue}
            .queryParams=${this.queryParams}
            short
          ></mr-issue-link>
        `;
      case 'summary':
        return html`
          ${issue.summary}
          ${labelRefsToOneWordLabels(issue.labelRefs).map(({label}) => html`
            <a
              class="summary-label"
              href="${this._baseUrl()}?q=label%3A${label}"
            >${label}</a>
          `)}
        `;
    }
    const values = this._extractFieldValuesFromIssue(issue, column);
    return values.join(', ');
  }

  /** @override */
  static get properties() {
    return {
      /**
       * Array of columns to display.
       */
      columns: {type: Array},
      /**
       * Array of columns that are used as groups for issues.
       */
      groups: {type: Array},
      /**
       * List of issues to display.
       */
      issues: {type: Array},
      /**
       * A function that takes in an issue and computes whether
       * reranking should be enabled for a given issue.
       */
      rerankEnabled: {type: Object},
      /**
       * Whether issues should be selectable or not.
       */
      selectionEnabled: {type: Boolean},
      /**
       * Whether to show issue starring or not.
       */
      starringEnabled: {type: Boolean},
      /**
       * Attribute set to make host element into a table for accessibility.
       * Do not override.
       */
      role: {
        type: String,
        reflect: true,
      },
      /**
       * A query representing the current set of matching issues in the issue
       * list. Does not necessarily match queryParams.q since queryParams.q can
       * be empty while currentQuery is set to a default project query.
       */
      currentQuery: {type: String},
      /**
       * Object containing URL parameters to be preserved when issue links are
       * clicked. This Object is only used for the purpose of preserving query
       * parameters across links, not for the purpose of evaluating the query
       * parameters themselves to get values like columns, sort, or q. This
       * separation is important because we don't want to tightly couple this
       * list component with a specific URL system.
       */
      queryParams: {type: Object},
      /**
       * The initial cursor that a list view uses. This attribute allows users
       * of the list component to specify and control the cursor. When the
       * initialCursor attribute updates, the list focuses the element specified
       * by the cursor.
       */
      initialCursor: {type: String},
      /**
       * IssueRef Object specifying which issue the user is currently focusing.
       */
      _localCursor: {type: Object},
      /**
       * Set of group keys that are currently hidden.
       */
      _hiddenGroups: {type: Object},
      /**
       * Set of all selected issues where each entry is an issue ref string.
       */
      _selectedIssues: {type: Object},
      /**
       * A function that takes in an issue and a field name and returns the
       * value for that field in the issue. This function accepts custom fields,
       * built in fields, and ad hoc fields computed from label prefixes.
       */
      _extractFieldValuesFromIssue: {type: Object},
      /**
       * List of unique phase names for all phases in issues.
       */
      _phaseNames: {type: Array},
      /**
       * CSV data in data HREF format, used to download csv
       */
      _csvDataHref: {type: String},
    };
  };

  /** @override */
  constructor() {
    super();
    /** @type {Array<Issue>} */
    this.issues = [];
    // TODO(jojwang): monorail:6336#c8, when ezt listissues page is fully
    // deprecated, remove phaseNames from mr-issue-list.
    this._phaseNames = [];
    /** @type {IssueRef} */
    this._localCursor;
    /** @type {IssueRefString} */
    this.initialCursor;
    /** @type {Set<IssueRefString>} */
    this._selectedIssues = new Set();
    /** @type {string} */
    this.projectName;
    /** @type {Object} */
    this.queryParams = {};
    /** @type {string} */
    this.currentQuery = '';
    /** @type {boolean} */
    this.selectionEnabled = false;
    /** @type {boolean} */
    this.starringEnabled = false;
    /** @type {Array} */
    this.columns = ['ID', 'Summary'];
    /** @type {Array} */
    this.groups = [];
    /**
     * @type {string}
     * Role attribute set for accessibility. Do not override.
     */
    this.role = 'table';

    /** @type {function} */
    this._boundRunListHotKeys = this._runListHotKeys.bind(this);

    /**
     * @param {Issue} _issue
     * @param {string} _fieldName
     * @return {Array<string>}
     */
    this._extractFieldValuesFromIssue = (_issue, _fieldName) => [];

    this._hiddenGroups = new Set();

    this._starredIssues = new Set();
    this._fetchingStarredIssues = false;
    this._starringIssues = new Map();

    this._uniqueValuesByColumn = new Map();

    /** @type {number} */
    this._lastSelectedCheckbox = -1;

    // Expose page.js for stubbing.
    this._page = page;
    /** @type {string} page data in csv format as data href */
    this._csvDataHref = '';
  };

  /** @override */
  stateChanged(state) {
    this._starredIssues = issue.starredIssues(state);
    this._fetchingStarredIssues =
        issue.requests(state).fetchStarredIssues.requesting;
    this._starringIssues = issue.starringIssues(state);

    this._phaseNames = (issue.issueListPhaseNames(state) || []);
    this._extractFieldValuesFromIssue = project.extractFieldValuesFromIssue(
        state);
  }

  /** @override */
  firstUpdated() {
    // Only attach an event listener once the DOM has rendered.
    window.addEventListener('keydown', this._boundRunListHotKeys);
    this._dataLink = this.shadowRoot.querySelector('#hidden-data-link');
  }

  /** @override */
  disconnectedCallback() {
    super.disconnectedCallback();

    window.removeEventListener('keydown', this._boundRunListHotKeys);
  }

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('issues')) {
      // Clear selected issues to avoid an ever-growing Set size. In the future,
      // we may want to consider saving selections across issue reloads, though,
      // such as in the case or list refreshing.
      this._selectedIssues = new Set();

      // Clear group toggle state when the list of issues changes to prevent an
      // ever-growing Set size.
      this._hiddenGroups = new Set();

      this._lastSelectedCheckbox = -1;
    }

    const valuesByColumnArgs = ['issues', 'columns',
      '_extractFieldValuesFromIssue'];
    if (setHasAny(changedProperties, valuesByColumnArgs)) {
      this._uniqueValuesByColumn = this._computeUniqueValuesByColumn(
          ...objectValuesForKeys(this, valuesByColumnArgs));
    }

    super.update(changedProperties);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('initialCursor')) {
      const ref = issueStringToRef(this.projectName, this.initialCursor);
      const row = this._getRowFromIssueRef(ref);
      if (row) {
        row.focus();
      }
    }
  }

  /**
   * Iterates through all issues in a list to sort unique values
   * across columns, for use in the "Show only" feature.
   * @param {Array} issues
   * @param {Array} columns
   * @param {function(Issue, string): Array<string>} fieldExtractor
   * @return {Map} Map where each entry has a String key for the
   *   lowercase column name and a Set value, continuing all values for
   *   that column.
   */
  _computeUniqueValuesByColumn(issues, columns, fieldExtractor) {
    const valueMap = new Map(
        columns.map((col) => [col.toLowerCase(), new Set()]));

    issues.forEach((issue) => {
      columns.forEach((col) => {
        const key = col.toLowerCase();
        const valueSet = valueMap.get(key);

        const values = fieldExtractor(issue, col);
        // Note: This allows multiple casings of the same values to be added
        // to the Set.
        values.forEach((v) => valueSet.add(v));
      });
    });
    return valueMap;
  }

  /**
   * Used for dynamically computing z-index to ensure column dropdowns overlap
   * properly.
   */
  get highestZIndex() {
    return this.columns.length + 10;
  }

  /**
   * The number of columns displayed in the table. This is the count of
   * customized columns + number of built in columns.
   */
  get numColumns() {
    return this.columns.length + 2;
  }

  /**
   * Sort issues into groups if groups are defined. The grouping feature is used
   * when the "groupby" URL parameter is set in the list view.
   */
  get groupedIssues() {
    if (!this.groups || !this.groups.length) return;

    const issuesByGroup = new Map();

    this.issues.forEach((issue) => {
      const groupName = this._groupNameForIssue(issue);
      const groupKey = groupName.toLowerCase();

      if (!issuesByGroup.has(groupKey)) {
        issuesByGroup.set(groupKey, {groupName, issues: [issue]});
      } else {
        const entry = issuesByGroup.get(groupKey);
        entry.issues.push(issue);
      }
    });
    return [...issuesByGroup.values()];
  }

  /**
   * The currently selected issue, with _localCursor overriding initialCursor.
   *
   * @return {IssueRef} The currently selected issue.
   */
  get cursor() {
    if (this._localCursor) {
      return this._localCursor;
    }
    if (this.initialCursor) {
      return issueStringToRef(this.projectName, this.initialCursor);
    }
    return {};
  }

  /**
   * Computes the name of the group that an issue belongs to. Issues are grouped
   * by fields that the user specifies and group names are generated using a
   * combination of an issue's field values for all specified groups.
   *
   * @param {Issue} issue
   * @return {string}
   */
  _groupNameForIssue(issue) {
    const groups = this.groups;
    const keyPieces = [];

    groups.forEach((group) => {
      const values = this._extractFieldValuesFromIssue(issue, group);
      if (!values.length) {
        keyPieces.push(`-has:${group}`);
      } else {
        values.forEach((v) => {
          keyPieces.push(`${group}=${v}`);
        });
      }
    });

    return keyPieces.join(' ');
  }

  /**
   * @return {Array<Issue>} Selected issues in the order they appear.
   */
  get selectedIssues() {
    return this.issues.filter((issue) =>
      this._selectedIssues.has(issueToIssueRefString(issue)));
  }

  /**
   * Update the search query to filter values matching a specific one.
   *
   * @param {string} column name of the column being filtered.
   * @param {string} value value of the field to filter by.
   */
  showOnly(column, value) {
    column = column.toLowerCase();

    // TODO(zhangtiff): Handle edge cases where column names are not
    // mapped directly to field names. For example, "AllLabels", should
    // query for "Labels".
    const querySegment = `${column}=${value}`;

    let query = this.currentQuery.trim();

    if (!query.includes(querySegment)) {
      query += ' ' + querySegment;

      this._updateQueryParams({q: query.trim()}, ['start']);
    }
  }

  /**
   * Update sort parameter in the URL based on user input.
   *
   * @param {string} column name of the column to be sorted.
   * @param {boolean} descending descending or ascending order.
   */
  updateSortSpec(column, descending = false) {
    column = column.toLowerCase();
    const oldSpec = this.queryParams.sort || '';
    const columns = parseColSpec(oldSpec.toLowerCase());

    // Remove any old instances of the same sort spec.
    const newSpec = columns.filter(
        (c) => c && c !== column && c !== `-${column}`);

    newSpec.unshift(`${descending ? '-' : ''}${column}`);

    this._updateQueryParams({sort: newSpec.join(' ')}, ['start']);
  }

  /**
   * Updates the groupby URL parameter to include a new column to group.
   *
   * @param {number} i index of the column to be grouped.
   */
  addGroupBy(i) {
    const groups = [...this.groups];
    const columns = [...this.columns];
    const groupedColumn = columns[i];
    columns.splice(i, 1);

    groups.unshift(groupedColumn);

    this._updateQueryParams({
      groupby: groups.join(' '),
      colspec: columns.join('+'),
    }, ['start']);
  }

  /**
   * Removes the column at a particular index.
   *
   * @param {number} i the issue column to be removed.
   */
  removeColumn(i) {
    const columns = [...this.columns];
    columns.splice(i, 1);
    this.reloadColspec(columns);
  }

  /**
   * Adds a new column to a particular index.
   *
   * @param {string} name of the new column added.
   */
  addColumn(name) {
    this.reloadColspec([...this.columns, name]);
  }

  /**
   * Reflects changes to the columns of an issue list to the URL, through
   * frontend routing.
   *
   * @param {Array} newColumns the new colspec to set in the URL.
   */
  reloadColspec(newColumns) {
    this._updateQueryParams({colspec: newColumns.join('+')});
  }

  /**
   * Navigates to the same URL as the current page, but with query
   * params updated.
   *
   * @param {Object} newParams keys and values of the queryParams
   * Object to be updated.
   * @param {Array} deletedParams keys to be cleared from queryParams.
   */
  _updateQueryParams(newParams = {}, deletedParams = []) {
    const url = urlWithNewParams(this._baseUrl(), this.queryParams, newParams,
        deletedParams);
    this._page(url);
  }

  /**
   * Get the current URL of the page, without query params. Useful for
   * test stubbing.
   *
   * @return {string} the URL of the list page, without params.
   */
  _baseUrl() {
    return window.location.pathname;
  }

  /**
   * Run issue list hot keys. This event handler needs to be bound globally
   * because a list cursor can be defined even when no element in the list is
   * focused.
   * @param {KeyboardEvent} e
   */
  _runListHotKeys(e) {
    if (!this.issues || !this.issues.length) return;
    const target = findDeepEventTarget(e);
    if (!target || isTextInput(target)) return;

    const key = e.key;

    const activeRow = this._getCursorElement();

    let i = -1;
    if (activeRow) {
      i = Number.parseInt(activeRow.dataset.index);

      const issue = this.issues[i];

      switch (key) {
        case 's': // Star focused issue.
          this._starIssue(issueToIssueRef(issue));
          return;
        case 'x': // Toggle selection of focused issue.
          const issueRefString = issueToIssueRefString(issue);
          this._updateSelectedIssues([issueRefString],
              !this._selectedIssues.has(issueRefString));
          return;
        case 'o': // Open current issue.
        case 'O': // Open current issue in new tab.
          this._navigateToIssue(issue, e.shiftKey);
          return;
      }
    }

    // Move up and down the issue list.
    // 'j' moves 'down'.
    // 'k' moves 'up'.
    if (key === 'j' || key === 'k') {
      if (key === 'j') { // Navigate down the list.
        i += 1;
        if (i >= this.issues.length) {
          i = 0;
        }
      } else if (key === 'k') { // Navigate up the list.
        i -= 1;
        if (i < 0) {
          i = this.issues.length - 1;
        }
      }

      const nextRow = this.shadowRoot.querySelector(`.row-${i}`);
      this._setRowAsCursor(nextRow);
    }
  }

  /**
   * @return {HTMLTableRowElement}
   */
  _getCursorElement() {
    const cursor = this.cursor;
    if (cursor) {
      // If there's a cursor set, use that instead of focus.
      return this._getRowFromIssueRef(cursor);
    }
    return;
  }

  /**
   * @param {FocusEvent} e
   */
  _setRowAsCursorOnFocus(e) {
    this._setRowAsCursor(/** @type {HTMLTableRowElement} */ (e.target));
  }

  /**
   *
   * @param {HTMLTableRowElement} row
   */
  _setRowAsCursor(row) {
    this._localCursor = issueStringToRef(this.projectName,
        row.dataset.issueRef);
    row.focus();
  }

  /**
   * @param {IssueRef} ref The issueRef to query for.
   * @return {HTMLTableRowElement}
   */
  _getRowFromIssueRef(ref) {
    return this.shadowRoot.querySelector(
        `.list-row[data-issue-ref="${issueRefToString(ref)}"]`);
  }

  /**
   * @param {IssueRef} issueRef Issue to star
   */
  _starIssue(issueRef) {
    if (!this.starringEnabled) return;
    const issueKey = issueRefToString(issueRef);

    // TODO(zhangtiff): Find way to share star disabling logic more.
    const isStarring = this._starringIssues.has(issueKey) &&
      this._starringIssues.get(issueKey).requesting;
    const starEnabled = !this._fetchingStarredIssues && !isStarring;
    if (starEnabled) {
      const newIsStarred = !this._starredIssues.has(issueKey);
      this._starIssueInternal(issueRef, newIsStarred);
    }
  }

  /**
   * Wrap store.dispatch and issue.star, for testing.
   *
   * @param {IssueRef} issueRef the issue being starred.
   * @param {boolean} newIsStarred whether to star or unstar the issue.
   */
  _starIssueInternal(issueRef, newIsStarred) {
    store.dispatch(issue.star(issueRef, newIsStarred));
  }
  /**
   * @param {Event} e
   */
  _selectAll(e) {
    const checkbox = /** @type {HTMLInputElement} */ (e.target);

    if (checkbox.checked) {
      this._selectedIssues = new Set(this.issues.map(issueRefToString));
    } else {
      this._selectedIssues = new Set();
    }
    this.dispatchEvent(new CustomEvent('selectionChange'));
  }

  // TODO(zhangtiff): Implement Shift+Click to select a range of checkboxes
  // for the 'x' hot key.
  /**
   * @param {MouseEvent} e
   */
  _selectIssueRange(e) {
    if (!this.selectionEnabled) return;

    const checkbox = /** @type {HTMLInputElement} */ (e.target);

    const index = Number.parseInt(checkbox.dataset.index);
    if (Number.isNaN(index)) {
      console.error('Issue checkbox has invalid data-index attribute.');
      return;
    }

    const lastIndex = this._lastSelectedCheckbox;
    if (e.shiftKey && lastIndex >= 0) {
      const newCheckedState = checkbox.checked;

      const start = Math.min(lastIndex, index);
      const end = Math.max(lastIndex, index) + 1;

      const updatedIssueKeys = this.issues.slice(start, end).map(
          issueToIssueRefString);
      this._updateSelectedIssues(updatedIssueKeys, newCheckedState);
    }

    this._lastSelectedCheckbox = index;
  }

  /**
   * @param {Event} e
   */
  _selectIssue(e) {
    if (!this.selectionEnabled) return;

    const checkbox = /** @type {HTMLInputElement} */ (e.target);
    const issueKey = checkbox.value;

    this._updateSelectedIssues([issueKey], checkbox.checked);
  }

  /**
   * @param {Array<IssueRefString>} issueKeys Stringified issue refs.
   * @param {boolean} selected
   */
  _updateSelectedIssues(issueKeys, selected) {
    let hasChanges = false;

    issueKeys.forEach((issueKey) => {
      const oldSelection = this._selectedIssues.has(issueKey);

      if (selected) {
        this._selectedIssues.add(issueKey);
      } else if (this._selectedIssues.has(issueKey)) {
        this._selectedIssues.delete(issueKey);
      }

      const newSelection = this._selectedIssues.has(issueKey);

      hasChanges = hasChanges || newSelection !== oldSelection;
    });


    if (hasChanges) {
      this.requestUpdate('_selectedIssues');
      this.dispatchEvent(new CustomEvent('selectionChange'));
    }
  }

  /**
   * Handles 'Enter' being pressed when a row is focused.
   * Note we install the 'Enter' listener on the row rather than the window so
   * 'Enter' behaves as expected when the focus is on other elements.
   *
   * @param {KeyboardEvent} e
   */
  _keydownIssueRow(e) {
    if (e.key === 'Enter') {
      this._maybeOpenIssueRow(e);
    }
  }

  /**
   * Handle click and auxclick on issue row
   * @param {MouseEvent} event
   */
  _clickIssueRow(event) {
    if (event.button === PRIMARY_BUTTON || event.button === MIDDLE_BUTTON) {
      this._maybeOpenIssueRow(
          event, /* openNewTab= */ event.button === MIDDLE_BUTTON);
    }
  }

  /**
   * Checks that the given event should not be ignored, then navigates to the
   * issue associated with the row.
   *
   * @param {MouseEvent|KeyboardEvent} rowEvent A click or 'enter' on a row.
   * @param {boolean} [openNewTab] Forces opening in a new tab
   */
  _maybeOpenIssueRow(rowEvent, openNewTab = false) {
    const path = rowEvent.composedPath();
    const containsIgnoredElement = path.find(
        (node) => (node.tagName || '').toUpperCase() === 'A' ||
        (node.classList && node.classList.contains('ignore-navigation')));
    if (containsIgnoredElement) return;

    const row = /** @type {HTMLTableRowElement} */ (rowEvent.currentTarget);

    const i = Number.parseInt(row.dataset.index);

    if (i >= 0 && i < this.issues.length) {
      this._navigateToIssue(this.issues[i], openNewTab || rowEvent.metaKey ||
          rowEvent.ctrlKey);
    }
  }

  /**
   * @param {Issue} issue
   * @param {boolean} newTab
   */
  _navigateToIssue(issue, newTab) {
    const link = issueRefToUrl(issueToIssueRef(issue),
        this.queryParams);

    if (newTab) {
      // Whether the link opens in a new tab or window is based on the
      // user's browser preferences.
      window.open(link, '_blank', 'noopener');
    } else {
      this._page(link);
    }
  }

  /**
   * Download content as csv. Conversion to CSV only on button click
   * instead of on data change because CSV download is not often used.
   * @param {MouseEvent} event
   */
  async _downloadCsv(event) {
    event.preventDefault();

    // convert the data, this.issues, into csv formatted string.
    const csvDataString = convertListToCsv(
        this.issues,
        this.columns,
        this._extractFieldValuesFromIssue);

    // construct data href
    const href = constructHref(csvDataString);

    // modify a tag's href
    this._csvDataHref = href;
    await this.requestUpdate('_csvDataHref');

    // click to trigger download
    this._dataLink.click();

    // reset dataHref
    this._csvDataHref = '';
  }
};

customElements.define('mr-issue-list', MrIssueList);
