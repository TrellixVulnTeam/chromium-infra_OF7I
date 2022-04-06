// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { ClusterId } from '../../services/rules';

export const linkToCluster = (project: string, c: ClusterId): string => {
    if (c.algorithm.startsWith('rules-')) {
        return linkToRule(project, c.id);
    } else {
        const projectEncoded = encodeURIComponent(project);
        const algorithmEncoded = encodeURIComponent(c.algorithm);
        const idEncoded = encodeURIComponent(c.id);
        return `/p/${projectEncoded}/clusters/${algorithmEncoded}/${idEncoded}`;
    }
};

export const linkToRule = (project: string, ruleId: string): string => {
    const projectEncoded = encodeURIComponent(project);
    const ruleIdEncoded = encodeURIComponent(ruleId);
    return `/p/${projectEncoded}/rules/${ruleIdEncoded}`;
};