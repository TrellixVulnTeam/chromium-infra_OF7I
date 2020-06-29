// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

/**
 * `<mr-star>`
 *
 * A button for starring a resource. Does not directly integrate with app
 * state. Subclasses by <mr-issue-star> and <mr-project-star>, which add
 * resource-specific logic for state management.
 *
 */
export class MrStar extends LitElement {
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
    const {isStarred} = this;
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <button class="star-button"
        @click=${this._loginOrStar}
        ?disabled=${this.disabled}
        title=${this._starToolTip}
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

  /** @override */
  static get properties() {
    return {
      /**
       * Note: In order for re-renders to happen based on the getters defined
       * in this class, those getters must have values based on properties.
       * Subclasses of <mr-star> are not expected to inherit <mr-star>'s
       * properties, but they should make sure their getter implementations
       * are also backed by properties.
       */
      _isStarred: {type: Boolean},
      _isLoggedIn: {type: Boolean},
      _canStar: {type: Boolean},
      _requesting: {type: Boolean},
    };
  }

  /** @override */
  constructor() {
    super();
    /**
     * @type {boolean} Whether the user has starred the resource or not.
     */
    this._isStarred = false;

    /**
     * @type {boolean} If the user is logged in.
     */
    this._isLoggedIn = false;

    /**
     * @return {boolean} Whether the user has permission to star the star.
     */
    this._canStar = true;

    /**
     * @return {boolean} Whether there's an in-flight request to star
     * the resource.
     */
    this._requesting = false;
  }

  /** @override */
  connectedCallback() {
    super.connectedCallback();

    // Prevent clicks on this element from causing navigation if the element
    // is embedded inside a link.
    this.addEventListener('click', (e) => e.preventDefault());
  }

  /**
   * @return {boolean} If the user is logged in.
   */
  get isLoggedIn() {
    return this._isLoggedIn;
  }

  /**
   * @return {boolean} If there's an in-flight request that might affect the
   *   star's data.
   */
  get requesting() {
    return this._requesting;
  }

  /**
   * @return {boolean} Whether the resource is starred or not.
   */
  get isStarred() {
    return this._isStarred;
  }

  /**
   * @return {boolean} If the user has permission to star.
   */
  get canStar() {
    return this._canStar;
  }

  /**
   * @return {boolean} If the star button should be disabled right now.
   * Note that the star button can be enabled either because the user is logged
   * in or because the user is able to star.
   */
  get disabled() {
    return this.isLoggedIn && !this._starringEnabled;
  }

  /**
   * @return {boolean}
   */
  get _starringEnabled() {
    return this.isLoggedIn && this.canStar && !this.requesting;
  }

  /**
   * @return {string} The name of the resource kind being starred.
   * ie: issue, project, etc.
   */
  get type() {
    return 'resource';
  }

  /**
   * @return {string} the title to display on the star button.
   */
  get _starToolTip() {
    if (!this.isLoggedIn) {
      return `Login to star this ${this.type}.`;
    }
    if (!this.canStar) {
      return `You don't have permission to star this ${this.type}.`;
    }
    if (this.requesting) {
      return `Loading star state for this ${this.type}.`;
    }
    return `${this.isStarred ? 'Unstar' : 'Star'} this ${this.type}.`;
  }

  /**
   * Logins the user if they're not logged in. Otherwise, stars or
   * unstars the resource based on star state.
   */
  _loginOrStar() {
    if (!this.isLoggedIn) {
      this.login();
    } else {
      this.toggleStar();
    }
  }

  /**
   * Logs in the user.
   */
  login() {
    // TODO(crbug.com/monorail/6073): Replace this logic with a function call
    // when moving authentication to frontend.
    // HACK: In our current login implementation, login URLs can only be
    // generated by the backend which makes piping a login URL into a component
    // a <mr-star> complex. To get around this, we're using the
    // legacy window.CS_env infrastructure.
    window.location.href = window.CS_env.login_url;
  }

  /**
   * Stars or unstars the resource based on the user's interaction.
   */
  toggleStar() {
    if (!this._starringEnabled) return;
    if (this.isStarred) {
      this.unstar();
    } else {
      this.star();
    }
  }

  /**
   * Stars the given resource. To be implemented by a subclass.
   */
  star() {
    throw new Error('Method not implemented.');
  }

  /**
   * Unstars the given resource. To be implemented by a subclass.
   */
  unstar() {
    throw new Error('Method not implemented.');
  }
}

customElements.define('mr-star', MrStar);
