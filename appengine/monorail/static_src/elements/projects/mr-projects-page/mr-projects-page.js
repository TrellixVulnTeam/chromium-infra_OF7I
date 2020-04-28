// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';
import {store, connectStore} from 'reducers/base.js';
import {SHARED_STYLES} from 'shared/shared-styles.js';
import 'elements/framework/mr-star-button/mr-star-button.js';
import 'shared/typedef.js';
import 'elements/chops/chops-chip/chops-chip.js';

import * as project from 'reducers/project.js';


/**
 * `<mr-projects-page>`
 *
 * Displays list of all projects.
 *
 */
export class MrProjectsPage extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      css`
        :host {
          box-sizing: border-box;
          display: block;
          padding: 1em 8px;
          padding-left: 40px; /** 32px + 8px */
          margin: auto;
          max-width: 1280px;
          width: 100%;
        }
        :host::after {
          content: "";
          background-image: url('/static/images/chromium.svg');
          background-repeat: no-repeat;
          background-position: right -100px bottom -150px;
          background-size: 700px;
          opacity: 0.09;
          width: 100%;
          height: 100%;
          bottom: 0;
          right: 0;
          position: fixed;
          z-index: -1;
        }
        h2 {
          font-size: 20px;
          letter-spacing: 0.1px;
          font-weight: 500;
          margin-top: 1em;
        }
        .project-header {
          display: flex;
          align-items: flex-start;
          flex-direction: row;
          justify-content: space-between;
          font-size: 16px;
          line-height: 24px;
          margin: 0;
          margin-bottom: 16px;
          padding-top: 0.1em;
          padding-bottom: 16px;
          letter-spacing: 0.1px;
          font-weight: 500;
          width: 100%;
          border-bottom: var(--chops-normal-border);
          border-color: var(--chops-gray-400);
        }
        .project-title {
          display: flex;
          flex-direction: column;
        }
        h3 {
          margin: 0;
          padding: 0;
          font-weight: inherit;
          font-size: inherit;
          transition: color var(--chops-transition-time) ease-in-out;
        }
        h3:hover {
          color: var(--chops-link-color);
        }
        .subtitle {
          color: var(--chops-gray-600);
          font-size: var(--chops-main-font-size);
          line-height: var(--chops-main-font-size);
          font-weight: normal;
        }
        .project-container {
          display: flex;
          align-items: stretch;
          flex-wrap: wrap;
          width: 100%;
          padding: 0.5em 0;
          margin-bottom: 4em;
        }
        .project {
          background: white;
          width: 220px;
          margin-right: 32px;
          margin-bottom: 32px;
          display: flex;
          flex-direction: column;
          align-items: flex-start;
          justify-content: flex-start;
          border-radius: 4px;
          border: var(--chops-normal-border);
          padding: 16px;
          color: var(--chops-primary-font-color);
          font-weight: normal;
          line-height: 16px;
          transition: all var(--chops-transition-time) ease-in-out;
        }
        .project:hover {
          text-decoration: none;
          cursor: pointer;
          box-shadow: 0 2px 6px hsla(0,0%,0%,0.12),
            0 1px 3px hsla(0,0%,0%,0.24);
        }
        .project > p {
          margin: 0;
          margin-bottom: 32px;
          flex-grow: 1;
        }
        .view-project-link {
          text-transform: uppercase;
          margin: 0;
          font-weight: 600;
          flex-grow: 0;
        }
        .view-project-link:hover {
          text-decoration: underline;
        }
      `,
    ];
  }

  /** @override */
  render() {
    const myProjects = this.myProjects;
    const otherProjects = this.otherProjects;
    const noProjects = !myProjects.length && !otherProjects.length;

    if (this._isFetchingProjects && noProjects) {
      return html`Loading...`;
    }

    if (noProjects) {
      return html`No projects found.`;
    }

    if (!myProjects.length) {
      // Skip sorting projects into different sections if the user
      // has no projects.
      return html`
        <h2>All projects</h2>
        <div class="project-container all-projects">
          ${otherProjects.map((project) => this._renderProject(project))}
        </div>
      `;
    }

    return html`
      <h2>My projects</h2>
      <div class="project-container my-projects">
        ${myProjects.map((project) => this._renderProject(project, 'Owner'))}
      </div>

      <h2>Browse other projects</h2>
      <div class="project-container other-projects">
        ${otherProjects.map((project) => this._renderProject(project))}
      </div>
    `;
  }

  /** @override */
  static get properties() {
    return {
      _projects: {type: Array},
      _isFetchingProjects: {type: Boolean},
    };
  }

  /** @override */
  constructor() {
    super();
    /**
     * @type {Array<Project>}
     */
    this._projects = [];
    /**
     * @type {boolean}
     */
    this._isFetchingProjects = false;
  }

  /** @override */
  connectedCallback() {
    super.connectedCallback();
    store.dispatch(project.list());
  }

  /** @override */
  stateChanged(state) {
    this._projects = project.all(state);
    this._isFetchingProjects = project.requests(state).list.requesting;
  }

  /**
   * @param {Project} project
   * @param {string=} role
   * @return {TemplateResult}
   */
  _renderProject(project, role) {
    return html`
      <a href="/p/${project.displayName}/" class="project">
        <div class="project-header">
          <span class="project-title">
            <h3>${project.displayName}</h3>
            <span class="subtitle" ?hidden=${!role} title="My role: ${role}">
              ${role}
            </span>
          </span>

          <mr-star-button></mr-star-button>
        </div>
        <p>
          ${project.summary}
        </p>
        <button class="view-project-link linkify">
          View project
        </button>
      </a>
    `;
  }

  /**
   * Projects the currently logged in user is a member of.
   * @return {Array<Project>}
   */
  get myProjects() {
    // TODO(crbug.com/monorail/6966): Implement this.
    return [];
  }

  /**
   * Projects the currently logged in user is not a member of.
   * @return {Array<Project>}
   */
  get otherProjects() {
    return this._projects;
  }
}
customElements.define('mr-projects-page', MrProjectsPage);
