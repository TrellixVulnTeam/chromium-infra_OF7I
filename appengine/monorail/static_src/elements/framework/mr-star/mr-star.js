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
    const {isStarred, canStar} = this;
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
      <button class="star-button"
        @click=${this._toggleStar}
        ?disabled=${!canStar}
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
      _isStarred: {type: Boolean},
      _canStar: {type: Boolean},
    };
  }

  /** @override */
  constructor() {
    super();
    /**
     * @type {boolean} Whether the user has starred the issue or not.
     */
    this._isStarred = false;

    /**
     * @return {boolean} Whether the user is able to star the current object.
     */
    this._canStar = false;
  }

  /**
   * Gets whether a resource is starred or not.
   *
   * Note: In order for re-renders to happen based on this getter,
   * the subclass must compute this value based on a property.
   * @return {boolean} Whether the resource is starred or not.
   */
  get isStarred() {
    return this._isStarred;
  }

  /**
   * Gets whether a user can star the current resource.
   *
   * Note: In order for re-renders to happen based on this getter,
   * the subclass must compute this value based on a property.
   * @return {boolean} If the user can star.
   */
  get canStar() {
    return this._canStar;
  }

  /**
   * @return {string} the title to display on the star button.
   */
  get _starToolTip() {
    if (!this.canStar) {
      return `You don't have permission to star this issue.`;
    }
    return `${this.isStarred ? 'Unstar' : 'Star'} this issue.`;
  }

  /**
   * Click handler for triggering toggleStar.
   * @param {MouseEvent} e
   */
  _toggleStar(e) {
    e.preventDefault();
    this.toggleStar();
  }

  /**
   * Stars or unstars the resource based on the user's interaction.
   */
  toggleStar() {
    if (!this.canStar) return;
    if (this.isStarred) {
      this.unstar();
    } else {
      this.star();
    }
  }

  /**
   * Stars the given resource.
   */
  star() {
    return;
  }

  /**
   * Unstars the given resource.
   */
  unstar() {
    return;
  }
}

customElements.define('mr-star', MrStar);
