// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {html, css} from 'lit-element';

import 'elements/framework/mr-warning/mr-warning.js';
import {hotlists} from 'reducers/hotlists.js';
import {prpcClient} from 'prpc-client-instance.js';
import {MrIssueHotlistsDialog} from './mr-issue-hotlists-dialog';

/**
 * `<mr-move-issue-hotlists-dialog>`
 *
 * Displays a dialog to select the Hotlist to move the provided Issues.
 */
export class MrMoveIssueDialog extends MrIssueHotlistsDialog {
  /** @override */
  static get styles() {
    return [
      super.styles,
      css`
        .hotlist {
          padding: 4px;
        }
        .hotlist:hover {
          background: var(--chops-active-choice-bg);
          cursor: pointer;
        }
      `,
    ];
  }

  /** @override */
  renderHeader() {
    const warningText =
        `Moving issues will remove them from ${this._viewedHotlist ?
          this._viewedHotlist.displayName : 'this hotlist'}.`;
    return html`
      <h3 class="medium-heading">Move issues to hotlist</h3>
      <mr-warning title=${warningText}>${warningText}</mr-warning>
    `;
  }

  /** @override */
  renderFilteredHotlist(hotlist) {
    if (this._viewedHotlist &&
      hotlist.name === this._viewedHotlist.displayName) return;
    return html`
      <div
        class="hotlist"
        data-hotlist-name="${hotlist.name}"
        @click=${this._targetHotlistPicked}>
        ${hotlist.name}
      </div>`;
  }

  /** @override */
  static get properties() {
    return {
      ...MrIssueHotlistsDialog.properties,
      // Populated from Redux.
      _viewedHotlist: {type: Object},
    };
  }

  /** @override */
  stateChanged(state) {
    super.stateChanged(state);
    this._viewedHotlist = hotlists.viewedHotlist(state);
  }

  /** @override */
  constructor() {
    super();

    /**
     * The currently viewed Hotlist.
     * @type {?Hotlist}
     **/
    this._viewedHotlist = null;
  }

  /**
   * Handles picking a Hotlist to move to.
   * @param {Event} e
   */
  async _targetHotlistPicked(e) {
    const targetHotlistName = e.target.dataset.hotlistName;
    const changes = {
      added: [],
      removed: [],
    };

    for (const hotlist of this.userHotlists) {
      // We move from the current Hotlist to the target Hotlist.
      if (changes.added.length === 1 && changes.removed.length === 1) break;
      const change = {
        name: hotlist.name,
        owner: hotlist.ownerRef,
      };
      if (hotlist.name === targetHotlistName) {
        changes.added.push(change);
      } else if (hotlist.name === this._viewedHotlist.displayName) {
        changes.removed.push(change);
      }
    }

    const issueRefs = this.issueRefs;
    if (!issueRefs) return;

    // TODO(https://crbug.com/monorail/7778): Use action creators.
    const promises = [];
    if (changes.added && changes.added.length) {
      promises.push(prpcClient.call(
          'monorail.Features', 'AddIssuesToHotlists', {
            hotlistRefs: changes.added,
            issueRefs,
          },
      ));
    }
    if (changes.removed && changes.removed.length) {
      promises.push(prpcClient.call(
          'monorail.Features', 'RemoveIssuesFromHotlists', {
            hotlistRefs: changes.removed,
            issueRefs,
          },
      ));
    }

    try {
      await Promise.all(promises);
      this.dispatchEvent(new Event('saveSuccess'));
      this.close();
    } catch (error) {
      this.error = error.message || error.description;
    }
  }
}

customElements.define('mr-move-issue-hotlists-dialog', MrMoveIssueDialog);
