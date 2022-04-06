// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { useCallback } from 'react';
import { useQuery } from 'react-query';
import { useParams } from 'react-router-dom';

import LinearProgress from '@mui/material/LinearProgress';
import Paper from '@mui/material/Paper';

import Container from '@mui/material/Container';
import { getCluster } from '../../services/cluster';
import ErrorAlert from '../error_alert/error_alert';

const ImpactSection = () => {
    const { project, algorithm, id } = useParams();
    let currentAlgorithm = algorithm;
    if(!currentAlgorithm) {
        currentAlgorithm = 'rules-v1';
    }
    const { isLoading, isError, data: cluster, error } = useQuery(['cluster'], () => {
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        return getCluster(project!, currentAlgorithm!, id!);
    });

    const impactTableRef = useCallback(node => {
        if (node !== null) {
            node.currentCluster = cluster;
        }
    }, [cluster]);

    if(isLoading) {
        return <LinearProgress />;
    }

    if(isError) {
        return <ErrorAlert
            errorText={`Got an error while loading the cluster: ${error}`}
            errorTitle="Failed to load cluster"
            showError
        />;
    }

    return (
        <Paper elevation={3} sx={{ pt: 2, mb:5 }}>
            <Container maxWidth={false}>
                <h2>Impact</h2>
                <impact-table ref={impactTableRef}></impact-table>
                <h2>Recent Failures</h2>
                <failure-table project={project} clusterAlgorithm={currentAlgorithm} clusterID={id}></failure-table>
            </Container>
        </Paper>
    );
};

export default ImpactSection;