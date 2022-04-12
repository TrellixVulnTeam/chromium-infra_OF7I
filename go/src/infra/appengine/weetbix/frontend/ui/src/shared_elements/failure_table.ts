// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { LitElement, html, customElement, property, css, state, TemplateResult } from 'lit-element';
import { styleMap } from 'lit-html/directives/style-map';
import { DateTime } from 'luxon';
import '@material/mwc-button';
import '@material/mwc-icon';
import "@material/mwc-list/mwc-list-item";
import '@material/mwc-select';

// Indent of each level of grouping in the table in pixels.
const levelIndent = 10;

// FailureTable lists the failures in a cluster tracked by Weetbix.
@customElement('failure-table')
export class FailureTable extends LitElement {
    @property()
    project: string = '';

    @property()
    clusterAlgorithm: string = '';

    @property()
    clusterID: string = '';

    @state()
    failures: ClusterFailure[] | undefined;

    @state()
    groups: FailureGroup[] = [];

    @state()
    variants: FailureVariant[] = [];

    @state()
    failureFilter: FailureFilter = failureFilters[0];

    @state()
    impactFilter: ImpactFilter = impactFilters[1];

    @property()
    sortMetric: MetricName = 'latestFailureTime';

    @property({ type: Boolean })
    ascending: boolean = false;

    connectedCallback() {
        super.connectedCallback()

        fetch(`/api/projects/${encodeURIComponent(this.project)}/clusters/${encodeURIComponent(this.clusterAlgorithm)}/${encodeURIComponent(this.clusterID)}/failures`)
            .then(r => r.json())
            .then((failures: ClusterFailure[]) => {
                this.failures = failures
                this.countDistictVariantValues();
                this.groupCountAndSortFailures();
            });
    }

    countDistictVariantValues() {
        if (!this.failures) {
            return;
        }
        this.variants = [];
        this.failures.forEach(f => {
            f.variant.forEach(v => {
                if (!v.key) {
                    return;
                }
                const variant = this.variants.filter(e => e.key === v.key)?.[0];
                if (!variant) {
                    this.variants.push({ key: v.key, values: [v.value || ''], isSelected: false });
                } else {
                    if (variant.values.indexOf(v.value || '') === -1) {
                        variant.values.push(v.value || '');
                    }
                }
            });
        });
    }

    groupCountAndSortFailures() {
        if (this.failures) {
            let failures = this.failures;
            if (this.failureFilter == 'Presubmit Failures') {
                failures = failures.filter(f => f.presubmitRunId);
            } else if (this.failureFilter == 'Postsubmit Failures') {
                failures = failures.filter(f => !f.presubmitRunId);
            }
            this.groups = groupFailures(failures, f => {
                const variantValues = this.variants.filter(v => v.isSelected)
                    .map(v => f.variant.filter(fv => fv.key === v.key)?.[0]?.value || '');
                return [...variantValues, f.testId || ''];
            });
        }
        this.countAndSortFailures();
    }

    countAndSortFailures() {
        this.groups.forEach(group => {
            treeDistinctValues(group, failureIdsExtractor(), (g, values) => g.failures = values.size);
            treeDistinctValues(group, rejectedTestRunIdsExtractor(this.impactFilter), (g, values) => g.testRunFailures = values.size);
            treeDistinctValues(group, rejectedIngestedInvocationIdsExtractor(this.impactFilter), (g, values) => g.invocationFailures = values.size);
            treeDistinctValues(group, rejectedPresubmitRunIdsExtractor(this.impactFilter), (g, values) => g.presubmitRejects = values.size);
        });
        this.sortFailures();
    }

    sortFailures() {
        sortFailureGroups(this.groups, this.sortMetric, this.ascending);
        this.requestUpdate();
    }

    toggleSort(metric: MetricName) {
        if (metric === this.sortMetric) {
            this.ascending = !this.ascending;
        } else {
            this.sortMetric = metric;
            this.ascending = false;
        }
        this.sortFailures();
    }

