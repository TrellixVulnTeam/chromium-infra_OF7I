// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import {connectStore, store} from 'reducers/base.js';
import * as users from 'reducers/users.js';
import {stars} from 'reducers/stars.js';
import {projectAndUserToStarName} from 'shared/converters.js';
import {MrStar} from './mr-star.js';
import 'shared/typedef.js';


/**
 * `<mr-project-star>`
 *
 * A button for starring a project.
 *
 */
export class MrProjectStar extends connectStore(MrStar) {
  /** @override */
  static get properties() {
    return {
      /**
       * Resource name of the project being starred.
       */
      name: {type: String},
      /**
       * List of all stars, indexed by star name.
       */
      _stars: {type: Object},
      /**
       * Whether project stars are currently being fetched.
       */
      _fetchingStars: {type: Boolean},
      /**
       * Request data for projects currently being starred.
       */
      _starringProjects: {type: Object},
      /**
       * Request data for projects currently being unstarred.
       */
      _unstarringProjects: {type: Object},
      /**
       * The currently logged in user. Required to determine if the user can
       * star.
       */
      _currentUserName: {type: String},
    };
  }

  /** @override */
  constructor() {
    super();
    /** @type {string} */
    this.name = undefined;

    /** @type {boolean} */
    this._fetchingStars = false;

    /** @type {Object<ProjectStarName, ReduxRequestState>} */
    this._starringProjects = {};

    /** @type {Object<ProjectStarName, ReduxRequestState>} */
    this._unstarringProjects = {};

    /** @type {Object<StarName, Star>} */
    this._stars = {};

    /** @type {string} */
    this._currentUserName = undefined;
  }

  /** @override */
  stateChanged(state) {
    this._currentUserName = users.currentUserName(state);

    this._stars = stars.byName(state);

    const requests = stars.requests(state);
    this._fetchingStars = requests.listProjects.requesting;
    this._starringProjects = requests.starProject;
    this._unstarringProjects = requests.unstarProject;
  }

  /**
   * @return {string} The resource name of the ProjectStar.
   */
  get _starName() {
    return projectAndUserToStarName(this.name, this._currentUserName);
  }

  /**
   * @return {ProjectStar} The ProjectStar object for the referenced project,
   *   if one exists.
   */
  get _projectStar() {
    const name = this._starName;
    if (!(name in this._stars)) return {};
    return this._stars[name];
  }

  /**
   * @return {boolean} Whether there's an in-flight star request.
   */
  get _isStarring() {
    const requestKey = this._starName;
    if (requestKey in this._starringProjects &&
        this._starringProjects[requestKey].requesting) {
      return true;
    }
    if (requestKey in this._unstarringProjects &&
        this._unstarringProjects[requestKey].requesting) {
      return true;
    }
    return false;
  }

  /** @override */
  get canStar() {
    return !!(this._currentUserName && !this._fetchingStars &&
        !this._isStarring);
  }

  /** @override */
  get isStarred() {
    return !!(this._projectStar && this._projectStar.name);
  }

  /** @override */
  star() {
    store.dispatch(stars.starProject(this.name, this._currentUserName));
  }

  /** @override */
  unstar() {
    store.dispatch(stars.unstarProject(this.name, this._currentUserName));
  }
}

customElements.define('mr-project-star', MrProjectStar);
