// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property } from 'lit-element';

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
        <table>
            <thead>
                <tr>
                    <th>Project</th>
                    <th>Cluster ID</th>
                    <th>Unexpected Failures (1d)</th>
                    <th>Unexpected Failures (3d)</th>
                    <th>Unexpected Failures (7d)</th>
                    <th>Unexonerated Failures (1d)</th>
                    <th>Unexonerated Failures (3d)</th>
                    <th>Unexonerated Failures (7d)</th>
                    <th>Affected Runs (1d)</th>
                    <th>Affected Runs (3d)</th>
                    <th>Affected Runs (7d)</th>
                    <th>Example Failure Reason</th>
                </tr>
            </thead>
            <tbody>
                ${this.clusters.map(c => html`
                <tr>
                    <td>${c.project}</td>
                    <td>${c.clusterId}</td>
                    <td>${c.unexpectedFailures1d}</td>
                    <td>${c.unexpectedFailures3d}</td>
                    <td>${c.unexpectedFailures7d}</td>
                    <td>${c.unexoneratedFailures1d}</td>
                    <td>${c.unexoneratedFailures3d}</td>
                    <td>${c.unexoneratedFailures7d}</td>
                    <td>${c.affectedRuns1d}</td>
                    <td>${c.affectedRuns3d}</td>
                    <td>${c.affectedRuns7d}</td>
                    <td>${c.exampleFailureReason}</td>
                </tr>`)}
            </tbody>
        </table>
        `;
    }
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