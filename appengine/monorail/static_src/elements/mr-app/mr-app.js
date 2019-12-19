// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import page from 'page';
import qs from 'qs';

import {getServerStatusCron} from 'shared/cron.js';
import 'elements/framework/mr-site-banner/mr-site-banner.js';
import {store, connectStore} from 'reducers/base.js';
import * as project from 'reducers/project.js';
import * as hotlist from 'reducers/hotlist.js';
import * as issue from 'reducers/issue.js';
import * as user from 'reducers/user.js';
import * as ui from 'reducers/ui.js';
import * as sitewide from 'reducers/sitewide.js';
import {userIdOrDisplayNameToUserRef} from 'shared/converters.js';
import {arrayToEnglish} from 'shared/helpers.js';
import {trackPageChange} from 'shared/ga-helpers.js';
import 'elements/framework/mr-header/mr-header.js';
import 'elements/framework/mr-keystrokes/mr-keystrokes.js';
import 'elements/help/mr-cue/mr-cue.js';
import {cueNames} from 'elements/help/mr-cue/cue-helpers.js';

import {SHARED_STYLES} from 'shared/shared-styles.js';

/**
 * `<mr-app>`
 *
 * The container component for all pages under the Monorail SPA.
 *
 */
export class MrApp extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          display: block;
          padding-top: var(--monorail-header-height);
          margin-top: -1px; /* Prevent a double border from showing up. */
        }
        main {
          border-top: var(--chops-normal-border);
        }
      `,
    ];
  }

  /** @override */
  render() {
    return html`
      <mr-keystrokes
        .issueId=${this.queryParams.id}
        .queryParams=${this.queryParams}
        .issueEntryUrl=${this.issueEntryUrl}
      ></mr-keystrokes>
      <mr-header
        .userDisplayName=${this.userDisplayName}
        .issueEntryUrl=${this.issueEntryUrl}
        .loginUrl=${this.loginUrl}
        .logoutUrl=${this.logoutUrl}
      ></mr-header>
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
    `;
  }

  /**
   * Helper for determiing which page component to render.
   * @return {TemplateResult}
   */
  _renderPage() {
    if (this.page === 'detail') {
      return html`
        <mr-issue-page
          .userDisplayName=${this.userDisplayName}
          .loginUrl=${this.loginUrl}
        ></mr-issue-page>
      `;
    } else if (this.page === 'grid') {
      return html`
        <mr-grid-page
          .userDisplayName=${this.userDisplayName}
        ></mr-grid-page>
      `;
    } else if (this.page === 'list') {
      return html`
        <mr-list-page
          .userDisplayName=${this.userDisplayName}
        ></mr-list-page>
      `;
    } else if (this.page === 'chart') {
      return html`<mr-chart-page></mr-chart-page>`;
    } else if (this.page === 'hotlist-details') {
      return html`<mr-hotlist-details-page></mr-hotlist-details-page>`;
    } else if (this.page === 'hotlist-issues') {
      return html`<mr-hotlist-issues-page></mr-hotlist-issues-page>`;
    } else if (this.page === 'hotlist-people') {
      return html`<mr-hotlist-people-page></mr-hotlist-people-page>`;
    }
  }

  /** @override */
  static get properties() {
    return {
      /**
       * Backend-generated URL for the page the user is redirected to
       * for filing issues. This functionality is a bit complicated by the
       * issue wizard which redirects non-project members to an
       * authentiation flow for a separate App Engine app for the chromium
       * project.
       */
      issueEntryUrl: {type: String},
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
    };
  }

  /** @override */
  constructor() {
    super();
    this.queryParams = {};
    this.dirtyForms = [];

    /**
     * @type {PageJS.Context}
     * The context of the page. This should not be a LitElement property
     * because we don't want to re-render when updating this.
     */
    this._currentContext = undefined;
  }

  /** @override */
  stateChanged(state) {
    this.dirtyForms = ui.dirtyForms(state);
    this.queryParams = sitewide.queryParams(state);
    this.pageTitle = sitewide.pageTitle(state);
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('userDisplayName')) {
      store.dispatch(user.fetch(this.userDisplayName));
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

    // TODO(zhangtiff): Figure out some way to save Redux state between
    // page loads.

    // page doesn't handle users reloading the page or closing a tab.
    window.onbeforeunload = this._confirmDiscardMessage.bind(this);

    // Start a cron task to periodically request the status from the server.
    getServerStatusCron.start();

    // NOTE: In most cases, we want more general route handlers like
    // _initializeViewedProject() and _selectHotlist() to come AFTER more
    // specific route handlers. This is because the more specific route
    // handlers usually load the bundle for a given page, and we want the
    // actions caused by these handlers to happen after a bundle has loaded,
    // not before.
    // This may change if we change the granularity of our bundling.
    page('*', this._preRouteHandler.bind(this));
    page('/p/:project/issues/list_new', this._loadListPage.bind(this));
    page('/p/:project/issues/detail', this._loadIssuePage.bind(this));
    page('/p/:project/*', this._selectProject.bind(this));
    page(
        '/users/:user/hotlists/:hotlist',
        this._selectHotlist, this._loadHotlistIssuesPage.bind(this));
    page(
        '/users/:user/hotlists/:hotlist/details',
        this._loadHotlistDetailsPage.bind(this));
    page(
        '/users/:user/hotlists/:hotlist/people',
        this._loadHotlistPeoplePage.bind(this));
    page('/users/:user/hotlists/:hotlist/*', this._selectHotlist);
    page('*', this._postRouteHandler.bind(this));
    page();
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
    if (this._currentContext && this._currentContext.path &&
      ctx.path === this._currentContext.path) {
      Object.assign(ctx, this._currentContext);
      // Set ctx.handled to false, so we don't push the state to browser's
      // history.
      ctx.handled = false;
      return;
    }

    // Check if there were forms with unsaved data before loading the next
    // page.
    const discardMessage = this._confirmDiscardMessage();
    if (discardMessage && !confirm(discardMessage)) {
      Object.assign(ctx, this._currentContext);
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

    // Save the context of this page to be compared to later.
    this._currentContext = ctx;

    next();
  }

  /**
   * Handler that runs after a project page has loaded.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  _selectProject(ctx, next) {
    store.dispatch(project.select(ctx.params.project));
    store.dispatch(project.fetch(ctx.params.project));
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

    store.dispatch(issue.setIssueRef(
        Number.parseInt(ctx.queryParams.id), ctx.params.project));
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
        await import(/* webpackChunkName: "mr-list-page" */
            '../issue-list/mr-list-page/mr-list-page.js');
        this.page = 'list';
        break;
    }
    next();
  }

  /**
   * Gets the currently viewed HotlistRef from the URL, selects
   * it in the Redux store, and fetches the Hotlist data.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  _selectHotlist(ctx, next) {
    const hotlistRef = {
      owner: userIdOrDisplayNameToUserRef(ctx.params.user),
      name: ctx.params.hotlist,
    };
    store.dispatch(hotlist.select(hotlistRef));
    store.dispatch(hotlist.fetch(hotlistRef));
    store.dispatch(hotlist.fetchItems(hotlistRef));
    next();
  }

  /**
   * Loads mr-hotlist-details-page.js and makes it the currently viewed page.
   * @param {PageJS.Context} ctx A page.js Context containing routing state.
   * @param {function} next Passes execution on to the next registered callback.
   */
  async _loadHotlistDetailsPage(ctx, next) {
    await import(/* webpackChunkName: "mr-hotlist-details-page" */
        `../hotlist/mr-hotlist-details-page/mr-hotlist-details-page.js`);
    this.page = 'hotlist-details';
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