    onImpactFilterChanged() {
        const item = this.shadowRoot!.querySelector('#impact-filter [selected]');
        if (item) {
            const selected = item.getAttribute('value');
            this.impactFilter = impactFilters.filter(f => f.name == selected)?.[0] || impactFilters[1];
        }
        this.countAndSortFailures();
    }

    onFailureFilterChanged() {
        const item = this.shadowRoot!.querySelector('#failure-filter [selected]');
        if (item) {
            this.failureFilter = (item.getAttribute('value') as FailureFilter) || failureFilters[0];
        }
        this.groupCountAndSortFailures();
    }

    toggleVariant(variant: FailureVariant) {
        const index = this.variants.indexOf(variant);
        this.variants.splice(index, 1);
        variant.isSelected = !variant.isSelected;
        const numSelected = this.variants.filter(v => v.isSelected).length;
        this.variants.splice(numSelected, 0, variant);
        this.groupCountAndSortFailures();
    }

    toggleExpand(group: FailureGroup) {
        group.isExpanded = !group.isExpanded;
        this.requestUpdate();
    }

    render() {
        const unselectedVariants = this.variants.filter(v => !v.isSelected).map(v => v.key);
        if (this.failures === undefined) {
            return html`Loading cluster failures...`;
        }
        const ungroupedVariants = (failure: ClusterFailure) => {
            return unselectedVariants.map(key => failure.variant.filter(v => v.key == key)?.[0]).filter(v => v);
        }
        const failureLink = (failure: ClusterFailure) => {
            const query = `ID:${failure.testId} `
            if (failure.ingestedInvocationId?.startsWith('build-')) {
                return `https://ci.chromium.org/ui/b/${failure.ingestedInvocationId.replace('build-', '')}/test-results?q=${encodeURIComponent(query)}`;
            }
            return `https://ci.chromium.org/ui/inv/${failure.ingestedInvocationId}/test-results?q=${encodeURIComponent(query)}`;
        }
        const clLink = (cl: Changelist) => {
            return `https://${cl.host}/c/${cl.change}/${cl.patchset}`;
        }
        const clName = (cl: Changelist) => {
            const host = cl.host.replace("-review.googlesource.com", "")
            return `${host}/${cl.change}/${cl.patchset}`;
        }
        const indentStyle = (level: number) => {
            return styleMap({ paddingLeft: (levelIndent * level) + 'px' });
        }
        const groupRow = (group: FailureGroup): TemplateResult => {
            return html`
            <tr>
                ${group.failure ?
                    html`<td style=${indentStyle(group.level)}>
                        <a href=${failureLink(group.failure)} target="_blank">${group.failure.ingestedInvocationId}</a>
                        ${group.failure.presubmitRunCl ? html`(<a href=${clLink(group.failure.presubmitRunCl)}>${clName(group.failure.presubmitRunCl)}</a>)` : html``}
                        <span class="variant-info">${ungroupedVariants(group.failure).map(v => `${v.key}: ${v.value}`).join(', ')}</span>
                    </td>` :
                    html`<td class="group" style=${indentStyle(group.level)} @click=${() => this.toggleExpand(group)}>
                        <mwc-icon>${group.isExpanded ? 'keyboard_arrow_down' : 'keyboard_arrow_right'}</mwc-icon>
                        ${group.name || 'none'}
                    </td>`}
                <td class="number">
                    ${group.failure ?
                    (group.failure.presubmitRunId ?
                        html`<a class="presubmit-link" href="https://luci-change-verifier.appspot.com/ui/run/${group.failure.presubmitRunId.id}" target="_blank">${group.presubmitRejects}</a>` :
                        '-')
                    : group.presubmitRejects}
                </td>
                <td class="number">${group.invocationFailures}</td>
                <td class="number">${group.testRunFailures}</td>
                <td class="number">${group.failures}</td>
                <td>${group.latestFailureTime.toRelative()}</td>
            </tr>
            ${group.isExpanded ? group.children.map(child => groupRow(child)) : null}`
        }
        const groupByButton = (variant: FailureVariant) => {
            return html`
                <mwc-button
                    label=${`${variant.key} (${variant.values.length})`}
                    ?unelevated=${variant.isSelected}
                    ?outlined=${!variant.isSelected}
                    @click=${() => this.toggleVariant(variant)}></mwc-button>`;
        }
        return html`
            <div class="controls">
                <div class="select-offset">
                    <mwc-select id="failure-filter" outlined label="Failure Type" @change=${() => this.onFailureFilterChanged()}>
                        ${failureFilters.map((filter) => html`<mwc-list-item ?selected=${filter == this.failureFilter} value="${filter}">${filter}</mwc-list-item>`)}
                    </mwc-select>
                </div>
                <div class="select-offset">
                    <mwc-select id="impact-filter" outlined label="Impact" @change=${() => this.onImpactFilterChanged()}>
                        ${impactFilters.map((filter) => html`<mwc-list-item ?selected=${filter == this.impactFilter} value="${filter.name}">${filter.name}</mwc-list-item>`)}
                    </mwc-select>
                </div>
                <div>
                    <div class="label">
                        Group By
                    </div>
                    ${this.variants.map(v => groupByButton(v))}
                </div>
            </div>
            <table data-testid="failures-table">
                <thead>
                    <tr>
                        <th></th>
                        <th class="sortable" @click=${() => this.toggleSort('presubmitRejects')}>
                            User Cls Failed Presubmit
                            ${this.sortMetric === 'presubmitRejects' ? html`<mwc-icon>${this.ascending ? 'expand_less' : 'expand_more'}</mwc-icon>` : null}
                        </th>
                        <th class="sortable" @click=${() => this.toggleSort('invocationFailures')}>
                            Builds Failed
                            ${this.sortMetric === 'invocationFailures' ? html`<mwc-icon>${this.ascending ? 'expand_less' : 'expand_more'}</mwc-icon>` : null}
                        </th>
                        <th class="sortable" @click=${() => this.toggleSort('testRunFailures')}>
                            Test Runs Failed
                            ${this.sortMetric === 'testRunFailures' ? html`<mwc-icon>${this.ascending ? 'expand_less' : 'expand_more'}</mwc-icon>` : null}
                        </th>
                        <th class="sortable" @click=${() => this.toggleSort('failures')}>
                            Unexpected Failures
                            ${this.sortMetric === 'failures' ? html`<mwc-icon>${this.ascending ? 'expand_less' : 'expand_more'}</mwc-icon>` : null}
                        </th>
                        <th class="sortable" @click=${() => this.toggleSort('latestFailureTime')}>
                            Latest Failure Time
                            ${this.sortMetric === 'latestFailureTime' ? html`<mwc-icon>${this.ascending ? 'expand_less' : 'expand_more'}</mwc-icon>` : null}
                        </th>
                    </tr>
                </thead>
                <tbody>
                    ${this.groups.map(group => groupRow(group))}
                </tbody>
            </table>
        `;
    }
    static styles = [css`
        .controls {
            display: flex;            
            gap: 30px;
        }
        .label {
            color: var(--greyed-out-text-color);
            font-size: var(--font-size-small);
        }
        .select-offset {
            padding-top: 7px
        }
        #impact-filter {
            width: 280px;
        }
        table {
            border-collapse: collapse;
            width: 100%;
            table-layout: fixed;
        }
        th {
            font-weight: normal;
            color: var(--greyed-out-text-color);
            font-size: var(--font-size-small);
            text-align: left;
        }
        td,th {
            padding: 4px;
            max-width: 80%;
        }
        td.number {
            text-align: right;
        }
        td.group {
            word-break: break-all;
        }
        th.sortable {
            cursor: pointer;
            width:120px;
        }
        tbody tr:hover {
            background-color: var(--light-active-color);
        }
        .group {
            cursor: pointer;
            --mdc-icon-size: var(--font-size-default);
        }
        .variant-info {
            color: var(--greyed-out-text-color);
            font-size: var(--font-size-small);
        }
        .presubmit-link {
            font-size: var(--font-size-small);
        }
    `];
}

