// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {connectStore, store} from 'reducers/base.js';
import * as users from 'reducers/users.js';
import * as issueV0 from 'reducers/issueV0.js';
import {issueRefToString} from 'shared/convertersV0.js';

/**
 * `<mr-star>`
 *
 * A button for starring an issue.
 *
 */
export class MrStar extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
        --mr-star-size: var(--chops-icon-font-size);
      }
      button {
        background: none;
        border: none;
        cursor: pointer;
        padding: 0;
        margin: 0;
        display: flex;
        align-items: center;
      }
      button[disabled] {
        opacity: 0.5;
        cursor: default;
      }
      i.material-icons {
        font-size: var(--mr-star-size);
        color: var(--chops-primary-icon-color);
      }
      i.material-icons.starred {
        color: var(--chops-primary-accent-color);
      }
    `;
  }

  /** @override */
  render() {
    const isStarred = this._starredIssues.has(issueRefToString(this.issueRef));
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <button class="star-button"
        @click=${this.toggleStar}
        ?disabled=${!this._canStar}
        title=${this._renderStarToolTip(isStarred, !!this._currentUserName)}
        aria-checked=${isStarred ? 'true' : 'false'}
      >
        ${isStarred ? html`
          <i class="material-icons starred" role="presentation">
            star
          </i>
        `: html`
          <i class="material-icons" role="presentation">
            star_border
          </i>
        `}
      </button>
    `;
  }

  /**
   *
   * @param {boolean} isStarred Whether the issue is starred.
   * @param {boolean} isLoggedIn Whether the current user is logged in.
   * @return {string} the title to display on the star button.
   */
  _renderStarToolTip(isStarred, isLoggedIn) {
    if (isStarred) {
      return `Unstar this issue`;
    }

    return `${isLoggedIn ? 'Click' : 'Log in'} to star this issue`;
  }

  /** @override */
  static get properties() {
    return {
      /**
       * A reference to the issue that the star button interacts with.
       */
      issueRef: {type: Object},
      /**
       * Whether the issue is starred (used for accessing easily).
       */
      _starredIssues: {type: Set},
      /**
       * Whether the issue's star state is being fetched. This is taken from
       * the component's parent, which is expected to handle fetching initial
       * star state for an issue.
       */
      _fetchingIsStarred: {type: Boolean},
      /**
       * A Map of all issues currently being starred.
       */
      _starringIssues: {type: Object},
      /**
       * The currently logged in user. Required to determine if the user can
       * star.
       */
      _currentUserName: {type: String},
    };
  }

  /** @override */
  stateChanged(state) {
    this._currentUserName = users.currentUserName(state);

    // TODO(crbug.com/monorail/7374): Remove references to issueV0 in
    // <mr-star>.
    this._starringIssues = issueV0.starringIssues(state);
    this._starredIssues = issueV0.starredIssues(state);
    this._fetchingIsStarred = issueV0.requests(state).fetchIsStarred.requesting;
  }

  /**
   * @return {boolean} Whether there's an in-flight star request.
   */
  get _isStarring() {
    const requestKey = issueRefToString(this.issueRef);
    if (this._starringIssues.has(requestKey)) {
      return this._starringIssues.get(requestKey).requesting;
    }
    return false;
  }

  /**
   * @return {boolean} Whether the user is able to star the current object.
   */
  get _canStar() {
    return this._currentUserName && !this._fetchingIsStarred &&
        !this._isStarring;
  }

  /**
   * Stars or unstars the resource based on the user's interaction.
   */
  toggleStar() {
    if (!this._canStar) return;
    const newIsStarred = !this._starredIssues.has(
        issueRefToString(this.issueRef));
    // This component assumes that the user of this component is connected to
    // Redux and will update their star state based on this.
    store.dispatch(issueV0.star(this.issueRef, newIsStarred));
  }
}

customElements.define('mr-star', MrStar);
