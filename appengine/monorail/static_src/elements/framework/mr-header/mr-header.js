// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {connectStore} from 'reducers/base.js';
import * as user from 'reducers/user.js';
import * as project from 'reducers/project.js';
import * as sitewide from 'reducers/sitewide.js';

import '../mr-dropdown/mr-dropdown.js';
import '../mr-dropdown/mr-account-dropdown.js';
import './mr-search-bar.js';

import {SHARED_STYLES} from 'shared/shared-styles.js';

import ClientLogger from 'monitoring/client-logger.js';


/**
 * `<mr-header>`
 *
 * The header for Monorail.
 *
 */
export class MrHeader extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          color: var(--chops-header-text-color);
          box-sizing: border-box;
          background: hsl(221, 67%, 92%);
          width: 100%;
          height: var(--monorail-header-height);
          display: flex;
          flex-direction: row;
          justify-content: flex-start;
          align-items: center;
          z-index: 800;
          background-color: var(--chops-primary-header-bg);
          border-bottom: var(--chops-normal-border);
          top: 0;
          position: fixed;
          padding: 0 4px;
          font-size: var(--chops-large-font-size);
        }
        @media (max-width: 840px) {
          :host {
            position: static;
          }
        }
        a {
          font-size: inherit;
          color: var(--chops-link-color);
          text-decoration: none;
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100%;
          padding: 0 4px;
          flex-grow: 0;
          flex-shrink: 0;
        }
        a[hidden] {
          display: none;
        }
        a.button {
          font-size: inherit;
          height: auto;
          margin: 0 8px;
          border: 0;
          height: 30px;
        }
        a.home-link:hover {
          text-decoration: none;
        }
        mr-search-bar {
          margin-left: 8px;
          flex-grow: 2;
          max-width: 1000px;
        }
        i.material-icons {
          font-size: var(--chops-icon-font-size);
          color: var(--chops-primary-icon-color);
        }
        i.material-icons[hidden] {
          display: none;
        }
        .right-section {
          font-size: inherit;
          display: flex;
          align-items: center;
          height: 100%;
          margin-left: auto;
          justify-content: flex-end;
        }
      `,
    ];
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <a
        href=${this.projectName ? `/p/${this.projectName}/` : '/'}
        class="home-link"
      >
        ${this.projectThumbnailUrl ? html`
          <img
            class="project-logo"
            src=${this.projectThumbnailUrl}
            title="Issues"
          />
        ` : ''}
        <i
          class="material-icons"
          title="Issues"
          ?hidden=${this.projectThumbnailUrl}
        >home</i>
      </a>
      <mr-dropdown
        class="project-selector"
        .text=${this.projectName || 'Select a project'}
        .items=${this._projectDropdownItems}
        menuAlignment="left"
        title=${this.presentationConfig.projectSummary}
      ></mr-dropdown>
      <a
        ?hidden=${!this.projectName}
        class="button emphasized new-issue-link"
        href=${this.issueEntryUrl}
      >
        New issue
      </a>
      <mr-search-bar
        .projectName=${this.projectName}
        .userDisplayName=${this.userDisplayName}
        .projectSavedQueries=${this.presentationConfig.savedQueries}
        .initialCan=${this._currentCan}
        .initialQuery=${this._currentQuery}
        .queryParams=${this.queryParams}
        ?hidden=${!this.projectName}
      ></mr-search-bar>

      <div class="right-section">
        <mr-dropdown
          icon="settings"
          label="Project Settings"
          .items=${this._projectSettingsItems}
          ?hidden=${!this.projectName}
        ></mr-dropdown>

        ${this.userDisplayName ? html`
          <mr-account-dropdown
            .userDisplayName=${this.userDisplayName}
            .logoutUrl=${this.logoutUrl}
            .loginUrl=${this.loginUrl}
          ></mr-account-dropdown>
        `: ''}
        <a
          href=${this.loginUrl}
          ?hidden=${this.userDisplayName}
        >Sign in</a>
      </div>
    `;
  }

  /** @override */
  static get properties() {
    return {
      loginUrl: {type: String},
      logoutUrl: {type: String},
      projectName: {type: String},
      // Project thumbnail is set separately from presentationConfig to prevent
      // "flashing" logo when navigating EZT pages.
      projectThumbnailUrl: {type: String},
      userDisplayName: {type: String},
      isSiteAdmin: {type: Boolean},
      userProjects: {type: Object},
      presentationConfig: {type: Object},
      queryParams: {type: Object},
      // TODO(zhangtiff): Change this to be dynamically computed by the
      //   frontend with logic similar to ComputeIssueEntryURL().
      issueEntryUrl: {type: String},
      clientLogger: {type: Object},
      _currentQuery: {type: String},
      _currentCan: {type: String},
    };
  }

  /** @override */
  constructor() {
    super();

    this.presentationConfig = {};
    this.userProjects = {};
    this.isSiteAdmin = false;

    this.clientLogger = new ClientLogger('mr-header');
  }

  /** @override */
  stateChanged(state) {
    this.projectName = project.viewedProjectName(state);

    this.userProjects = user.projects(state);

    const currentUser = user.user(state);
    this.isSiteAdmin = currentUser ? currentUser.isSiteAdmin : false;

    const presentationConfig = project.viewedPresentationConfig(state);
    this.presentationConfig = presentationConfig;
    // Set separately in order allow EZT pages to load project logo before
    // the GetPresentationConfig pRPC request.
    this.projectThumbnailUrl = presentationConfig.projectThumbnailUrl;

    this._currentQuery = sitewide.currentQuery(state);
    this._currentCan = sitewide.currentCan(state);

    this.queryParams = sitewide.queryParams(state);
  }

  /**
   * @return {boolean} whether the currently logged in user has admin
   *   privileges for the currently viewed project.
   */
  get canAdministerProject() {
    if (!this.userDisplayName) return false; // Not logged in.
    if (this.isSiteAdmin) return true;
    if (!this.userProjects || !this.userProjects.ownerOf) return false;
    return this.userProjects.ownerOf.includes(this.projectName);
  }

  /**
   * @return {Array<MenuItem>} the dropdown items for the project selector,
   *   showing which projects a user can switch to.
   */
  get _projectDropdownItems() {
    const {userProjects, loginUrl} = this;
    if (!this.userDisplayName) {
      return [{text: 'Sign in to see your projects', url: loginUrl}];
    }

    const items = [];
    const starredProjects = userProjects.starredProjects || [];
    const projects = (userProjects.ownerOf || [])
        .concat(userProjects.memberOf || [])
        .concat(userProjects.contributorTo || []);

    if (projects.length) {
      projects.sort();
      items.push({text: 'My Projects', separator: true});

      projects.forEach((project) => {
        items.push({text: project, url: `/p/${project}/`});
      });
    }

    if (starredProjects.length) {
      starredProjects.sort();
      items.push({text: 'Starred Projects', separator: true});

      starredProjects.forEach((project) => {
        items.push({text: project, url: `/p/${project}/`});
      });
    }

    if (items.length) {
      items.push({separator: true});
    }

    items.push({text: 'All projects', url: '/hosting/'});
    items.forEach((item) => {
      item.handler = () => this._projectChangedHandler(item.url);
    });
    return items;
  }

  /**
   * @return {Array<MenuItem>} dropdown menu items to show in the project
   *   settings menu.
   */
  get _projectSettingsItems() {
    const {projectName, canAdministerProject} = this;
    const items = [
      {text: 'People', url: `/p/${projectName}/people/list`},
      {text: 'Development Process', url: `/p/${projectName}/adminIntro`},
      {text: 'History', url: `/p/${projectName}/updates/list`},
    ];

    if (canAdministerProject) {
      items.push({separator: true});
      items.push({text: 'Administer', url: `/p/${projectName}/admin`});
    }
    return items;
  }

  /**
   * Records Google Analytics events for when users change projects using
   * the selector.
   * @param {string} url which project URL the user is navigating to.
   */
  _projectChangedHandler(url) {
    // Just log it to GA and continue.
    this.clientLogger.logEvent('project-change', url);
  }
}

customElements.define('mr-header', MrHeader);
