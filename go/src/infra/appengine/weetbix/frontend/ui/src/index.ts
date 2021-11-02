// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './styles/style.css';

import { customElement, html, LitElement, property } from 'lit-element';
import { Context, Router } from '@vaadin/router';
import './element/bug_cluster_table.ts';
import './element/cluster_table.ts';
import './element/cluster_page.ts';
import './element/not_found_page.ts';
import './element/title_bar.ts';


// MonorailTest excersises the monorail API in the server and displays an
// issue summary to verify it is working.
// This component is only temporary, so keeping it here rather than creating
// a file to hold it.
@customElement('monorail-test')
export class MonorailTest extends LitElement {
    @property()
    issue: Issue | undefined;

    connectedCallback() {
        super.connectedCallback()
        fetch("/api/monorailtest").then(r => r.json()).then(issue => this.issue = issue);
    }

    render() {
        if (this.issue === undefined) {
            return html`Loading...`;
        }
        return html`<p>Issue summary: ${this.issue.summary}</p>`;
    }
}

// Issue is a part of the monorail issue data sent from the server.
interface Issue {
    summary: string;
}


const outlet = document.getElementById('outlet');
const router = new Router(outlet);
// serverRoute can be used as the action of routes that should be handled by the server
// instead of in the client.
const serverRoute = (ctx: Context) => { window.location.pathname = ctx.pathname; }
router.setRoutes([
    { path: '/auth/(.*)', action: serverRoute },  // For logout links.
    { path: '/', component: 'cluster-table' },
    { path: '/projects/:project/clusters/:algorithm/:id', component: 'cluster-page' },
    { path: '/monorail-test', component: 'monorail-test' },
    { path: '/bugcluster', component: 'bug-cluster-table' },
    { path: '(.*)', component: 'not-found-page' },
]);