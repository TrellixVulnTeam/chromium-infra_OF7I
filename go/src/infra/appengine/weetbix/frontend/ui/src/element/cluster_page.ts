// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state } from 'lit-element';
import { BeforeEnterObserver, RouterLocation } from '@vaadin/router';
import './rule_section.ts';
import { RuleChangedEvent } from './rule_section';
import './reclustering_progress_indicator.ts';

// ClusterPage lists the clusters tracked by Weetbix.
@customElement('cluster-page')
export class ClusterPage extends LitElement implements BeforeEnterObserver {
    @property({ attribute: false })
    location!: RouterLocation;

    @property()
    project: string = '';

    @property()
    clusterAlgorithm: string = '';

    @property()
    clusterId: string = '';

    @state()
    cluster: Cluster | undefined;

    @state()
    // When the displayed rule (if any) was last updated. This is provided to
    // the reclustering progress indicator to show the correct re-clustering
    // status.
    ruleLastUpdated: string = '';

    onBeforeEnter(location: RouterLocation) {
        // Take the first parameter value only.
        this.project = typeof location.params.project == 'string' ? location.params.project : location.params.project[0];
        this.clusterAlgorithm = typeof location.params.algorithm == 'string' ? location.params.algorithm : location.params.algorithm[0];
        this.clusterId = typeof location.params.id == 'string' ? location.params.id : location.params.id[0];
    }

    connectedCallback() {
        super.connectedCallback();

        this.ruleLastUpdated = "";
        this.refreshAnalysis();
    }

    render() {
        const c = this.cluster;
        const clusterCriteriaValue = (cluster: Cluster): string => {
            if (cluster.clusterId.algorithm.startsWith("testname-")) {
                return cluster.exampleTestId;
            } else if (cluster.clusterId.algorithm.startsWith("failurereason-")) {
                return cluster.exampleFailureReason;
            }
            return `${cluster.clusterId.algorithm}/${cluster.clusterId.id}`;
        }
        const metric = (counts: Counts): number => {
            return counts.nominal;
        }

        var definitionSection = html`Loading...`;
        if (this.clusterAlgorithm.startsWith("rules-")) {
            definitionSection = html`
                <rule-section
                    project=${this.project}
                    ruleId=${this.clusterId}
                    @rulechanged=${this.ruleChanged}>
                </rule-section>
            `;
        } else if (c !== undefined) {
            var criteriaName = ""
            if (this.clusterAlgorithm.startsWith("testname-")) {
                criteriaName = "Test name-based clustering";
            } else if (this.clusterAlgorithm.startsWith("failurereason-")) {
                criteriaName = "Failure reason-based clustering";
            }

            definitionSection = html`
            <div class="definition-box-container">
                <pre class="definition-box">${clusterCriteriaValue(c)}</pre>
            </div>
            <table>
                <tbody>
                    <tr>
                        <th>Type</th>
                        <td>Suggested</td>
                    </tr>
                    <tr>
                        <th>Algorithm</th>
                        <td>${criteriaName}</td>
                    </tr>
                </tbody>
            </table>
            `
        }

        var impactTable = html`Loading...`;
        if (c !== undefined) {
            impactTable = html`
            <table>
                <thead>
                    <tr>
                        <th></th>
                        <th>1 day</th>
                        <th>3 days</th>
                        <th>7 days</th>
                    </tr>
                </thead>
                <tbody class="data">
                    <tr>
                        <th>Presubmit Runs Failed</th>
                        <td class="number">${metric(c.presubmitRejects1d)}</td>
                        <td class="number">${metric(c.presubmitRejects3d)}</td>
                        <td class="number">${metric(c.presubmitRejects7d)}</td>
                    </tr>
                    <tr>
                        <th>Test Runs Failed</th>
                        <td class="number">${metric(c.testRunFailures1d)}</td>
                        <td class="number">${metric(c.testRunFailures3d)}</td>
                        <td class="number">${metric(c.testRunFailures7d)}</td>
                    </tr>
                    <tr>
                        <th>Unexpected Failures</th>
                        <td class="number">${metric(c.failures1d)}</td>
                        <td class="number">${metric(c.failures3d)}</td>
                        <td class="number">${metric(c.failures7d)}</td>
                    </tr>
                </tbody>
            </table>`;
        }

        var breakdownTable = html`Loading...`;
        if (c !== undefined) {
            const merged = mergeSubClusters([c.affectedTests1d, c.affectedTests3d, c.affectedTests7d]);
            breakdownTable = html`
            <table>
                <thead>
                    <tr>
                        <th>Test</th>
                        <th>1 day</th>
                        <th>3 days</th>
                        <th>7 days</th>
                    </tr>
                </thead>
                <tbody class="data">
                    ${merged.map(entry => html`
                    <tr>
                        <td class="test-id">${entry.name}</td>
                        ${entry.values.map(v => html`<td class="number">${v}</td>`)}
                    </tr>`)}
                </tbody>
            </table>`;
        }

        return html`
        <reclustering-progress-indicator
            project=${this.project}
            ?hasrule=${this.clusterAlgorithm.startsWith("rules-")}
            ruleLastUpdated=${this.ruleLastUpdated}
            @refreshanalysis=${this.refreshAnalysis}>
        </reclustering-progress-indicator>
        <div id="container">
            <h1>Cluster <span class="cluster-id">${this.clusterAlgorithm}/${this.clusterId}</span></h1>
            ${definitionSection}
            <h2>Impact</h2>
            ${impactTable}
            <h2>Breakdown</h2>
            ${breakdownTable}
        </div>
        `;
    }

    // Called when the rule displayed in the rule section is loaded
    // for the first time, or updated.
    ruleChanged(e: CustomEvent<RuleChangedEvent>) {
        this.ruleLastUpdated = e.detail.lastUpdated;
    }

    // (Re-)loads cluster impact analysis. Called on page load or
    // if the refresh button on the reclustering progress indicator
    // is clicked at completion of re-clustering.
    async refreshAnalysis() {
        this.cluster = undefined;

        const response = await fetch(`/api/projects/${encodeURIComponent(this.project)}/clusters/${encodeURIComponent(this.clusterAlgorithm)}/${encodeURIComponent(this.clusterId)}`);
        const cluster = await response.json();
        this.cluster = cluster;
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
        .definition-box-container {
            margin-bottom: 20px;
        }
        .definition-box {
            border: solid 1px var(--divider-color);
            background-color: var(--block-background-color);
            padding: 20px 14px;
            margin: 0px;
            display: inline-block;
            white-space: pre-wrap;
            overflow-wrap: anywhere;
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
        tbody.data tr:hover {
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
    clusterId: ClusterId;
    presubmitRejects1d: Counts;
    presubmitRejects3d: Counts;
    presubmitRejects7d: Counts;
    testRunFailures1d: Counts;
    testRunFailures3d: Counts;
    testRunFailures7d: Counts;
    failures1d: Counts;
    failures3d: Counts;
    failures7d: Counts;
    affectedTests1d: SubCluster[] | null;
    affectedTests3d: SubCluster[] | null;
    affectedTests7d: SubCluster[] | null;
    exampleFailureReason: string;
    exampleTestId: string;
}

interface Counts {
    nominal: number;
    preExoneration: number;
    residual: number;
    residualPreExoneration: number;
}

interface ClusterId {
    algorithm: string;
    id: string;
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
const mergeSubClusters = (subClusters: Array<SubCluster[] | null>): MergedSubClusters[] => {
    const lookup: { [name: string]: number[] } = {};
    for (let i = 0; i < subClusters.length; i++) {
        const clusters = subClusters[i];
        for (const entry of clusters || []) {
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