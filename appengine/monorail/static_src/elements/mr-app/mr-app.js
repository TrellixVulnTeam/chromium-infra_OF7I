// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html} from 'lit-element';
import {repeat} from 'lit-html/directives/repeat';
import page from 'page';
import qs from 'qs';

import {getServerStatusCron} from 'shared/cron.js';
import 'elements/framework/mr-site-banner/mr-site-banner.js';
import {store, connectStore} from 'reducers/base.js';
import * as projectV0 from 'reducers/projectV0.js';
import {hotlists} from 'reducers/hotlists.js';
import * as issueV0 from 'reducers/issueV0.js';
import * as permissions from 'reducers/permissions.js';
import * as users from 'reducers/users.js';
import * as userv0 from 'reducers/userV0.js';
import * as ui from 'reducers/ui.js';
import * as sitewide from 'reducers/sitewide.js';
import {arrayToEnglish} from 'shared/helpers.js';
import {trackPageChange} from 'shared/ga-helpers.js';
import 'elements/chops/chops-announcement/chops-announcement.js';
import 'elements/issue-list/mr-list-page/mr-list-page.js';
import 'elements/issue-entry/mr-issue-entry-page.js';
import 'elements/framework/mr-header/mr-header.js';
import 'elements/help/mr-cue/mr-cue.js';
import {cueNames} from 'elements/help/mr-cue/cue-helpers.js';
import 'elements/chops/chops-snackbar/chops-snackbar.js';

import {SHARED_STYLES} from 'shared/shared-styles.js';

const QUERY_PARAMS_THAT_RESET_SCROLL = ['q', 'mode', 'id'];
const GOOGLE_EMAIL_SUFFIX = '@google.com';

/**
 * `<mr-app>`
 *
 * The container component for all pages under the Monorail SPA.
 *
 */
export class MrApp extends connectStore(LitElement) {
  /** @override */
  render() {
    if (this.page === 'wizard') {
      return html`<div id="reactMount"></div>`;
    }

    return html`
      <style>
        ${SHARED_STYLES}
        mr-app {
          display: block;
          padding-top: var(--monorail-header-height);
          margin-top: -1px; /* Prevent a double border from showing up. */

          /* From shared-styles.js. */
          --mr-edit-field-padding: 0.125em 4px;
          --mr-edit-field-width: 90%;
          --mr-input-grid-gap: 6px;
          font-family: var(--chops-font-family);
          color: var(--chops-primary-font-color);
          font-size: var(--chops-main-font-size);
        }
        main {
          border-top: var(--chops-normal-border);
        }
        .snackbar-container {
          position: fixed;
          bottom: 1em;
          left: 1em;
          display: flex;
          flex-direction: column;
          align-items: flex-start;
          z-index: 1000;
        }
        /** Unfix <chops-snackbar> to allow stacking. */
        chops-snackbar {
          position: static;
          margin-top: 0.5em;
        }
      </style>
      <mr-header
        .userDisplayName=${this.userDisplayName}
        .loginUrl=${this.loginUrl}
        .logoutUrl=${this.logoutUrl}
      ></mr-header>
      <chops-announcement service="monorail"></chops-announcement>
      <mr-site-banner></mr-site-banner>
      <mr-cue
        cuePrefName=${cueNames.SWITCH_TO_PARENT_ACCOUNT}
        .loginUrl=${this.loginUrl}
        centered
        nondismissible
      ></mr-cue>
      <mr-cue
        cuePrefName=${cueNames.SEARCH_FOR_NUMBERS}
        centered
      ></mr-cue>
      <main>${this._renderPage()}</main>
      <div class="snackbar-container" aria-live="polite">
        ${repeat(this._snackbars, (snackbar) => html`
          <chops-snackbar
            @close=${this._closeSnackbar.bind(this, snackbar.id)}
          >${snackbar.text}</chops-snackbar>
        `)}
      </div>
    `;
  }

  /**
   * @param {string} id The name of the snackbar to close.
   */
  _closeSnackbar(id) {
    store.dispatch(ui.hideSnackbar(id));
  }

