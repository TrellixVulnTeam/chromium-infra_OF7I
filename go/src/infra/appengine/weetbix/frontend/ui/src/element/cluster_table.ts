// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css } from 'lit-element';

// ClusterTable lists the clusters tracked by Weetbix.
@customElement('cluster-table')
export class ClusterTable extends LitElement {
    @property()
    clusters: Cluster[] | undefined;

    connectedCallback() {
        super.connectedCallback()
        fetch("/api/cluster").then(r => r.json()).then(clusters => this.clusters = clusters);
    }

    render() {
        if (this.clusters === undefined) {
            return html`Loading...`;
        }
        return html`
        <div id="container">
            <h1>${this.clusters[0].project}: Clusters</h1>
            <table>
                <thead>
                    <tr>
                        <th>Cluster</th>
                        <th>Unexpected Failures (1d)</th>
                        <th>Unexpected Failures (3d)</th>
                        <th>Unexpected Failures (7d)</th>
                        <th>Unexonerated Failures (1d)</th>
                        <th>Unexonerated Failures (3d)</th>
                        <th>Unexonerated Failures (7d)</th>
                        <th>Affected Runs (1d)</th>
                        <th>Affected Runs (3d)</th>
                        <th>Affected Runs (7d)</th>
                    </tr>
                </thead>
                <tbody>
                    ${this.clusters.map(c => html`
                    <tr>
                        <td>${c.exampleFailureReason || c.clusterId}</td>
                        <td class="number">${c.unexpectedFailures1d}</td>
                        <td class="number">${c.unexpectedFailures3d}</td>
                        <td class="number">${c.unexpectedFailures7d}</td>
                        <td class="number">${c.unexoneratedFailures1d}</td>
                        <td class="number">${c.unexoneratedFailures3d}</td>
                        <td class="number">${c.unexoneratedFailures7d}</td>
                        <td class="number">${c.affectedRuns1d}</td>
                        <td class="number">${c.affectedRuns3d}</td>
                        <td class="number">${c.affectedRuns7d}</td>
                    </tr>`)}
                </tbody>
            </table>
        </div>
        `;
    }
    static styles = [css`
        #container {
            margin: 20px 14px;
        }
        h1 {
            font-size: 1em;
        }
        table {
            border-collapse: collapse;
        }
        th {
            font-weight: normal;
            color: var(--greyed-out-text-color);
        }
        td,th {
            padding: 4px;
        }
        td.number {
            text-align: right;
        }
    `];
}

// Cluster is the cluster information sent by the server.
interface Cluster {
    project: string;
    clusterId: number;
    unexpectedFailures1d: number;
    unexpectedFailures3d: number;
    unexpectedFailures7d: number;
    unexoneratedFailures1d: number;
    unexoneratedFailures3d: number;
    unexoneratedFailures7d: number;
    affectedRuns1d: number;
    affectedRuns3d: number;
    affectedRuns7d: number;
    exampleFailureReason: string;
}