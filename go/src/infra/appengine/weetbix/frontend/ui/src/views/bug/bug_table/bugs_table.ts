// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, state, property, css } from 'lit-element';

import { getRulesService, ListRulesRequest, Rule } from '../../../services/rules';

// BugsTable lists the failure association rules configured in Weetbix.
@customElement('bugs-table')
export class BugsTable extends LitElement {

    @property()
    project!: string;

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
        this.rules = response.rules || [];
    }

    render() {
        if (this.rules === undefined) {
            return html`Loading...`;
        }
        if (this.rules.length === 0 ) {
            return html`
            <div class="empty">
                <h3>Nothing to see here...</h3>
                <p>No bugs are currently active for ${this.project}.</p>
            </div>`;
        }
        return html`
        <h1>Bugs in project ${this.project}</h1>
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
        th.sortable {
            cursor: pointer;
        }
        td, th {
            padding: 4px;
            max-width: 80%;
        }
        td.number {
            text-align: right;
        }
        td a.cluster-link {
            display: block;
            text-decoration: none;
            color: inherit;
        }
        tbody tr:hover {
            background-color: var(--light-active-color);
        }
        .bug a {
            font-size: var(--font-size-small);
        }
        .bug a:hover {
            text-decoration: underline;
        }
        .failure-reason {
            word-break: break-all;
            font-size: var(--font-size-small);
        }
        a[data-suggested] {
            font-style: italic;
        }
        .empty {
            margin: 50px auto;
            max-width: 600px;
            font-size: 24px;
        }
        `];
}