// ImpactFilter represents what kind of impact should be counted or ignored in
// calculating impact for failures.
export interface ImpactFilter {
    name: string;
    ignoreWeetbixExoneration: boolean;
    ignoreAllExoneration: boolean;
    ignoreIngestedInvocationBlocked: boolean;
    ignoreTestRunBlocked: boolean;
}
export const impactFilters: ImpactFilter[] = [
    {
        name: 'Actual Impact',
        ignoreWeetbixExoneration: false,
        ignoreAllExoneration: false,
        ignoreIngestedInvocationBlocked: false,
        ignoreTestRunBlocked: false,
    }, {
        name: 'Without Weetbix Exoneration',
        ignoreWeetbixExoneration: true,
        ignoreAllExoneration: false,
        ignoreIngestedInvocationBlocked: false,
        ignoreTestRunBlocked: false,
    }, {
        name: 'Without All Exoneration',
        ignoreWeetbixExoneration: true,
        ignoreAllExoneration: true,
        ignoreIngestedInvocationBlocked: false,
        ignoreTestRunBlocked: false,
    }, {
        name: 'Without Retrying Test Runs',
        ignoreWeetbixExoneration: true,
        ignoreAllExoneration: true,
        ignoreIngestedInvocationBlocked: true,
        ignoreTestRunBlocked: false,
    }, {
        name: 'Without Any Retries',
        ignoreWeetbixExoneration: true,
        ignoreAllExoneration: true,
        ignoreIngestedInvocationBlocked: true,
        ignoreTestRunBlocked: true,
    }
];

