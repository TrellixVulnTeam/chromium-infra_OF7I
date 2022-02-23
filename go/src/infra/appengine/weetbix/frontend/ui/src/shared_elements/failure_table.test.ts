// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { assert } from 'chai';
import { DateTime } from 'luxon/src/luxon';

import { ClusterFailure, ImpactFilter, exportedForTesting, FailureGroup } from './failure_table';

const { impactFilters, rejectedIngestedInvocationIdsExtractor, rejectedPresubmitRunIdsExtractor, rejectedTestRunIdsExtractor, groupFailures, treeDistinctCounts, sortFailureGroups } = exportedForTesting;

describe('Extractors', () => {
    it('should return ids in only the cases expected by failure type and impact filter.', () => {
        interface extractorTestCase {
            failure: ClusterFailure;
            filter: String;
            shouldExtractTestRunId: Boolean;
            shouldExtractIngestedInvocationId: Boolean;
        }

        const testCases: extractorTestCase[] = [{
            failure: newFailure().build(),
            filter: 'Actual Impact',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().testRunBlocked().build(),
            filter: 'Actual Impact',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().ingestedInvocationBlocked().build(),
            filter: 'Actual Impact',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().exonerate().build(),
            filter: 'Actual Impact',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().testRunBlocked().exonerate().build(),
            filter: 'Actual Impact',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().ingestedInvocationBlocked().exonerate().build(),
            filter: 'Actual Impact',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().build(),
            filter: 'Without Exoneration',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().testRunBlocked().build(),
            filter: 'Without Exoneration',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().ingestedInvocationBlocked().build(),
            filter: 'Without Exoneration',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().exonerate().build(),
            filter: 'Without Exoneration',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().testRunBlocked().exonerate().build(),
            filter: 'Without Exoneration',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().ingestedInvocationBlocked().exonerate().build(),
            filter: 'Without Exoneration',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().build(),
            filter: 'Without Retrying Test Runs',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().testRunBlocked().build(),
            filter: 'Without Retrying Test Runs',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().ingestedInvocationBlocked().build(),
            filter: 'Without Retrying Test Runs',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().exonerate().build(),
            filter: 'Without Retrying Test Runs',
            shouldExtractTestRunId: false,
            shouldExtractIngestedInvocationId: false,
        }, {
            failure: newFailure().testRunBlocked().exonerate().build(),
            filter: 'Without Retrying Test Runs',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().ingestedInvocationBlocked().exonerate().build(),
            filter: 'Without Retrying Test Runs',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().build(),
            filter: 'Without Any Retries',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().testRunBlocked().build(),
            filter: 'Without Any Retries',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().ingestedInvocationBlocked().build(),
            filter: 'Without Any Retries',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().exonerate().build(),
            filter: 'Without Any Retries',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().testRunBlocked().exonerate().build(),
            filter: 'Without Any Retries',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }, {
            failure: newFailure().ingestedInvocationBlocked().exonerate().build(),
            filter: 'Without Any Retries',
            shouldExtractTestRunId: true,
            shouldExtractIngestedInvocationId: true,
        }];
        for (const tc of testCases) {
            const testRunIds = rejectedTestRunIdsExtractor(impactFilterNamed(tc.filter))(tc.failure);
            if (tc.shouldExtractTestRunId) {
                assert.isNotEmpty(testRunIds, `failed to extract testRunId with filter ${tc.filter} on failure ${JSON.stringify(tc.failure)}`);
            } else {
                assert.isEmpty(testRunIds, `unexpectedly extracted testRunId with filter ${tc.filter} on failure ${JSON.stringify(tc.failure)}`);
            }
            const ingestedInvocationIds = rejectedIngestedInvocationIdsExtractor(impactFilterNamed(tc.filter))(tc.failure);
            if (tc.shouldExtractIngestedInvocationId) {
                assert.isNotEmpty(ingestedInvocationIds, `failed to extract ingestedInvocationId with filter ${tc.filter} on failure ${JSON.stringify(tc.failure)}`);
            } else {
                assert.isEmpty(ingestedInvocationIds, `unexpectedly extracted ingestedInvocationId with filter ${tc.filter} on failure ${JSON.stringify(tc.failure)}`);
            }
            const presubmitRunIds = rejectedPresubmitRunIdsExtractor(impactFilterNamed(tc.filter))(tc.failure);
            // presubmitRunId is extracted under exactly the same conditions as ingestedInvocationId.
            if (tc.shouldExtractIngestedInvocationId) {
                assert.isNotEmpty(presubmitRunIds, `failed to extract presubmitRunId with filter ${tc.filter} on failure ${JSON.stringify(tc.failure)}`);
            } else {
                assert.isEmpty(presubmitRunIds, `unexpectedly extracted presubmitRunId with filter ${tc.filter} on failure ${JSON.stringify(tc.failure)}`);
            }
        }
    });
})