  /**
   * Helper for determiing which page component to render.
   * @return {TemplateResult}
   */
  _renderPage() {
    switch (this.page) {
      case 'detail':
        return html`
          <mr-issue-page
            .userDisplayName=${this.userDisplayName}
            .loginUrl=${this.loginUrl}
          ></mr-issue-page>
        `;
      case 'entry':
        return html`
          <mr-issue-entry-page
            .userDisplayName=${this.userDisplayName}
            .loginUrl=${this.loginUrl}
          ></mr-issue-entry-page>
        `;
      case 'grid':
        return html`
          <mr-grid-page
            .userDisplayName=${this.userDisplayName}
          ></mr-grid-page>
        `;
      case 'list':
        return html`
          <mr-list-page
            .userDisplayName=${this.userDisplayName}
          ></mr-list-page>
        `;
      case 'chart':
        return html`<mr-chart-page></mr-chart-page>`;
      case 'projects':
        return html`<mr-projects-page></mr-projects-page>`;
      case 'hotlist-issues':
        return html`<mr-hotlist-issues-page></mr-hotlist-issues-page>`;
      case 'hotlist-people':
        return html`<mr-hotlist-people-page></mr-hotlist-people-page>`;
      case 'hotlist-settings':
        return html`<mr-hotlist-settings-page></mr-hotlist-settings-page>`;
      default:
        return;
    }
  }

  /** @override */
  static get properties() {
    return {
      /**
       * Backend-generated URL for the page the user is directed to for login.
       */
      loginUrl: {type: String},
      /**
       * Backend-generated URL for the page the user is directed to for logout.
       */
      logoutUrl: {type: String},
      /**
       * The display name of the currently logged in user.
       */
      userDisplayName: {type: String},
      /**
       * The search parameters in the user's current URL.
       */
      queryParams: {type: Object},
      /**
       * A list of forms to check for "dirty" values when the user navigates
       * across pages.
       */
      dirtyForms: {type: Array},
      /**
       * App Engine ID for the current version being viewed.
       */
      versionBase: {type: String},
      /**
       * A String identifier for the page that the user is viewing.
       */
      page: {type: String},
      /**
       * A String for the title of the page that the user will see in their
       * browser tab. ie: equivalent to the <title> tag.
       */
      pageTitle: {type: String},
      /**
       * Array of snackbar objects to render.
       */
      _snackbars: {type: Array},
    };
  }

  /** @override */
  constructor() {
    super();
    this.queryParams = {};
    this.dirtyForms = [];
    this.userDisplayName = '';

    /**
     * @type {PageJS.Context}
     * The context of the page. This should not be a LitElement property
     * because we don't want to re-render when updating this.
     */
    this._lastContext = undefined;
  }

  /** @override */
  createRenderRoot() {
    return this;
  }

  /** @override */
  stateChanged(state) {
    this.dirtyForms = ui.dirtyForms(state);
    this.queryParams = sitewide.queryParams(state);
    this.pageTitle = sitewide.pageTitle(state);
    this._snackbars = ui.snackbars(state);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('userDisplayName') && this.userDisplayName) {
      // TODO(https://crbug.com/monorail/7238): Migrate userv0 calls to v3 API.
      store.dispatch(userv0.fetch(this.userDisplayName));

      // Typically we would prefer 'users/<userId>' instead.
      store.dispatch(users.fetch(`users/${this.userDisplayName}`));
    }

