// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, state } from 'lit-element';

import { getRulesService, ListRulesRequest, Rule } from '../../../services/rules';

// BugsTable lists the failure association rules configured in Weetbix.
@customElement('bugs-table')
export class BugsTable extends LitElement {
    @property()
    project = 'chromium';

    @state()
    rules: Rule[] | undefined;

    connectedCallback() {
        super.connectedCallback();
        this.fetch();
    }

    async fetch() {
        const service = getRulesService();
        const request: ListRulesRequest = {
            parent: `projects/${this.project}`,
        }
        const response = await service.list(request);
        this.rules = response.rules;
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
                    <td>${c.bug.system}/${c.bug.id}</td>
                    <td>${c.ruleDefinition}</td>
                    <td>${c.ruleId}</td>
                    <td>${c.sourceCluster.algorithm}/${c.sourceCluster.id}</td>
                </tr>`)}
            </tbody>
        </table>
        `;
    }
}
