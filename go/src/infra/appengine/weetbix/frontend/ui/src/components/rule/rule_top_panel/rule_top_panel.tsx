// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';

import Grid from '@mui/material/Grid';
import LinearProgress from '@mui/material/LinearProgress';

import useFetchRule from '../../../hooks/useFetchRule';
import ErrorAlert from '../../error_alert/error_alert';
import ReclusteringProgressIndicator from '../../reclustering_progress_indicator/reclustering_progress_indicator';
import TimestampInfoBar from '../../timestamp_info_bar/timestamp_info_bar';
import BugInfo from '../bug_info/bug_info';
import RuleInfo from '../rule_info/rule_info';

interface Props {
    project: string;
    ruleId: string;
}

const RuleTopPanel = ({ project, ruleId }: Props) => {
    const { isLoading, isError, data: rule, error } = useFetchRule(ruleId, project);

    if (isLoading) {
        return <LinearProgress />;
    }

    if (isError) {
        return (
            <ErrorAlert
                errorText={`An erro occured while fetching the rule: ${error}`}
                errorTitle="Failed to load rule"
                showError
            />
        );
    }

    return (
        <>
            {rule &&
                <Grid container columnSpacing={2}>
                    <Grid item xs={12}>
                        <ReclusteringProgressIndicator
                            hasRule={true}
                            project={project}
                            // eslint-disable-next-line @typescript-eslint/no-empty-function
                            refreshAnalysis={() => {}}
                            rulePredicateLastUpdated={rule.predicateLastUpdateTime}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TimestampInfoBar
                            createUsername={rule.createUser}
                            createTime={rule.createTime}
                            updateUsername={rule.lastUpdateUser}
                            updateTime={rule.lastUpdateTime}
                        />
                    </Grid>
                    <Grid container item xs={12} alignItems="stretch" display="flex" columnSpacing={2}>
                        <Grid item xs={6} display="flex" alignItems="stretch">
                            <RuleInfo project={project} rule={rule} />
                        </Grid>
                        <Grid item xs={6} display="flex" alignItems="stretch">
                            <BugInfo rule={rule} />
                        </Grid>
                    </Grid>
                </Grid>
            }
        </>
    );
};

export default RuleTopPanel;