// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

// URL where announcements are fetched from.
const ANNOUNCEMENT_SERVICE =
  'https://chopsdash.appspot.com/prpc/dashboard.ChopsAnnouncements/SearchAnnouncements';

// Prefix prepended to responses for security reasons.
export const XSSI_PREFIX = ')]}\'';

const FETCH_HEADERS = Object.freeze({
  'accept': 'application/json',
  'content-type': 'application/json',
});

// How often to refresh announcements.
export const REFRESH_TIME_MS = 5 * 60 * 1000;

/**
 * @typedef {Object} Announcement
 * @property {string} id
 * @property {string} messageContent
 */

/**
 * @typedef {Object} AnnouncementResponse
 * @property {Array<Announcement>} announcements
 */

/**
 * `<chops-announcement>` displays a ChopsDash message when there's an outage
 * or other important announcement.
 *
 * @customElement chops-announcement
 */
export class ChopsAnnouncement extends LitElement {
  /** @override */
  static get styles() {
    return css`
      :host {
        display: block;
        width: 100%;
      }
      p {
        display: block;
        color: #222;
        font-size: 13px;
        background: #FFCDD2; /* Material design red */
        width: 100%;
        text-align: center;
        padding: 0.5em 16px;
        box-sizing: border-box;
        margin: 0;
        /* Using a red-tinted grey border makes hues feel harmonious. */
        border-bottom: 1px solid #D6B3B6;
      }
    `;
  }
  /** @override */
  render() {
    if (this._error) {
      return html`<p><strong>Error: </strong>${this._error}</p>`;
    }
    return html`
      ${this._announcements.map(
      ({messageContent}) => html`<p>${messageContent}</p>`)}
    `;
  }

  /** @override */
  static get properties() {
    return {
      service: {type: String},
      _error: {type: String},
      _announcements: {type: Array},
    };
  }

  /** @override */
  constructor() {
    super();

    /** @type {string} */
    this.service = undefined;
    /** @type {string} */
    this._error = undefined;
    /** @type {Array<Announcement>} */
    this._announcements = [];

    /** @type {number} Interval ID returned by window.setInterval. */
    this._interval = undefined;
  }

  /** @override */
  updated(changedProperties) {
    if (changedProperties.has('service')) {
      if (this.service) {
        this.startRefresh();
      } else {
        this.stopRefresh();
      }
    }
  }

  /** @override */
  disconnectedCallback() {
    super.disconnectedCallback();

    this.stopRefresh();
  }

  /**
   * Set up autorefreshing logic or announcement information.
   */
  startRefresh() {
    this.stopRefresh();
    this.refresh();
    this._interval = window.setInterval(() => this.refresh(), REFRESH_TIME_MS);
  }

  /**
   * Logic for clearing refresh behavior.
   */
  stopRefresh() {
    if (this._interval) {
      window.clearInterval(this._interval);
    }
  }

  /**
   * Refresh the announcement banner.
   */
  async refresh() {
    try {
      const {announcements} = await this.fetch(this.service);
      this._error = undefined;
      this._announcements = announcements;
    } catch (e) {
      this._error = e.message;
      this._announcements = [];
    }
  }

  /**
   * Fetches the announcement for a given service.
   * @param {string} service Name of the service to fetch from ChopsDash.
   *   ie: "monorail"
   * @return {Promise<AnnouncementResponse>} ChopsDash response JSON.
   * @throws {Error} If something went wrong while fetching.
   */
  async fetch(service) {
    const message = {
      retired: false,
      platformName: service,
    };

    const response = await window.fetch(ANNOUNCEMENT_SERVICE, {
      method: 'POST',
      headers: FETCH_HEADERS,
      body: JSON.stringify(message),
    });

    if (!response.ok) {
      throw new Error('Something went wrong while fetching announcements');
    }

    // We can't use response.json() because of the XSSI prefix.
    const text = await response.text();

    if (!text.startsWith(XSSI_PREFIX)) {
      throw new Error(`No XSSI prefix in announce response: ${XSSI_PREFIX}`);
    }

    return JSON.parse(text.substr(XSSI_PREFIX.length));
  }
}

customElements.define('chops-announcement', ChopsAnnouncement);