const failureFilters = ['All Failures', 'Presubmit Failures', 'Postsubmit Failures'] as const;
type FailureFilter = typeof failureFilters[number];

// group a number of failures into a tree of failure groups.
// grouper is a function that returns a list of keys, one corresponding to each level of the grouping tree.
// impactFilter controls how metric counts are aggregated from failures into parent groups (see treeCounts and rejected... functions).
const groupFailures = (failures: ClusterFailure[], grouper: (f: ClusterFailure) => string[]): FailureGroup[] => {
    const topGroups: FailureGroup[] = [];
    failures.forEach(f => {
        const keys = grouper(f);
        let groups = topGroups;
        let failureTime = DateTime.fromISO(f.partitionTime || '');
        let level = 0;
        for (const key of keys) {
            const group = getOrCreateGroup(groups, key, failureTime);
            group.level = level;
            level += 1;
            groups = group.children;
        }
        const failureGroup = newGroup('', failureTime);
        failureGroup.failure = f;
        failureGroup.level = level;
        groups.push(failureGroup);
    });
    return topGroups;
}

// Create a new group.
const newGroup = (name: string, failureTime: DateTime): FailureGroup => {
    return {
        name: name,
        failures: 0,
        invocationFailures: 0,
        testRunFailures: 0,
        presubmitRejects: 0,
        children: [],
        isExpanded: false,
        latestFailureTime: failureTime,
        level: 0
    };
}

// Find a group by name in the given list of groups, create a new one and insert it if it is not found.
// failureTime is only used when creating a new group.
const getOrCreateGroup = (groups: FailureGroup[], name: string, failureTime: DateTime): FailureGroup => {
    let group = groups.filter(g => g.name == name)?.[0];
    if (group) {
        return group;
    }
    group = newGroup(name, failureTime);
    groups.push(group);
    return group;
}

