// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, state } from 'lit-element';

// BugsTable lists the failure association rules configured in Weetbix.
@customElement('bugs-table')
export class BugsTable extends LitElement {
    @property()
    project = 'chromium';

    @state()
    rules: FailureAssociationRule[] | undefined;

    connectedCallback() {
        super.connectedCallback()
        fetch(`/api/projects/${encodeURIComponent(this.project)}/rules`).then(r => r.json()).then(rules => this.rules = rules);
    }

    render() {
        if (this.rules === undefined) {
            return html`Loading...`;
        }
        return html`
        <table>
            <thead>
                <tr>
                    <th>Bug</th>
                    <th>Rule Definition</th>
                    <th>Rule ID</th>
                    <th>Source Cluster ID</th>
                </tr>
            </thead>
            <tbody>
                ${this.rules.map(c => html`
                <tr>
                    <td>${c.bugId.system}/${c.bugId.id}</td>
                    <td>${c.ruleDefinition}</td>
                    <td>${c.ruleId}</td>
                    <td>${c.sourceCluster.algorithm}/${c.sourceCluster.id}</td>
                </tr>`)}
            </tbody>
        </table>
        `;
    }
}

// FailureAssociationRule is the failure association rule information sent by the server.
interface FailureAssociationRule {
    project: string;
    ruleId: string;
    ruleDefinition: string;
    bugId: BugId;
    sourceCluster: ClusterId;
}

interface BugId {
    system: string;
    id: string;
}

interface ClusterId {
    algorithm: string;
    id: string;
}
