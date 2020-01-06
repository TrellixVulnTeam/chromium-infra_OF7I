// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import qs from 'qs';
import {store, connectStore} from 'reducers/base.js';
import * as user from 'reducers/user.js';
import * as issue from 'reducers/issue.js';
import * as project from 'reducers/project.js';
import 'elements/chops/chops-button/chops-button.js';
import 'elements/chops/chops-dialog/chops-dialog.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';
import {cueNames} from './cue-helpers.js';


/**
 * `<mr-cue>`
 *
 * An element that displays one of a set of predefined help messages
 * iff that message is appropriate to the current user and page.
 *
 * TODO: Factor this class out into a base view component and separate
 * usage-specific components, such as those for user prefs.
 *
 */
export class MrCue extends connectStore(LitElement) {
  /** @override */
  constructor() {
    super();
    this.prefs = new Map();
    this.issue = null;
    this.referencedUsers = new Map();
    this.nondismissible = false;
    this.cuePrefName = '';
    this.loginUrl = '';
    this.hidden = this._shouldBeHidden(this.signedIn, this.prefsLoaded,
        this.cuePrefName, this.message);
  }

  /** @override */
  static get properties() {
    return {
      issue: {type: Object},
      referencedUsers: {type: Object},
      user: {type: Object},
      cuePrefName: {type: String},
      nondismissible: {type: Boolean},
      prefs: {type: Object},
      prefsLoaded: {type: Boolean},
      jumpLocalId: {type: Number},
      loginUrl: {type: String},
      hidden: {
        type: Boolean,
        reflect: true,
      },
    };
  }