// Returns the distinct values returned by featureExtractor for all children of the group.
// If featureExtractor returns undefined, the failure will be ignored.
// The distinct values for each group in the tree are also reported to `visitor` as the tree is traversed.
// A typical `visitor` function will store the count of distinct values in a property of the group.
const treeDistinctValues = (group: FailureGroup,
    featureExtractor: FeatureExtractor,
    visitor: (group: FailureGroup, distinctValues: Set<string>) => void): Set<string> => {
    const values: Set<string> = new Set();
    if (group.failure) {
        for (const value of featureExtractor(group.failure)) {
            values.add(value);
        }
    } else {
        for (const child of group.children) {
            for (let value of treeDistinctValues(child, featureExtractor, visitor)) {
                values.add(value);
            }
        }
    }
    visitor(group, values);
    return values;
}

// A FeatureExtractor returns a string representing some feature of a ClusterFailure.
// Returns undefined if there is no such feature for this failure.
type FeatureExtractor = (failure: ClusterFailure) => Set<string>;

// failureIdExtractor returns an extractor that returns a unique failure id for each failure.
// As failures don't actually have ids, it just returns an incrementing integer.
const failureIdsExtractor = (): FeatureExtractor => {
    let unique = 0;
    return f => {
        const values: Set<string> = new Set();
        for (let i = 0; i < f.count; i++) {
            unique += 1;
            values.add('' + unique);
        }
        return values;
    }
}

// Returns an extractor that returns the id of the test run that was rejected by this failure, if any.
// The impact filter is taken into account in determining if the run was rejected by this failure.
const rejectedTestRunIdsExtractor = (impactFilter: ImpactFilter): FeatureExtractor => {
    return f => {
        const values: Set<string> = new Set();
        if (!impactFilter.ignoreTestRunBlocked && !f.isTestRunBlocked) {
            return values;
        }
        for (const testRunId of f.testRunIds) {
            if (testRunId) {
                values.add(testRunId);
            }
        }
        return values;
    }
}

// Returns an extractor that returns the id of the ingested invocation that was rejected by this failure, if any.
// The impact filter is taken into account in determining if the invocation was rejected by this failure.
const rejectedIngestedInvocationIdsExtractor = (impactFilter: ImpactFilter): FeatureExtractor => {
    return f => {
        const values: Set<string> = new Set();
        if (f.exonerationStatus == 'WEETBIX' && 
                !(impactFilter.ignoreWeetbixExoneration || impactFilter.ignoreAllExoneration)) {
            return values;
        }
        if ((f.exonerationStatus == 'EXPLICIT' || f.exonerationStatus == 'IMPLICIT') && 
                !impactFilter.ignoreAllExoneration) {
            return values;
        }
        if (!f.isIngestedInvocationBlocked && !impactFilter.ignoreIngestedInvocationBlocked) {
            return values;
        }
        if (!impactFilter.ignoreTestRunBlocked && !f.isTestRunBlocked) {
            return values;
        }
        if (f.ingestedInvocationId) {
            values.add(f.ingestedInvocationId);
        }
        return values;
    }
}

// Returns an extractor that returns the identity of the CL that was rejected by this failure, if any.
// The impact filter is taken into account in determining if the CL was rejected by this failure.
const rejectedPresubmitRunIdsExtractor = (impactFilter: ImpactFilter): FeatureExtractor => {
    return f => {
        const values: Set<string> = new Set();
        if (f.exonerationStatus == 'WEETBIX' && 
                !(impactFilter.ignoreWeetbixExoneration || impactFilter.ignoreAllExoneration)) {
            return values;
        }
        if ((f.exonerationStatus == 'EXPLICIT' || f.exonerationStatus == 'IMPLICIT') && 
                !impactFilter.ignoreAllExoneration) {
            return values;
        }
        if (!f.isIngestedInvocationBlocked && !impactFilter.ignoreIngestedInvocationBlocked) {
            return values;
        }
        if (!impactFilter.ignoreTestRunBlocked && !f.isTestRunBlocked) {
            return values;
        }
        if (f.presubmitRunCl && f.presubmitRunOwner == 'user') {
            values.add(f.presubmitRunCl.host + '/' + f.presubmitRunCl.change.toFixed(0));
        }
        return values;
    }
}

