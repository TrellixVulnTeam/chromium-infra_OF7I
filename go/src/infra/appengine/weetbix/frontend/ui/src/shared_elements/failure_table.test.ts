// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { DateTime } from 'luxon';

import {
    ClusterFailure,
    exportedForTesting,
    FailureGroup,
    ImpactFilter
} from './failure_table';

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
            exonerationStatus: 'NOT_EXONERATED',
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
    exonerateWeetbix() {
        this.failure.exonerationStatus = "WEETBIX";
        return this;
    }
    exonerateExplicitly() {
        // Explicitly exonerated by something other than Weetbix.
        this.failure.exonerationStatus = "EXPLICIT";
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

// Helper functions.
const impactFilterNamed = (name: String) => {
    return impactFilters.filter((f: ImpactFilter) => name == f.name)?.[0];
}

const newFailure = (): ClusterFailureBuilder => {
    return new ClusterFailureBuilder();
}

const {
    impactFilters,
    rejectedIngestedInvocationIdsExtractor,
    rejectedPresubmitRunIdsExtractor,
    rejectedTestRunIdsExtractor,
    groupFailures,
    treeDistinctCounts,
    sortFailureGroups
} = exportedForTesting;

interface ExtractorTestCase {
    failure: ClusterFailure;
    filter: String;
    shouldExtractTestRunId: Boolean;
    shouldExtractIngestedInvocationId: Boolean;
}

describe.each<ExtractorTestCase>([{
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
    failure: newFailure().exonerateWeetbix().build(),
    filter: 'Actual Impact',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().testRunBlocked().exonerateWeetbix().build(),
    filter: 'Actual Impact',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().ingestedInvocationBlocked().exonerateWeetbix().build(),
    filter: 'Actual Impact',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().exonerateExplicitly().build(),
    filter: 'Actual Impact',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().testRunBlocked().exonerateExplicitly().build(),
    filter: 'Actual Impact',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().ingestedInvocationBlocked().exonerateExplicitly().build(),
    filter: 'Actual Impact',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: false,
},{
    failure: newFailure().build(),
    filter: 'Without Weetbix Exoneration',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().ingestedInvocationBlocked().build(),
    filter: 'Without Weetbix Exoneration',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}, {
    failure: newFailure().exonerateWeetbix().build(),
    filter: 'Without Weetbix Exoneration',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().ingestedInvocationBlocked().exonerateWeetbix().build(),
    filter: 'Without Weetbix Exoneration',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}, {
    failure: newFailure().exonerateExplicitly().build(),
    filter: 'Without Weetbix Exoneration',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().ingestedInvocationBlocked().exonerateExplicitly().build(),
    filter: 'Without Weetbix Exoneration',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().build(),
    filter: 'Without All Exoneration',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
},  {
    failure: newFailure().ingestedInvocationBlocked().build(),
    filter: 'Without All Exoneration',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}, {
    failure: newFailure().exonerateWeetbix().build(),
    filter: 'Without All Exoneration',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
},  {
    failure: newFailure().ingestedInvocationBlocked().exonerateWeetbix().build(),
    filter: 'Without All Exoneration',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}, {
    failure: newFailure().exonerateExplicitly().build(),
    filter: 'Without All Exoneration',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().ingestedInvocationBlocked().exonerateExplicitly().build(),
    filter: 'Without All Exoneration',
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
    failure: newFailure().exonerateWeetbix().build(),
    filter: 'Without Retrying Test Runs',
    shouldExtractTestRunId: false,
    shouldExtractIngestedInvocationId: false,
}, {
    failure: newFailure().testRunBlocked().exonerateWeetbix().build(),
    filter: 'Without Retrying Test Runs',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}, {
    failure: newFailure().ingestedInvocationBlocked().exonerateWeetbix().build(),
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
    failure: newFailure().exonerateWeetbix().build(),
    filter: 'Without Any Retries',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}, {
    failure: newFailure().testRunBlocked().exonerateWeetbix().build(),
    filter: 'Without Any Retries',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}, {
    failure: newFailure().ingestedInvocationBlocked().exonerateWeetbix().build(),
    filter: 'Without Any Retries',
    shouldExtractTestRunId: true,
    shouldExtractIngestedInvocationId: true,
}])('Extractors with %j', (tc: ExtractorTestCase) => {
    it('should return ids in only the cases expected by failure type and impact filter.', () => {
        const testRunIds = rejectedTestRunIdsExtractor(impactFilterNamed(tc.filter))(tc.failure);
        if (tc.shouldExtractTestRunId) {
            expect(testRunIds.size).toBeGreaterThan(0);
        } else {
            expect(testRunIds.size).toBe(0);
        }
        const ingestedInvocationIds = rejectedIngestedInvocationIdsExtractor(impactFilterNamed(tc.filter))(tc.failure);
        if (tc.shouldExtractIngestedInvocationId) {
            expect(ingestedInvocationIds.size).toBeGreaterThan(0);
        } else {
            expect(ingestedInvocationIds.size).toBe(0);
        }
        const presubmitRunIds = rejectedPresubmitRunIdsExtractor(impactFilterNamed(tc.filter))(tc.failure);
        // presubmitRunId is extracted under exactly the same conditions as ingestedInvocationId.
        if (tc.shouldExtractIngestedInvocationId) {
            expect(presubmitRunIds.size).toBeGreaterThan(0);
        } else {
            expect(presubmitRunIds.size).toBe(0);
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
        expect(groups.length).toBe(3);
        expect(groups[0].children.length).toBe(1);
    });
    it('should put each failure in a single group when given a single grouping key', () => {
        const failures = [
            newFailure().build(),
            newFailure().build(),
            newFailure().build(),
        ];
        const groups: FailureGroup[] = groupFailures(failures, () => ['group1'])
        expect(groups.length).toBe(1);
        expect(groups[0].children.length).toBe (3);
    });
    it('should put group failures into multiple levels', () => {
        const failures = [
            newFailure().withVariant('v1', 'a').withVariant('v2', 'a').build(),
            newFailure().withVariant('v1', 'a').withVariant('v2', 'b').build(),
            newFailure().withVariant('v1', 'b').withVariant('v2', 'a').build(),
            newFailure().withVariant('v1', 'b').withVariant('v2', 'b').build(),
        ];
        const groups: FailureGroup[] = groupFailures(failures, f => f.variant.map(v => v.value || ''))
        expect(groups.length).toBe(2);
        expect(groups[0].children.length).toBe(2);
        expect(groups[1].children.length).toBe(2);
        expect(groups[0].children[0].children.length).toBe(1);
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

        expect(groups[0].failures).toBe(1);
    });
    it('should have count of 0 for an invalid feature', () => {
        const groups = groupFailures([newFailure().build()], () => ['group']);

        treeDistinctCounts(groups[0], () => new Set(), setFailures);

        expect(groups[0].failures).toBe(0);
    });

    it('should have count of 1 for two identical features', () => {
        const groups = groupFailures([
            newFailure().build(),
            newFailure().build(),
        ], () => ['group']);

        treeDistinctCounts(groups[0], () => new Set(['a']), setFailures);

        expect(groups[0].failures).toBe(1);
    });
    it('should have count of 2 for two different features', () => {
        const groups = groupFailures([
            newFailure().withTestId('a').build(),
            newFailure().withTestId('b').build(),
        ], () => ['group']);

        treeDistinctCounts(groups[0], f => f.testId ? new Set([f.testId]) : new Set(), setFailures);

        expect(groups[0].failures).toBe(2);
    });
    it('should have count of 1 for two identical features in different subgroups', () => {
        const groups = groupFailures([
            newFailure().withTestId('a').withVariant('group', 'a').build(),
            newFailure().withTestId('a').withVariant('group', 'b').build(),
        ], f => ['top', ...f.variant.map((v) => v.value || '')]);

        treeDistinctCounts(groups[0], f => f.testId ? new Set([f.testId]) : new Set(), setFailures);

        expect(groups[0].failures).toBe(1);
        expect(groups[0].children[0].failures).toBe(1);
        expect(groups[0].children[1].failures).toBe(1);
    });
    it('should have count of 2 for two different features in different subgroups', () => {
        const groups = groupFailures([
            newFailure().withTestId('a').withVariant('group', 'a').build(),
            newFailure().withTestId('b').withVariant('group', 'b').build(),
        ], f => ['top', ...f.variant.map((v) => v.value || '')]);

        treeDistinctCounts(groups[0], f => f.testId ? new Set([f.testId]) : new Set(), setFailures);

        expect(groups[0].failures).toBe(2);
        expect(groups[0].children[0].failures).toBe(1);
        expect(groups[0].children[1].failures).toBe(1);
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

        expect(groups.map(g => g.name)).toEqual(['a', 'b', 'c']);
    });
    it('sorts top level groups descending', () => {
        const groups: FailureGroup[] = [
            newGroup('c').withFailures(3).build(),
            newGroup('a').withFailures(1).build(),
            newGroup('b').withFailures(2).build(),
        ];

        sortFailureGroups(groups, 'failures', false);

        expect(groups.map(g => g.name)).toEqual(['c', 'b', 'a'])
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

        expect(groups.map(g => g.name)).toEqual(['a', 'b', 'c']);
        expect(groups[0].children.map(g => g.name)).toEqual(['a1', 'a2', 'a3']);
    });
    it('sorts on an alternate metric', () => {
        const groups: FailureGroup[] = [
            newGroup('c').withPresubmitRejects(3).build(),
            newGroup('a').withPresubmitRejects(1).build(),
            newGroup('b').withPresubmitRejects(2).build(),
        ];

        sortFailureGroups(groups, 'presubmitRejects', true);

        expect(groups.map(g => g.name)).toEqual(['a', 'b', 'c'])
    });
});
