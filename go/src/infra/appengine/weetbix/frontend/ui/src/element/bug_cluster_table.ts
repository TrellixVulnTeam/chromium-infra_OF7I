// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property } from 'lit-element';

// BugClusterTable lists the bug clusters tracked by Weetbix.
@customElement('bug-cluster-table')
export class BugClusterTable extends LitElement {
    @property()
    bugClusters: BugCluster[] | undefined;

    connectedCallback() {
        super.connectedCallback()
        fetch("/api/bugcluster").then(r => r.json()).then(bugClusters => this.bugClusters = bugClusters);
    }

    render() {
        if (this.bugClusters === undefined) {
            return html`Loading...`;
        }
        return html`
        <table>
            <thead>
                <tr>
                    <th>Project</th>
                    <th>Bug</th>
                    <th>Associated Cluster ID</th>
                </tr>
            </thead>
            <tbody>
                ${this.bugClusters.map(c => html`
                <tr>
                    <td>${c.project}</td>
                    <td>${c.bug}</td>
                    <td>${c.associatedClusterId}</td>
                </tr>`)}
            </tbody>
        </table>
        `;
    }
}

// BugCluster is the bug cluster information sent by the server.
interface BugCluster {
    project: number;
    bug: number;
    associatedClusterId: number;
}
