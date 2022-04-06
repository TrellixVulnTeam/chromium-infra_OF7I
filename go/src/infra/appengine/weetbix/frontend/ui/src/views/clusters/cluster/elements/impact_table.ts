// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {
    customElement,
    html,
    LitElement,
    property
} from 'lit-element';
import { Ref } from 'react';

import { Cluster, Counts } from '../../../../services/cluster';

const metric = (counts: Counts): number => {
    return counts.nominal;
};

@customElement('impact-table')
export class ImpactTable extends LitElement{

    @property({ attribute: false })
    currentCluster!: Cluster;

    @property({ attribute: false })
    ref: Ref<ImpactTable> | null = null;

    render() {
        return html`
        <table data-testid="impact-table">
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
                    <td class="number">${metric(this.currentCluster.presubmitRejects1d)}</td>
                    <td class="number">${metric(this.currentCluster.presubmitRejects3d)}</td>
                    <td class="number">${metric(this.currentCluster.presubmitRejects7d)}</td>
                </tr>
                <tr>
                    <th>Test Runs Failed</th>
                    <td class="number">${metric(this.currentCluster.testRunFailures1d)}</td>
                    <td class="number">${metric(this.currentCluster.testRunFailures3d)}</td>
                    <td class="number">${metric(this.currentCluster.testRunFailures7d)}</td>
                </tr>
                <tr>
                    <th>Unexpected Failures</th>
                    <td class="number">${metric(this.currentCluster.failures1d)}</td>
                    <td class="number">${metric(this.currentCluster.failures3d)}</td>
                    <td class="number">${metric(this.currentCluster.failures7d)}</td>
                </tr>
            </tbody>
        </table>`;
    }
}