describe('groupFailures', () => {
    it('should put each failure in a separate group when given unique grouping keys', () => {
        const failures = [
            newFailure().build(),
            newFailure().build(),
            newFailure().build(),
        ];
        let unique = 0;
        const groups: FailureGroup[] = groupFailures(failures, () => ['' + unique++])
        assert.lengthOf(groups, 3);
        assert.lengthOf(groups[0].children, 1);
    });
    it('should put each failure in a single group when given a single grouping key', () => {
        const failures = [
            newFailure().build(),
            newFailure().build(),
            newFailure().build(),
        ];
        const groups: FailureGroup[] = groupFailures(failures, () => ['group1'])
        assert.lengthOf(groups, 1);
        assert.lengthOf(groups[0].children, 3);
    });
    it('should put group failures into multiple levels', () => {
        const failures = [
            newFailure().withVariant('v1', 'a').withVariant('v2', 'a').build(),
            newFailure().withVariant('v1', 'a').withVariant('v2', 'b').build(),
            newFailure().withVariant('v1', 'b').withVariant('v2', 'a').build(),
            newFailure().withVariant('v1', 'b').withVariant('v2', 'b').build(),
        ];
        const groups: FailureGroup[] = groupFailures(failures, f => f.variant.map(v => v.value || ''))
        assert.lengthOf(groups, 2);
        assert.lengthOf(groups[0].children, 2);
        assert.lengthOf(groups[1].children, 2);
        assert.lengthOf(groups[0].children[0].children, 1);
    });
});

describe('treeDistinctCounts', () => {
    // A helper to just store the counts to the failures field.
    const setFailures = (g: FailureGroup, values: Set<string>) => {
        g.failures = values.size;
    }
    it('should have count of 1 for a valid feature', () => {
        const groups = groupFailures([newFailure().build()], () => ['group']);

        treeDistinctCounts(groups[0], () => new Set(['a']), setFailures);

        assert.equal(groups[0].failures, 1);
    });
    it('should have count of 0 for an invalid feature', () => {
        const groups = groupFailures([newFailure().build()], () => ['group']);

        treeDistinctCounts(groups[0], () => new Set(), setFailures);

        assert.equal(groups[0].failures, 0);
    });
    it('should have count of 1 for two identical features', () => {
        const groups = groupFailures([
            newFailure().build(),
            newFailure().build(),
        ], () => ['group']);

        treeDistinctCounts(groups[0], () => new Set(['a']), setFailures);

        assert.equal(groups[0].failures, 1);
    });
    it('should have count of 2 for two different features', () => {
        const groups = groupFailures([
            newFailure().withTestId('a').build(),
            newFailure().withTestId('b').build(),
        ], () => ['group']);

        treeDistinctCounts(groups[0], f => f.testId ? new Set([f.testId]) : new Set(), setFailures);

        assert.equal(groups[0].failures, 2);
    });
    it('should have count of 1 for two identical features in different subgroups', () => {
        const groups = groupFailures([
            newFailure().withTestId('a').withVariant('group', 'a').build(),
            newFailure().withTestId('a').withVariant('group', 'b').build(),
        ], f => ['top', ...f.variant.map((v) => v.value || '')]);

        treeDistinctCounts(groups[0], f => f.testId ? new Set([f.testId]) : new Set(), setFailures);

        assert.equal(groups[0].failures, 1);
        assert.equal(groups[0].children[0].failures, 1);
        assert.equal(groups[0].children[1].failures, 1);
    });
    it('should have count of 2 for two different features in different subgroups', () => {
        const groups = groupFailures([
            newFailure().withTestId('a').withVariant('group', 'a').build(),
            newFailure().withTestId('b').withVariant('group', 'b').build(),
        ], f => ['top', ...f.variant.map((v) => v.value || '')]);

        treeDistinctCounts(groups[0], f => f.testId ? new Set([f.testId]) : new Set(), setFailures);

        assert.equal(groups[0].failures, 2);
        assert.equal(groups[0].children[0].failures, 1);
        assert.equal(groups[0].children[1].failures, 1);
    });
});


