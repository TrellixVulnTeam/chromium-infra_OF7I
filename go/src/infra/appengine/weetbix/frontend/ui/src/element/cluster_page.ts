// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state } from 'lit-element';
import { RouterLocation } from '@vaadin/router';

// ClusterPage lists the clusters tracked by Weetbix.
@customElement('cluster-page')
export class ClusterPage extends LitElement {
    @property()
    location: RouterLocation;

    @state()
    cluster: Cluster | undefined;

    @property()
    projectId: string;

    @property()
    clusterId: string;


    connectedCallback() {
        super.connectedCallback()

        // Take the first parameter value only.
        this.projectId = typeof this.location.params.project == 'string' ? this.location.params.project : this.location.params.project[0];
        this.clusterId = typeof this.location.params.id == 'string' ? this.location.params.id : this.location.params.id[0];

        fetch(`/api/project/${encodeURIComponent(this.projectId)}/cluster/${encodeURIComponent(this.clusterId)}`)
            .then(r => r.json())
            .then(cluster => this.cluster = cluster);
    }

    render() {
        if (this.cluster === undefined) {
            return html`Loading...`;
        }
        const c = this.cluster;
        const merged = mergeSubClusters([c.affectedTests1d, c.affectedTests3d, c.affectedTests7d]);
        return html`
        <div id="container">
            <h1>Cluster <span class="cluster-id">${c.clusterId}</span></h1>
            <h2>Example Failure</h2>
            <pre class="failure-reason">${c.exampleFailureReason || c.clusterId}</pre>
            <h2>Impact</h2>
            <table>
                <thead>
                    <tr>
                        <th></th>
                        <th>1 day</th>
                        <th>3 days</th>
                        <th>7 days</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <th>Unexpected Failures</th>
                        <td class="number">${c.unexpectedFailures1d}</td>
                        <td class="number">${c.unexpectedFailures3d}</td>
                        <td class="number">${c.unexpectedFailures7d}</td>
                    </tr>
                    <tr>
                        <th>Unexonerated Failures</th>
                        <td class="number">${c.unexoneratedFailures1d}</td>
                        <td class="number">${c.unexoneratedFailures3d}</td>
                        <td class="number">${c.unexoneratedFailures7d}</td>
                    </tr>
                    <tr>
                        <th>Affected Runs</th>
                        <td class="number">${c.affectedRuns1d}</td>
                        <td class="number">${c.affectedRuns3d}</td>
                        <td class="number">${c.affectedRuns7d}</td>
                    </tr>
                </tbody>
            </table>
            <h2>Breakdown</h2>
            <table>
                <thead>
                    <tr>
                        <th>Test</th>
                        <th>1 day</th>
                        <th>3 days</th>
                        <th>7 days</th>
                    </tr>
                </thead>
                <tbody>
                    ${merged.map(entry => html`
                    <tr>
                        <td class="test-id">${entry.name}</td>
                        ${entry.values.map(v => html`<td class="number">${v}</td>`)}
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
        h2 {
            margin-top: 40px;
            font-size: 16px;
            font-weight: normal;
        }
        .cluster-id {
            font-family: monospace;
            font-size: 80%;
            background-color: var(--light-active-color);
            border: solid 1px var(--active-color);
            border-radius: 20px;
            padding: 2px 8px;
        }
        .failure-reason {
            border: solid 1px var(--divider-color);
            background-color: var(--block-background-color);
            margin: 20px 14px;
            padding: 20px 14px;
            overflow-x: auto;
            font-size: var(--font-size-small);
        }
        table {
            border-collapse: collapse;
            max-width: 100%;
        }
        th {
            font-weight: normal;
            color: var(--greyed-out-text-color);
            text-align: left;
        }
        td,th {
            padding: 4px;
            max-width: 80%;
        }
        td.number {
            text-align: right;
        }
        tbody tr:hover {
            background-color: var(--light-active-color);
        }
        .test-id {
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
    affectedTests1d: SubCluster[];
    affectedTests3d: SubCluster[];
    affectedTests7d: SubCluster[];
    exampleFailureReason: string;
}

interface SubCluster {
    value: string;
    numFails: number;
}

interface MergedSubClusters {
    name: string;
    values: number[];
}

// Merge multiple related subcluster lists into a single list with multiple values.
// Missing values are filled with zeros. The returned list is sorted by the values.
//
// eg: mergeSubClusters([[{value: "a", numFails: 1}, {value: "b", numFails: 2}], [{value:"a", numFails: 3}]])
//     =>
//     [{name: "a", values: [1, 3]}, {name: "b", values: [2, 0]}]
const mergeSubClusters = (subClusters: Array<SubCluster[]>): MergedSubClusters[] => {
    const lookup: { [name: string]: number[] } = {};
    for (let i = 0; i < subClusters.length; i++) {
        const clusters = subClusters[i];
        for (const entry of clusters) {
            let values = lookup[entry.value]
            if (!values) {
                values = new Array(subClusters.length).fill(0);
                lookup[entry.value] = values;
            }
            values[i] = entry.numFails;
        }
    }

    const merged = Object.entries(lookup).map(([name, values]) => ({ name, values }))

    // sort descending by first value in subcluster, then second, and so on.
    merged.sort((a, b) => {
        for (let i = 0; i < a.values.length; i++) {
            const cmp = b.values[i] - a.values[i];
            if (cmp !== 0) {
                return cmp;
            }
        }
        return 0;
    });
    return merged;
}