    if (changedProperties.has('pageTitle')) {
      // To ensure that changes to the page title are easy to reason about,
      // we want to sync the current pageTitle in the Redux state to
      // document.title in only one place in the code.
      document.title = this.pageTitle;
    }
    if (changedProperties.has('page')) {
      trackPageChange(this.page, this.userDisplayName);
    }
  }

  /** @override */
  connectedCallback() {
    super.connectedCallback();

    this._logGooglerUsage();

    // TODO(zhangtiff): Figure out some way to save Redux state between
    // page loads.

    // page doesn't handle users reloading the page or closing a tab.
    window.onbeforeunload = this._confirmDiscardMessage.bind(this);

    // Start a cron task to periodically request the status from the server.
    getServerStatusCron.start();

    const postRouteHandler = this._postRouteHandler.bind(this);

    // Populate the project route parameter before _preRouteHandler runs.
    page('/p/:project/*', (_ctx, next) => next());
    page('*', this._preRouteHandler.bind(this));

    page('/hotlists/:hotlist', (ctx) => {
      page.redirect(`/hotlists/${ctx.params.hotlist}/issues`);
    });
    page('/hotlists/:hotlist/*', this._selectHotlist);
    page('/hotlists/:hotlist/issues',
        this._loadHotlistIssuesPage.bind(this), postRouteHandler);
    page('/hotlists/:hotlist/people',
        this._loadHotlistPeoplePage.bind(this), postRouteHandler);
    page('/hotlists/:hotlist/settings',
        this._loadHotlistSettingsPage.bind(this), postRouteHandler);

    // Handle Monorail's landing page.
    page('/p', '/');
    page('/projects', '/');
    page('/hosting', '/');
    page('/', this._loadProjectsPage.bind(this), postRouteHandler);

    page('/p/:project/issues/list', this._loadListPage.bind(this),
        postRouteHandler);
    page('/p/:project/issues/detail', this._loadIssuePage.bind(this),
        postRouteHandler);
    page('/p/:project/issues/entry_new', this._loadEntryPage.bind(this),
        postRouteHandler);
    page('/p/:project/issues/wizard', this._loadWizardPage.bind(this),
        postRouteHandler);

    // Redirects from old hotlist pages to SPA hotlist pages.
    const hotlistRedirect = (pageName) => async (ctx) => {
      const name =
          await hotlists.getHotlistName(ctx.params.user, ctx.params.hotlist);
      page.redirect(`/${name}/${pageName}`);
    };
    page('/users/:user/hotlists/:hotlist', hotlistRedirect('issues'));
    page('/users/:user/hotlists/:hotlist/people', hotlistRedirect('people'));
    page('/users/:user/hotlists/:hotlist/details', hotlistRedirect('settings'));

    page();
  }

  /**
   * Helper to log how often Googlers access Monorail.
   */
  _logGooglerUsage() {
    const email = this.userDisplayName;
    if (!email) return;
    if (!email.endsWith(GOOGLE_EMAIL_SUFFIX)) return;

    const username = email.replace(GOOGLE_EMAIL_SUFFIX, '');

    // Context: b/229758140
    window.fetch(`https://buganizer.corp.google.com/action/yes?monorail=yes&username=${username}`);
  }

  /**
   * Handler that runs on every single route change, before the new page has
   * loaded. This function should not use store.dispatch() or assign properties
   * on this because running these actions causes extra re-renders to happen.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  _preRouteHandler(ctx, next) {
    // We're not really navigating anywhere, so don't do anything.
    if (this._lastContext && this._lastContext.path &&
      ctx.path === this._lastContext.path) {
      Object.assign(ctx, this._lastContext);
      // Set ctx.handled to false, so we don't push the state to browser's
      // history.
      ctx.handled = false;
      return;
    }

    // Check if there were forms with unsaved data before loading the next
    // page.
    const discardMessage = this._confirmDiscardMessage();
    if (discardMessage && !confirm(discardMessage)) {
      Object.assign(ctx, this._lastContext);
      // Set ctx.handled to false, so we don't push the state to browser's
      // history.
      ctx.handled = false;
      // We don't call next to avoid loading whatever page was supposed to
      // load next.
      return;
    }

    // Run query string parsing on all routes. Query params must be parsed
    // before routes are loaded because some routes use them to conditionally
    // load bundles.
    // Based on: https://visionmedia.github.io/page.js/#plugins
    const params = qs.parse(ctx.querystring);

    // Make sure queryParams are not case sensitive.
    const lowerCaseParams = {};
    Object.keys(params).forEach((key) => {
      lowerCaseParams[key.toLowerCase()] = params[key];
    });
    ctx.queryParams = lowerCaseParams;

    this._selectProject(ctx.params.project);

    next();
  }

  /**
   * Handler that runs on every single route change, after the new page has
   * loaded.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  _postRouteHandler(ctx, next) {
    // Scroll to the requested element if a hash is present.
    if (ctx.hash) {
      store.dispatch(ui.setFocusId(ctx.hash));
    }

    // Sync queryParams to Redux after the route has loaded, rather than before,
    // to avoid having extra queryParams update on the previously loaded
    // component.
    store.dispatch(sitewide.setQueryParams(ctx.queryParams));

    // Increment the count of navigations in the Redux store.
    store.dispatch(ui.incrementNavigationCount());

    // Clear dirty forms when entering a new page.
    store.dispatch(ui.clearDirtyForms());


    if (!this._lastContext || this._lastContext.pathname !== ctx.pathname ||
        this._hasReleventParamChanges(ctx.queryParams,
            this._lastContext.queryParams)) {
      // Reset the scroll position after a new page has rendered.
      window.scrollTo(0, 0);
    }

    // Save the context of this page to be compared to later.
    this._lastContext = ctx;
  }

  /**
   * Finds if a route change changed query params in a way that should cause
   * scrolling to reset.
   * @param {Object} currentParams
   * @param {Object} oldParams
   * @param {Array<string>=} paramsToCompare Which params to check.
   * @return {boolean} Whether any of the relevant query params changed.
   */
  _hasReleventParamChanges(currentParams, oldParams,
      paramsToCompare = QUERY_PARAMS_THAT_RESET_SCROLL) {
    return paramsToCompare.some((paramName) => {
      return currentParams[paramName] !== oldParams[paramName];
    });
  }

  /**
   * Helper to manage syncing project route state to Redux.
   * @param {string=} project displayName for a referenced project.
   *   Defaults to null for consistency with Redux.
   */
  _selectProject(project = null) {
    if (projectV0.viewedProjectName(store.getState()) !== project) {
      // Note: We want to update the project even if the new project
      // is null.
      store.dispatch(projectV0.select(project));
      if (project) {
        store.dispatch(projectV0.fetch(project));
      }
    }
  }

  /**
   * Loads and triggers rendering for the list of all projects.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadProjectsPage(ctx, next) {
    await import(/* webpackChunkName: "mr-projects-page" */
        '../projects/mr-projects-page/mr-projects-page.js');
    this.page = 'projects';
    next();
  }

  /**
   * Loads and triggers render for the issue detail page.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadIssuePage(ctx, next) {
    performance.clearMarks('start load issue detail page');
    performance.mark('start load issue detail page');

    await import(/* webpackChunkName: "mr-issue-page" */
        '../issue-detail/mr-issue-page/mr-issue-page.js');

    const issueRef = {
      localId: Number.parseInt(ctx.queryParams.id),
      projectName: ctx.params.project,
    };
    store.dispatch(issueV0.viewIssue(issueRef));
    store.dispatch(issueV0.fetchIssuePageData(issueRef));
    this.page = 'detail';
    next();
  }

  /**
   * Loads and triggers render for the issue list page, including the list,
   * grid, and chart modes.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadListPage(ctx, next) {
    performance.clearMarks('start load issue list page');
    performance.mark('start load issue list page');
    switch (ctx.queryParams && ctx.queryParams.mode &&
        ctx.queryParams.mode.toLowerCase()) {
      case 'grid':
        await import(/* webpackChunkName: "mr-grid-page" */
            '../issue-list/mr-grid-page/mr-grid-page.js');
        this.page = 'grid';
        break;
      case 'chart':
        await import(/* webpackChunkName: "mr-chart-page" */
            '../issue-list/mr-chart-page/mr-chart-page.js');
        this.page = 'chart';
        break;
      default:
        this.page = 'list';
        break;
    }
    next();
  }

  /**
   * Load the issue entry page
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  _loadEntryPage(ctx, next) {
    this.page = 'entry';
    next();
  }

  /**
   * Load the issue wizard
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadWizardPage(ctx, next) {
    const {renderWizard} = await import(
        /* webpackChunkName: "IssueWizard" */ '../../react/IssueWizard.tsx');

    this.page = 'wizard';
    next();

    await this.updateComplete;

    const mount = document.getElementById('reactMount');

    renderWizard(mount, this.loginUrl, this.userDisplayName);
  }

  /**
   * Gets the currently viewed HotlistRef from the URL, selects
   * it in the Redux store, and fetches the Hotlist data.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  _selectHotlist(ctx, next) {
    const name = 'hotlists/' + ctx.params.hotlist;
    store.dispatch(hotlists.select(name));
    store.dispatch(hotlists.fetch(name));
    store.dispatch(hotlists.fetchItems(name));
    store.dispatch(permissions.batchGet([name]));
    next();
  }

  /**
   * Loads mr-hotlist-issues-page.js and makes it the currently viewed page.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadHotlistIssuesPage(ctx, next) {
    await import(/* webpackChunkName: "mr-hotlist-issues-page" */
        `../hotlist/mr-hotlist-issues-page/mr-hotlist-issues-page.js`);
    this.page = 'hotlist-issues';
    next();
  }

  /**
   * Loads mr-hotlist-people-page.js and makes it the currently viewed page.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadHotlistPeoplePage(ctx, next) {
    await import(/* webpackChunkName: "mr-hotlist-people-page" */
        `../hotlist/mr-hotlist-people-page/mr-hotlist-people-page.js`);
    this.page = 'hotlist-people';
    next();
  }

  /**
   * Loads mr-hotlist-settings-page.js and makes it the currently viewed page.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadHotlistSettingsPage(ctx, next) {
    await import(/* webpackChunkName: "mr-hotlist-settings-page" */
        `../hotlist/mr-hotlist-settings-page/mr-hotlist-settings-page.js`);
    this.page = 'hotlist-settings';
    next();
  }

  /**
   * Constructs a message to warn users about dirty forms when they navigate
   * away from a page, to prevent them from loasing data.
   * @return {string} Message shown to users to warn about in flight form
   *   changes.
   */
  _confirmDiscardMessage() {
    if (!this.dirtyForms.length) return null;
    const dirtyFormsMessage =
      'Discard your changes in the following forms?\n' +
      arrayToEnglish(this.dirtyForms);
    return dirtyFormsMessage;
  }
}

customElements.define('mr-app', MrApp);