describe('sortFailureGroups', () => {
    it('sorts top level groups ascending', () => {
        const groups: FailureGroup[] = [
            newGroup('c').withFailures(3).build(),
            newGroup('a').withFailures(1).build(),
            newGroup('b').withFailures(2).build(),
        ];

        sortFailureGroups(groups, 'failures', true);

        assert.deepEqual(groups.map(g => g.name), ['a', 'b', 'c'])
    });
    it('sorts top level groups descending', () => {
        const groups: FailureGroup[] = [
            newGroup('c').withFailures(3).build(),
            newGroup('a').withFailures(1).build(),
            newGroup('b').withFailures(2).build(),
        ];

        sortFailureGroups(groups, 'failures', false);

        assert.deepEqual(groups.map(g => g.name), ['c', 'b', 'a'])
    });
    it('sorts child groups', () => {
        const groups: FailureGroup[] = [
            newGroup('c').withFailures(3).build(),
            newGroup('a').withFailures(1).withChildren([
                newGroup('a3').withFailures(3).build(),
                newGroup('a2').withFailures(2).build(),
                newGroup('a1').withFailures(1).build(),
            ]).build(),
            newGroup('b').withFailures(2).build(),
        ];

        sortFailureGroups(groups, 'failures', true);

        assert.deepEqual(groups.map(g => g.name), ['a', 'b', 'c']);
        assert.deepEqual(groups[0].children.map(g => g.name), ['a1', 'a2', 'a3']);
    });
    it('sorts on an alternate metric', () => {
        const groups: FailureGroup[] = [
            newGroup('c').withPresubmitRejects(3).build(),
            newGroup('a').withPresubmitRejects(1).build(),
            newGroup('b').withPresubmitRejects(2).build(),
        ];

        sortFailureGroups(groups, 'presubmitRejects', true);

        assert.deepEqual(groups.map(g => g.name), ['a', 'b', 'c'])
    });
});

// Helper functions.
const impactFilterNamed = (name: String) => {
    return impactFilters.filter((f: ImpactFilter) => name == f.name)?.[0];
}

const newFailure = (): ClusterFailureBuilder => {
    return new ClusterFailureBuilder();
}

class ClusterFailureBuilder {
    failure: ClusterFailure;
    constructor() {
        this.failure = {
            realm: 'testproject/testrealm',
            testId: 'ninja://dir/test.param',
            variant: [],
            presubmitRunCl: { host: 'clproject-review.googlesource.com', change: 123456, patchset: 7 },
            presubmitRunId: { system: 'cv', id: 'presubmitRunId' },
            presubmitRunOwner: 'user',
            partitionTime: '2021-05-12T19:05:34',
            isExonerated: false,
            ingestedInvocationId: 'ingestedInvocationId',
            isIngestedInvocationBlocked: false,
            testRunIds: ['testRunId'],
            isTestRunBlocked: false,
            count: 1,
        };
    }
    build(): ClusterFailure {
        return this.failure;
    }
    testRunBlocked() {
        this.failure.isTestRunBlocked = true;
        return this;
    }
    ingestedInvocationBlocked() {
        this.failure.isIngestedInvocationBlocked = true;
        this.failure.isTestRunBlocked = true;
        return this;
    }
    exonerate() {
        this.failure.isExonerated = true;
        return this;
    }
    withVariant(key: string, value: string) {
        this.failure.variant.push({ key, value });
        return this;
    }
    withTestId(id: string) {
        this.failure.testId = id;
        return this;
    }
}

const newGroup = (name: string): FailureGroupBuilder => {
    return new FailureGroupBuilder(name);
}

class FailureGroupBuilder {
    failureGroup: FailureGroup
    constructor(name: string) {
        this.failureGroup = {
            name,
            children: [],
            failures: 0,
            testRunFailures: 0,
            invocationFailures: 0,
            presubmitRejects: 0,
            latestFailureTime: DateTime.now(),
            isExpanded: false,
            level: 0,
        }
    }

    build(): FailureGroup {
        return this.failureGroup;
    }

    withFailures(failures: number) {
        this.failureGroup.failures = failures;
        return this;
    }

    withPresubmitRejects(presubmitRejects: number) {
        this.failureGroup.presubmitRejects = presubmitRejects;
        return this;
    }

    withChildren(children: FailureGroup[]) {
        this.failureGroup.children = children;
        return this;
    }
}