// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import {connectStore, store} from 'reducers/base.js';
import * as users from 'reducers/users.js';
import * as issueV0 from 'reducers/issueV0.js';
import {issueRefToString} from 'shared/convertersV0.js';
import {MrStar} from './mr-star.js';


/**
 * `<mr-issue-star>`
 *
 * A button for starring an issue.
 *
 */
export class MrIssueStar extends connectStore(MrStar) {
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

  /** @override */
  get type() {
    return 'issue';
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

  /** @override */
  get canStar() {
    return this._currentUserName && !this._fetchingIsStarred &&
        !this._isStarring;
  }

  /** @override */
  get isStarred() {
    return this._starredIssues.has(issueRefToString(this.issueRef));
  }

  /** @override */
  star() {
    store.dispatch(issueV0.star(this.issueRef, true));
  }

  /** @override */
  unstar() {
    store.dispatch(issueV0.star(this.issueRef, false));
  }
}

customElements.define('mr-issue-star', MrIssueStar);
