// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@material/mwc-circular-progress/mwc-circular-progress';

import {
    css,
    customElement,
    html,
    LitElement,
    property
} from 'lit-element';

import { Project } from '../../../services/project';

@customElement('project-card')
export class ProjectCard extends LitElement {

    @property({attribute: false})
    project: Project | null = null;

    protected render() {
        if (this.project == null) {
            return html`
            <mwc-circular-progress></mwc-circular-progress>
            `;
        } else {
            return html`
            <a id="card" href="/p/${this.project.project}/clusters">
                ${this.project.displayName}
            </a>
            `;
        }
    }

    static styles = css`
    #card {
        padding: 1rem;
        display: flex;
        justify-content: center;
        align-items: center;
        box-shadow: 0px 2px 1px -1px rgb(0 0 0 / 20%), 0px 1px 1px 0px rgb(0 0 0 / 14%), 0px 1px 3px 0px rgb(0 0 0 / 12%);
        font-weight: bold;
        font-size: 1.5rem;
        text-decoration: none;
        color: black;
        border-radius: 4px;
        transition: transform .2s;
        height: 100%;
    }
    #card:hover {
        transform: scale(1.1);
    }
    `;
}