// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import dayjs from 'dayjs';
import isSameOrAfter from 'dayjs/plugin/isSameOrAfter';

dayjs.extend(isSameOrAfter);

export const fetchProgress = async (project: string): Promise<ReclusteringProgress> => {
    const response = await fetch(`/api/projects/${encodeURIComponent(project)}/reclusteringProgress`);
    return await response.json();
};

// ReclusteringTarget captures the rules and algorithms a re-clustering run
// is re-clustering to.
export interface ReclusteringTarget {
    rulesVersion: string; // RFC 3339 encoded date/time.
    configVersion: string; // RFC 3339 encoded date/time.
    algorithmsVersion: number;
}

// ReclusteringProgress captures the progress re-clustering a
// given LUCI project's test results with a specific rules
// version and/or algorithms version.
export interface ReclusteringProgress {
    // ProgressPerMille is the progress of the current re-clustering run,
    // measured in thousandths (per mille).
    progressPerMille: number;
    // LatestAlgorithmsVersion is the latest version of algorithms
    // used in a Weetbix re-clustering run.
    latestAlgorithmsVersion: number;
    // LatestAlgorithmsVersion is the latest version of configuration
    // used in a Weetbix re-clustering run, as a RFC 3339 encoded date/time.
    latestConfigVersion: string;
    // Next is the goal of the current re-clustering run. (For which
    // ProgressPerMille is specified.)
    next: ReclusteringTarget;
    // Last is the goal of the last completed re-clustering run.
    last: ReclusteringTarget;
}

export const progressNotYetStarted = -1;
export const noProgressToShow = -2;

export const  progressToLatestAlgorithms = (progress: ReclusteringProgress): number => {
    return progressTo(progress, (target: ReclusteringTarget) => {
        return target.algorithmsVersion >= progress.latestAlgorithmsVersion;
    });
};

export const progressToLatestConfig = (progress: ReclusteringProgress): number => {
    const targetConfigVersion = dayjs(progress.latestConfigVersion);
    return progressTo(progress, (target: ReclusteringTarget) => {
        return dayjs(target.configVersion).isSameOrAfter(targetConfigVersion);
    });
};

export const progressToRulesVersion = (progress: ReclusteringProgress, rulesVersion: string): number => {
    const ruleDate = dayjs(rulesVersion);
    return progressTo(progress, (target: ReclusteringTarget) => {
        return dayjs(target.rulesVersion).isSameOrAfter(ruleDate);
    });
};

// progressTo returns the progress to completing a re-clustering run
// satisfying the given re-clustering target, expressed as a predicate.
// If re-clustering has started, the returned value is value from 0 to
// 1000. If the run is pending, the value -1 is returned.
const progressTo = (progress: ReclusteringProgress, predicate: (target: ReclusteringTarget) => boolean): number => {
    if (predicate(progress.last)) {
        // Completed
        return 1000;
    }
    if (predicate(progress.next)) {
        return progress.progressPerMille;
    }
    // Run not yet started (e.g. because we are still finishing a previous
    // re-clustering).
    return progressNotYetStarted;
};