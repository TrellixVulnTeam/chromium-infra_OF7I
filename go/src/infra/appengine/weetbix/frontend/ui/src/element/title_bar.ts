// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { css, customElement, html, LitElement, property } from 'lit-element';

/**
 * Renders page header, including a sign-in widget, a settings button, and a
 * feedback button, at the top of the child nodes.
 * Refreshes the page when a new clientId is provided.
 */
@customElement('title-bar')
export class TitleBar extends LitElement {
  @property()
  email = "";

  @property()
  logoutUrl = "";

  protected render() {
    return html`
    <div id="container">
      <div id="title-container">
        <a href="/" id="title-link">
          <img id="chromium-icon" src="https://storage.googleapis.com/chrome-infra/lucy-small.png" />
          <span id="headline">Weetbix</span>
        </a>
      </div>
      <div id="signin">
        ${this.email} | <a href=${this.logoutUrl}>Logout</a>
      </div>
    </div>
    <div id="secondary-container">
      <ul>
        <li><a href="/">Clusters</a></li>
        <li><a href="/bugs">Bugs</a></li>
      </ul>
    </div>
    <slot></slot>
    `;
  }

  static styles = [
    css`
      #container {
        box-sizing: border-box;
        height: 52px;
        padding: 10px 0;
        display: flex;
      }
      #title-container {
        display: flex;
        flex: 1 1 100%;
        align-items: center;
        margin-left: 14px;
      }
      #title-link {
        display: flex;
        align-items: center;
        text-decoration: none;
      }
      #chromium-icon {
        display: inline-block;
        width: 32px;
        height: 32px;
        margin-right: 8px;
      }
      #headline {
        color: var(--light-text-color);
        font-family: 'Google Sans', 'Helvetica Neue', sans-serif;
        font-size: 18px;
        font-weight: 300;
        letter-spacing: 0.25px;
      }
      #signin {
        margin-right: 14px;
        flex-shrink: 0;
        display: inline-block;
        height: 32px;
        line-height: 32px;
      }
      #signin a {
        color: var(--default-text-color);
        text-decoration: underline;
        cursor: pointer;
      }
      #secondary-container {
        background-color: var(--block-background-color);
      }
      #secondary-container ul {
          margin: 0;
          padding-left: 14px;
          display: flex;
          list-style-type: none;
      }
      #secondary-container li {
          padding: 4px 28px 4px 0;
      }
      #secondary-container a {
        color: var(--active-text-color);
        text-decoration: none;
      }
    `,
  ];
}
