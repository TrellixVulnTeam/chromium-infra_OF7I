// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import 'elements/chops/chops-dialog/chops-dialog.js';
import * as userV0 from 'reducers/userV0.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';
import {connectStore} from 'reducers/base.js';

/**
 * `<mr-issue-hotlists-dialog>`
 *
 * The base dialog that <mr-move-issue-hotlists-dialog> and
 * <mr-update-issue-hotlists-dialog> inherits common methods and behaviors from.
 * <mr-update-issue-hotlists-dialog> is used across multiple pages where as
 * <mr-move-issue-hotlists-dialog> is largely used within Hotlists.
 *
 * Important: The `render` method should be overridden by child classes.
 */
export class MrIssueHotlistsDialog extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          font-size: var(--chops-main-font-size);
          --chops-dialog-max-width: 500px;
        }
        .error {
          max-width: 100%;
          color: red;
          margin-bottom: 1px;
        }
        select,
        input {
          box-sizing: border-box;
          width: var(--mr-edit-field-width);
          padding: var(--mr-edit-field-padding);
          font-size: var(--chops-main-font-size);
        }
        input#filter {
          margin-top: 4px;
          width: 85%;
          max-width: 240px;
        }
        .user-hotlists {
          max-height: 240px;
          overflow: auto;
        }
        .hotlist.filter-fail {
          display: none;
        }
        i.material-icons {
          font-size: 20px;
          margin-right: 4px;
          vertical-align: bottom;
        }
      `,
    ];
  }

  /** @override */
  render() {
    return html`
    <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
    <chops-dialog closeOnOutsideClick>
      ${this.renderHeader()}
      ${this.renderContent()}
    </chops-dialog>
    `;
  }

  /**
   * Renders the dialog header.
   * @return {TemplateResult}
   */
  renderHeader() {
    return html`
      <h3 class="medium-heading">Dialog elements below:</h3>
    `;
  }

  /**
   * Renders the dialog content.
   * @return {TemplateResult}
   */
  renderContent() {
    return html`
      ${this.renderFilter()}
      ${this.renderHotlists()}
      ${this.renderError()}
    `;
  }

  /**
   * Renders the Hotlist filter.
   * @return {TemplateResult}
   */
  renderFilter() {
    return html`
      <input id="filter" type="text" @keyup=${this.filterHotlists}>
      <i class="material-icons">search</i>
    `;
  }

  /**
   * Renders the user's Hotlists.
   * @return {TemplateResult}
   */
  renderHotlists() {
    return html`
      <div class="user-hotlists">
        ${this.filteredHotlists.length ?
          this.filteredHotlists.map(this.renderFilteredHotlist, this) : ''}
      </div>
    `;
  }

  /**
   * Renders a user's filtered Hotlist.
   * @param {HotlistV0} hotlist The user Hotlist to render.
   * @return {TemplateResult}
   */
  renderFilteredHotlist(hotlist) {
    return html`
      <div
        class="hotlist"
        data-hotlist-name="${hotlist.name}"
      >
        ${hotlist.name}
      </div>`;
  }

  /**
   * Renders dialog error.
   * @return {TemplateResult}
   */
  renderError() {
    return html`
      <br>
      ${this.error ? html`
        <div class="error">${this.error}</div>
      `: ''}
    `;
  }

  /** @override */
  static get properties() {
    return {
      // Populated from Redux.
      userHotlists: {type: Array},
      filteredHotlists: {type: Array},
      issueRefs: {type: Array},
      error: {type: String},
    };
  }

  /** @override */
  stateChanged(state) {
    this.userHotlists = userV0.currentUser(state).hotlists;
  }

  /** @override */
  constructor() {
    super();

    /** @type {Array} */
    this.userHotlists = [];

    /** @type {Array} */
    this.filteredHotlists = this.userHotlists;

    /** @type {Array<IssueRef>} */
    this.issueRefs = [];

    /** @type {string} */
    this.error = '';
  }

  /**
   * Opens the dialog.
   */
  open() {
    this.reset();
    this.shadowRoot.querySelector('chops-dialog').open();
  }

  /**
   * Resets any changes to the form and error.
   */
  reset() {
    this.error = '';
    const filter = this.shadowRoot.querySelector('#filter');
    filter.value = '';
    this.filterHotlists();
  }

  /**
   * Closes the dialog.
   */
  close() {
    this.shadowRoot.querySelector('chops-dialog').close();
  }

  /**
   * Filters the visible Hotlists with the given user input.
   * Requires filter to be an input element with its id as "filter".
   */
  filterHotlists() {
    const input = this.shadowRoot.querySelector('#filter');
    if (!input) {
      // Short circuit because there's no filter.
      this.filteredHotlists = this.userHotlists;
    } else {
      const filter = input.value.toLowerCase();
      const visibleHotlists = [];
      this.userHotlists.forEach((hotlist) => {
        const hotlistName = hotlist.name.toLowerCase();
        if (hotlistName.includes(filter)) {
          visibleHotlists.push(hotlist);
        }
      });
      this.filteredHotlists = visibleHotlists;
    }
  }
}

customElements.define('mr-issue-hotlists-dialog', MrIssueHotlistsDialog);