// Sorts child failure groups at each node of the tree by the given metric.
const sortFailureGroups = (groups: FailureGroup[], metric: MetricName, ascending: boolean) => {
    const getMetric = (group: FailureGroup): number => {
        switch (metric) {
            case 'failures':
                return group.failures;
            case 'presubmitRejects':
                return group.presubmitRejects;
            case 'invocationFailures':
                return group.invocationFailures;
            case 'testRunFailures':
                return group.testRunFailures;
            case 'latestFailureTime':
                return group.latestFailureTime.toSeconds();
            default:
                throw new Error('unknown metric: ' + metric);
        }
    }
    groups.sort((a, b) => ascending ? (getMetric(a) - getMetric(b)) : (getMetric(b) - getMetric(a)));;
    for (const group of groups) {
        if (group.children) {
            sortFailureGroups(group.children, metric, ascending);
        }
    }
}

// The failure grouping code is complex, so export the parts for unit testing.
export const exportedForTesting = {
    groupFailures,
    impactFilters,
    rejectedIngestedInvocationIdsExtractor,
    rejectedPresubmitRunIdsExtractor,
    rejectedTestRunIdsExtractor,
    sortFailureGroups,
    treeDistinctCounts: treeDistinctValues,
}

// Test result was no exonerated.
type ExonerationStatus = 'NOT_EXONERATED' 
    // Test result was not recorded as exonerated, but the build
    // result was not FAILURE (e.g. SUCCESS, CANCELED, INFRA_FAILURE
    // instead), so the test result was not responsible for making
    // the build fail.
    | 'IMPLICIT' 
    // Test result was recorded as exonerated
    // for a reason other than Weetbix (or FindIt).
    | 'EXPLICIT' 
    // Test result was recorded as exonerated
    // based on Weetbix (or FindIt) data.
    | 'WEETBIX';

// ClusterFailure is the data returned by the server for each failure.
export interface ClusterFailure {
    realm: string | null;
    testId: string | null;
    variant: Variant[];
    presubmitRunCl: Changelist | null;
    presubmitRunId: PresubmitRunId | null;
    presubmitRunOwner: string | null;
    partitionTime: string | null;
    exonerationStatus: ExonerationStatus | null;
    ingestedInvocationId: string | null;
    isIngestedInvocationBlocked: boolean | null;
    testRunIds: Array<string | null>;
    isTestRunBlocked: boolean | null;
    count: number;
}

// Key/Value Variant pairs for failures.
interface Variant {
    key: string | null;
    value: string | null;
}

// Presubmit Run Ids of failures returned from the server.
interface PresubmitRunId {
    system: string | null;
    id: string | null;
}

// Changelist represents a gerrit patchset.
interface Changelist {
    host: string;
    change: number;
    patchset: number;
}

// Metrics that can be used for sorting FailureGroups.
// Each value is a property of FailureGroup.
type MetricName = 'presubmitRejects' | 'invocationFailures' | 'testRunFailures' | 'failures' | 'latestFailureTime';

// FailureGroups are nodes in the failure tree hierarchy.
export interface FailureGroup {
    name: string;
    presubmitRejects: number;
    invocationFailures: number;
    testRunFailures: number;
    failures: number;
    latestFailureTime: DateTime;
    level: number;
    children: FailureGroup[];
    isExpanded: boolean;
    failure?: ClusterFailure;
}

// FailureVariant represents variant keys that appear on at least one failure.
interface FailureVariant {
    key: string;
    values: string[];
    isSelected: boolean;
}