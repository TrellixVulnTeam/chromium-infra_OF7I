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
        const clusterLink = (cluster: Cluster): string => {
            return `/project/${cluster.project}/cluster/${cluster.clusterId}`;
        }
        return html`
        <div id="container">
            <h1>Clusters in project ${this.clusters[0].project}</h1>
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
                        <td class="failure-reason">
                            <a href=${clusterLink(c)}>
                                ${c.exampleFailureReason || c.clusterId}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.unexpectedFailures1d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.unexpectedFailures3d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.unexpectedFailures7d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.unexoneratedFailures1d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.unexoneratedFailures3d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.unexoneratedFailures7d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.affectedRuns1d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.affectedRuns3d}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${c.affectedRuns7d}
                            </a>
                        </td>
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
            font-size: 18px;
            font-weight: normal;
        }
        table {
            border-collapse: collapse;
            max-width: 100%;
        }
        th {
            font-weight: normal;
            color: var(--greyed-out-text-color);
            text-align: left;
            font-size: var(--font-size-small);
        }
        td,th {
            padding: 4px;
            max-width: 80%;
        }
        td.number {
            text-align: right;
        }
        td a {
            display: block;
            text-decoration: none;
            color: var(--default-text-color);
        }
        tbody tr:hover {
            background-color: var(--light-active-color);
        }
        .failure-reason {
            word-break: break-all;
            font-size: var(--font-size-small);
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