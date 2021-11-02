// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, state, css } from 'lit-element';

// ClusterTable lists the clusters tracked by Weetbix.
@customElement('cluster-table')
export class ClusterTable extends LitElement {
    @property()
    project: string;

    @state()
    clusters: Cluster[] | undefined;

    connectedCallback() {
        super.connectedCallback()
        this.project = "chromium";
        fetch(`/api/projects/${encodeURIComponent(this.project)}/clusters`).then(r => r.json()).then(clusters => this.clusters = clusters);
    }

    render() {
        if (this.clusters === undefined) {
            return html`Loading...`;
        }
        const clusterLink = (cluster: Cluster): string => {
            return `/projects/${encodeURIComponent(this.project)}/clusters/${encodeURIComponent(cluster.clusterAlgorithm)}/${encodeURIComponent(cluster.clusterId)}`;
        }
        const clusterDescription = (cluster: Cluster): string => {
            if (cluster.clusterAlgorithm.startsWith("testname-")) {
                return cluster.exampleTestId;
            } else if (cluster.clusterAlgorithm.startsWith("failurereason-")) {
                return cluster.exampleFailureReason;
            }
            return `${cluster.clusterAlgorithm}/${cluster.clusterId}`;
        }
        const metric = (counts: Counts): number => {
            return counts.nominal;
        }
        return html`
        <div id="container">
            <h1>Clusters in project ${this.project}</h1>
            <table>
                <thead>
                    <tr>
                        <th>Cluster</th>
                        <th>Presubmit Runs Failed (1d)</th>
                        <th>Presubmit Runs Failed (3d)</th>
                        <th>Presubmit Runs Failed (7d)</th>
                        <th>Test Runs Failed (1d)</th>
                        <th>Test Runs Failed (3d)</th>
                        <th>Test Runs Failed (7d)</th>
                        <th>Unexpected Failures (1d)</th>
                        <th>Unexpected Failures (3d)</th>
                        <th>Unexpected Failures (7d)</th>
                    </tr>
                </thead>
                <tbody>
                    ${this.clusters.map(c => html`
                    <tr>
                        <td class="failure-reason">
                            <a href=${clusterLink(c)}>
                                ${clusterDescription(c)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.presubmitRejects1d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.presubmitRejects3d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.presubmitRejects7d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.testRunFailures1d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.testRunFailures3d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.testRunFailures7d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.failures1d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.failures3d)}
                            </a>
                        </td>
                        <td class="number">
                            <a href=${clusterLink(c)}>
                                ${metric(c.failures7d)}
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
    clusterAlgorithm: string;
    clusterId: number;
    presubmitRejects1d: Counts;
    presubmitRejects3d: Counts;
    presubmitRejects7d: Counts;
    testRunFailures1d: Counts;
    testRunFailures3d: Counts;
    testRunFailures7d: Counts;
    failures1d: Counts;
    failures3d: Counts;
    failures7d: Counts;
    exampleFailureReason: string;
    exampleTestId: string;
}

interface Counts {
    nominal: number;
    preExoneration: number;
    residual: number;
    residualPreExoneration: number;
}
