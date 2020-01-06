// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import {connectStore} from 'reducers/base.js';
import * as issue from 'reducers/issue.js';
import './mr-phase.js';

/**
 * `<mr-launch-overview>`
 *
 * This is a shorthand view of the phases for a user to see a quick overview.
 *
 */
export class MrLaunchOverview extends connectStore(LitElement) {
  /** @override */
  static get styles() {
    return css`
      :host {
        width: 100%;
        display: flex;
        flex-flow: column;
        justify-content: flex-start;
        align-items: stretch;
      }
      :host([hidden]) {
        display: none;
      }
      mr-phase {
        margin-bottom: 0.75em;
      }
    `;
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons"
            rel="stylesheet">
      ${this.phases.map((phase) => html`
        <mr-phase
          .phaseName=${phase.phaseRef.phaseName}
          .approvals=${this._approvalsForPhase(this.approvals, phase.phaseRef.phaseName)}
        ></mr-phase>
      `)}
      ${this._phaselessApprovals.length ? html`
        <mr-phase .approvals=${this._phaselessApprovals}></mr-phase>
      `: ''}
    `;
  }

  /** @override */
  static get properties() {
    return {
      approvals: {type: Array},
      phases: {type: Array},
      hidden: {
        type: Boolean,
        reflect: true,
      },
    };
  }

  /** @override */
  constructor() {
    super();
    this.approvals = [];
    this.phases = [];
    this.hidden = true;
  }

  /** @override */
  stateChanged(state) {
    if (!issue.viewedIssue(state)) return;

    this.approvals = issue.viewedIssue(state).approvalValues || [];
    this.phases = issue.viewedIssue(state).phases || [];
  }

  /** @override */
  update(changedProperties) {
    if (changedProperties.has('phases') || changedProperties.has('approvals')) {
      this.hidden = !this.phases.length && !this.approvals.length;
    }
    super.update(changedProperties);
  }

  get _phaselessApprovals() {
    return this._approvalsForPhase(this.approvals);
  }

  _approvalsForPhase(approvals, phaseName) {
    return (approvals || []).filter((a) => {
      // We can assume phase names will be unique.
      return a.phaseRef.phaseName == phaseName;
    });
  }
}
customElements.define('mr-launch-overview', MrLaunchOverview);
