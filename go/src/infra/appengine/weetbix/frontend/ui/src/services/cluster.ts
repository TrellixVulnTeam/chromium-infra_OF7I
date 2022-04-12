// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.


export const getCluster = async (
    project: string,
    clusterAlgorithm: string,
    clusterId: string
):  Promise<Cluster> => {
    const response = await fetch(`/api/projects/${encodeURIComponent(project)}/clusters/${encodeURIComponent(clusterAlgorithm)}/${encodeURIComponent(clusterId)}`);
    return await response.json();
};

// Cluster is the cluster information sent by the server.
export interface Cluster {
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

export interface Counts {
    nominal: number;
    preWeetbix: number;
    preExoneration: number;
    residual: number;
    residualPreWeetbix: number;
    residualPreExoneration: number;
}

export interface ClusterId {
    algorithm: string;
    id: string;
}