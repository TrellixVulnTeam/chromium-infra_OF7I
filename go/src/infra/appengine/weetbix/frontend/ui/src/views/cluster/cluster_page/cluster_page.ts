// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import './elements/reclustering_progress_indicator';
import './elements/reclustering_progress_indicator.ts';
import '../../../shared_elements/failure_table';
import './elements/rule_section.ts';

import {
    css,
    customElement,
    html,
    LitElement,
    property,
    state,
    TemplateResult
} from 'lit-element';
import { Ref } from 'react';
import { NavigateFunction } from 'react-router-dom';

import { RuleChangedEvent } from './elements/rule_section';

// ClusterPage lists the clusters tracked by Weetbix.
@customElement('cluster-page')
export class ClusterPage extends LitElement {

    @property({ attribute: false })
    ref: Ref<ClusterPage> | null = null;

    @property()
    project: string = '';

    @property()
    clusterAlgorithm: string = '';

    @property()
    clusterId: string = '';

    navigate!: NavigateFunction;

    @state()
    cluster: Cluster | undefined;

    @state()
    // When the displayed rule's predicate (if any) was last updated.
    // This is provided to the reclustering progress indicator to show
    // the correct re-clustering status.
    rulePredicateLastUpdated: string = '';

    connectedCallback() {
        super.connectedCallback();
        this.rulePredicateLastUpdated = "";
        this.refreshAnalysis();
    }

    render() {
        const c = this.cluster;
        const metric = (counts: Counts): number => {
            return counts.nominal;
        }

        var definitionSection = html`Loading...`;
        if (this.clusterAlgorithm.startsWith("rules-")) {
            definitionSection = html`
                <rule-section project=${this.project} ruleId=${this.clusterId} @rulechanged=${this.onRuleChanged}>
                </rule-section>
            `;
        } else if (c !== undefined) {
            var criteriaName = ""
            if (this.clusterAlgorithm.startsWith("testname-")) {
                criteriaName = "Test name-based clustering";
            } else if (this.clusterAlgorithm.startsWith("reason-")) {
                criteriaName = "Failure reason-based clustering";
            }
            let newRuleButton: TemplateResult = html``
            if (c.failureAssociationRule) {
                newRuleButton = html`<mwc-button class="new-rule-button" raised @click=${this.newRuleClicked}>New Rule from Cluster</mwc-button>`;
            }

            definitionSection = html`
            <h1>Cluster <span class="cluster-id">${this.clusterAlgorithm}/${this.clusterId}</span></h1>
            <div class="definition-box-container">
                <pre class="definition-box">${c.title}</pre>
            </div>
            <table class="definition-table">
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
            ${newRuleButton}
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
                        <th>User Cls Failed Presubmit</th>
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

        return html`
        <reclustering-progress-indicator project=${this.project} ?hasrule=${this.clusterAlgorithm.startsWith("rules-")}
            rulePredicateLastUpdated=${this.rulePredicateLastUpdated} @refreshanalysis=${this.refreshAnalysis}>
        </reclustering-progress-indicator>
        <div id="container">
            ${definitionSection}
            <h2>Impact</h2>
            ${impactTable}
            <h2>Recent Failures</h2>
            <failure-table project=${this.project} clusterAlgorithm=${this.clusterAlgorithm} clusterID=${this.clusterId}>
            </failure-table>
        </div>
        `;
    }

    newRuleClicked() {
        if (!this.cluster) {
            throw new Error('invariant violated: newRuleClicked cannot be called before cluster is loaded');
        }
        const projectEncoded = encodeURIComponent(this.project);
        const ruleEncoded = encodeURIComponent(this.cluster.failureAssociationRule);
        const sourceAlgEncoded = encodeURIComponent(this.clusterAlgorithm);
        const sourceIdEncoded = encodeURIComponent(this.clusterId);

        const newRuleURL = `/p/${projectEncoded}/rules/new?rule=${ruleEncoded}&sourceAlg=${sourceAlgEncoded}&sourceId=${sourceIdEncoded}`;
        this.navigate(newRuleURL);
    }

    // Called when the rule displayed in the rule section is loaded
    // for the first time, or updated.
    onRuleChanged(e: CustomEvent<RuleChangedEvent>) {
        this.rulePredicateLastUpdated = e.detail.predicateLastUpdated;
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
        .new-rule-button {
            margin-top: 10px;
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
    `];
}

// Cluster is the cluster information sent by the server.
interface Cluster {
    clusterId: ClusterId;
    title: string;
    failureAssociationRule: string;
    presubmitRejects1d: Counts;
    presubmitRejects3d: Counts;
    presubmitRejects7d: Counts;
    testRunFailures1d: Counts;
    testRunFailures3d: Counts;
    testRunFailures7d: Counts;
    failures1d: Counts;
    failures3d: Counts;
    failures7d: Counts;
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
