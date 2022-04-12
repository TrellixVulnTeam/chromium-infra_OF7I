/* eslint-disable @typescript-eslint/indent */
// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './elements/project_card';

import {
    css,
    customElement,
    html,
    LitElement,
    state
} from 'lit-element';

import {
    getProjectsService,
    ListProjectsRequest,
    Project
} from '../../services/project';

/**
 *  Represents the home page where the user selects their project.
 */
@customElement('home-page')
export class HomePage extends LitElement {

    @state()
    projects: Project[] | null = [];

    connectedCallback() {
        super.connectedCallback();
        this.fetch();
    }

    async fetch() {
        const service = getProjectsService();
        const request: ListProjectsRequest = {};
        const response = await service.list(request);
        // Chromium milestone projects are explicitly ignored by the backend, match this in the frontend.
        this.projects = response.projects?.filter(p => !/^(chromium|chrome)-m[0-9]+$/.test(p.project)) || null;
        this.requestUpdate();
    }

    render() {
        return html`
        <main id="container">
            <section id="title">
                <h1>Projects</h1>
            </section>
            <section id="projects">
                ${this.projects?.map((project) => {
                    return html`
                    <project-card .project=${project}></project-card>
                    `;
                })}
            </section>
        </main>
        `;
    }

    static styles = css`
    #container {
        margin-left: 16rem;
        margin-right: 16rem;
    }

    #projects {
        margin: auto;
        display: grid;
        grid-template-columns: repeat(6, 1fr);
        gap: 2rem;
    }
    `;

}