  /** @override */
  static get styles() {
    return [SHARED_STYLES, css`
      :host {
        display: block;
        margin: 2px 0;
        padding: 2px 4px 2px 8px;
        background: var(--chops-notice-bubble-bg);
        border: var(--chops-notice-border);
        text-align: center;
      }
      :host([centered]) {
        display: flex;
        justify-content: center;
      }
      :host([hidden]) {
        display: none;
      }
      button[hidden] {
        visibility: hidden;
      }
      i.material-icons {
        font-size: 14px;
      }
      button {
        background: none;
        border: none;
        float: right;
        padding: 2px;
        cursor: pointer;
        border-radius: 50%;
        display: inline-flex;
        align-items: center;
        justify-content: center;
      }
      button:hover {
        background: rgba(0, 0, 0, .2);
      }
    `];
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <button
        @click=${this.dismiss}
        title="Don't show this message again."
        ?hidden=${this.nondismissible}>
        <i class="material-icons">close</i>
      </button>
      <div id="message">${this.message}</div>
    `;
  }

  /**
   * @return {TemplateResult} lit-html template for the cue message a user
   * should see.
   */
  get message() {
    if (this.cuePrefName === cueNames.CODE_OF_CONDUCT) {
      return html`
        Please keep discussions respectful and constructive.
        See our
        <a href="${this.codeOfConductUrl}"
           target="_blank">code of conduct</a>.
        `;
    } else if (this.cuePrefName === cueNames.AVAILABILITY_MSGS) {
      if (this._availablityMsgsRelevant(this.issue)) {
        return html`
          <b>Note:</b>
          Clock icons indicate that users may not be available.
          Tooltips show the reason.
          `;
      }
    } else if (this.cuePrefName === cueNames.SWITCH_TO_PARENT_ACCOUNT) {
      if (this._switchToParentAccountRelevant()) {
        return html`
          You are signed in to a linked account.
          <a href="${this.loginUrl}">
             Switch to ${this.user.linkedParentRef.displayName}</a>.
          `;
      }
    } else if (this.cuePrefName === cueNames.SEARCH_FOR_NUMBERS) {
      if (this._searchForNumbersRelevant(this.jumpLocalId)) {
        return html`
          <b>Tip:</b>
          To find issues containing "${this.jumpLocalId}", use quotes.
          `;
      }
    }
    return;
  }

  /**
  * Conditionally returns a hardcoded code of conduct URL for
  * different projects.
  * @return {string} the URL for the code of conduct.
   */
  get codeOfConductUrl() {
    // TODO(jrobbins): Store this in the DB and pass it via the API.
    if (this.projectName === 'fuchsia') {
      return 'https://fuchsia.dev/fuchsia-src/CODE_OF_CONDUCT';
    }
    return ('https://chromium.googlesource.com/' +
            'chromium/src/+/master/CODE_OF_CONDUCT.md');
  }

  /** @override */
  updated(changedProperties) {
    const hiddenWatchProps = ['prefsLoaded', 'cuePrefName', 'signedIn',
      'prefs'];
    const shouldUpdateHidden = Array.from(changedProperties.keys())
        .some((propName) => hiddenWatchProps.includes(propName));
    if (shouldUpdateHidden) {
      this.hidden = this._shouldBeHidden(this.signedIn, this.prefsLoaded,
          this.cuePrefName, this.message);
    }
  }

  /**
   * Checks if there are any unavailable users and only displays this cue if so.
   * @param {Issue} issue
   * @return {boolean} Whether the User Availability cue should be
   *   displayed or not.
   */
  _availablityMsgsRelevant(issue) {
    if (!issue) return false;
    return (this._anyUnvailable([issue.ownerRef]) ||
            this._anyUnvailable(issue.ccRefs));
  }

  /**
   * Checks if a given list of users contains any unavailable users.
   * @param {Array<UserRef>} userRefList
   * @return {boolean} Whether there are unavailable users.
   */
  _anyUnvailable(userRefList) {
    if (!userRefList) return false;
    for (const userRef of userRefList) {
      if (userRef) {
        const participant = this.referencedUsers.get(userRef.displayName);
        if (participant && participant.availability) return true;
      }
    }
  }

  /**
   * Finds if the user has a linked parent account that's separate from the
   * one they are logged into and conditionally hides the cue if so.
   * @return {boolean} Whether to show the cue to switch to a parent account.
   */
  _switchToParentAccountRelevant() {
    return this.user && this.user.linkedParentRef;
  }

  /**
   * Determines whether the user should see a cue telling them how to avoid the
   * "jump to issue" feature.
   * @param {number} jumpLocalId the ID of the issue the user jumped to.
   * @return {boolean} Whether the user jumped to a number or not.
   */
  _searchForNumbersRelevant(jumpLocalId) {
    return !!jumpLocalId;
  }

  /**
   * Checks the user's preferences to hide a particular cue if they have
   * dismissed it.
   * @param {boolean} signedIn Whether the user is signed in.
   * @param {boolean} prefsLoaded Whether the user's prefs have been fetched
   *   from the API.
   * @param {string} cuePrefName The name of the cue being checked.
   * @param {string} message
   * @return {boolean} Whether the cue should be hidden.
   */
  _shouldBeHidden(signedIn, prefsLoaded, cuePrefName, message) {
    if (signedIn && !prefsLoaded) return true;
    if (this.alreadyDismissed(cuePrefName)) return true;
    return !message;
  }

  /** @override */
  stateChanged(state) {
    this.projectName = project.viewedProjectName(state);
    this.issue = issue.viewedIssue(state);
    this.referencedUsers = issue.referencedUsers(state);
    this.user = user.user(state);
    this.prefs = user.prefs(state);
    this.signedIn = this.user && this.user.userId;
    this.prefsLoaded = user.user(state).prefsLoaded;

    const queryString = window.location.search.substring(1);
    const queryParams = qs.parse(queryString);
    const q = queryParams.q;
    if (q && q.match(new RegExp('^\\d+$'))) {
      this.jumpLocalId = Number(q);
    }
  }

  /**
   * Check whether a cue has already been dismissed in a user's
   * preferences.
   * @param {string} pref The name of the user preference to check.
   * @return {boolean} Whether the cue was dismissed or not.
   */
  alreadyDismissed(pref) {
    return this.prefs && this.prefs.get(pref) === 'true';
  }

  /**
   * Sends a request to the API to save that a user has dismissed a cue.
   * The results of this request update Redux's state, which leads to
   * the cue disappearing for the user after the request finishes.
   * @return {void}
   */
  dismiss() {
    const newPrefs = [{name: this.cuePrefName, value: 'true'}];
    store.dispatch(user.setPrefs(newPrefs, this.signedIn));
  }
}

customElements.define('mr-cue', MrCue);
