// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import dayjs from 'dayjs';
import React, {
    useEffect,
    useState
} from 'react';
import {
    useQuery,
    useQueryClient
} from 'react-query';

import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Grid';

import {
    fetchProgress,
    noProgressToShow,
    progressNotYetStarted,
    progressToLatestAlgorithms,
    progressToLatestConfig,
    progressToRulesVersion
} from '../../services/progress';
import CircularProgressWithLabel from '../circular_progress_with_label/circular_progress_with_label';
import ErrorAlert from '../error_alert/error_alert';

interface Props {
    project: string;
    hasRule?: boolean | undefined;
    rulePredicateLastUpdated?: string | undefined;
}

const ReclusteringProgressIndicator = ({
    project,
    hasRule,
    rulePredicateLastUpdated,
}: Props) => {
    const [show, setShow] = useState(false);
    const [lastRefreshed, setLastRefreshed] = useState(dayjs());

    const [progressPerMille, setProgressPerMille] = useState(noProgressToShow);
    const [reclusteringTarget, setReclusteringTarget] = useState('');
    const queryClient = useQueryClient();

    const { isError, isLoading, data: progress, error } = useQuery(
        'reclusteringProgress',
        async () => {
            return await fetchProgress(project);
        }, {
            refetchInterval: () => {
                // Only update the progress if we are still less than 100%
                if(progressPerMille >= 1000) {
                    return false;
                }
                return 1000;
            },
            onSuccess: () => {
                setLastRefreshed(dayjs());
            }
        }
    );

    useEffect(() => {
        if(progress) {
            let currentProgressPerMille = progressToLatestAlgorithms(progress);
            let currentTarget = 'updated clustering algorithms';
            const configProgress = progressToLatestConfig(progress);
            if (configProgress < currentProgressPerMille) {
                currentTarget = 'updated clustering configuration';
                currentProgressPerMille = configProgress;
            }
            if (hasRule && rulePredicateLastUpdated) {
                const ruleProgress = progressToRulesVersion(progress, rulePredicateLastUpdated);
                if (ruleProgress < currentProgressPerMille) {
                    currentTarget = 'the latest rule definition';
                    currentProgressPerMille = ruleProgress;
                }
            }

            setReclusteringTarget(currentTarget);
            setProgressPerMille(currentProgressPerMille);
        }
    }, [progress, rulePredicateLastUpdated]);

    useEffect(() => {
        if(progressPerMille >= progressNotYetStarted && progressPerMille < 1000) {
            setShow(true);
        }
    }, [progressPerMille]);

    if(isLoading && !progress) {
        // no need to show anything if there is no progress and we are still loading
        return <></>;
    }

    if(isError || !progress) {
        return (
            <ErrorAlert
                errorText={`Failed to load reclustering progress${error ? ' due to ' + error : '.'}`}
                errorTitle="Loading reclustering progress failed"
                showError
            />
        );
    }

    const handleRefreshAnalysis = () => {
        queryClient.invalidateQueries('cluster');
        setShow(false);
    };

    let progressText = 'task queued';
    if (progressPerMille >= 0) {
        progressText = (progressPerMille / 10).toFixed(1) + '%';
    }

    const progressContent = () => {
        if(progressPerMille < 1000) {
            return (
                <>
                    <p>Weetbix is re-clustering test results to reflect {reclusteringTarget} ({progressText}). Cluster impact may be out-of-date.</p>
                    <small> Last update {lastRefreshed.local().toString()}.</small>
                </>
            );
        } else  {
            return 'Weetbix has finished re-clustering test results. Updated cluster impact is now available.';
        }
    };
    return (
        <>
            { show &&
                <Alert
                    severity={progressPerMille >= 1000 ? 'success' : 'info'}
                    icon={false}
                    sx={{
                        mt:1
                    }}
                >
                    <Grid container justifyContent="center" alignItems="center" columnSpacing={{ xs: 2 }}>
                        <Grid item>
                            <CircularProgressWithLabel
                                variant="determinate"
                                value={Math.max(0, progressPerMille / 10)}
                            />
                        </Grid>
                        <Grid item>
                            {progressContent()}
                        </Grid>
                        <Grid item>
                            {
                                progressPerMille >= 1000 && (
                                    <Button
                                        color="inherit"
                                        size="small"
                                        onClick={handleRefreshAnalysis}
                                    >
                                        View updated impact
                                    </Button>
                                )
                            }
                        </Grid>
                    </Grid>
                </Alert>
            }
        </>
    );
};

export default ReclusteringProgressIndicator;