// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { useState } from 'react';
import { useQuery } from 'react-query';

import Edit from '@mui/icons-material/Edit';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Container from '@mui/material/Container';
import Divider from '@mui/material/Divider';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import LinearProgress from '@mui/material/LinearProgress';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Switch from '@mui/material/Switch';
import Typography from '@mui/material/Typography';

import { useMutateRule } from '../../../hooks/useMutateRule';
import {
    GetIssueRequest,
    getIssuesService
} from '../../../services/monorail';
import {
    Rule,
    UpdateRuleRequest
} from '../../../services/rules';
import { MuiDefaultColor } from '../../../types/mui_types';
import ErrorAlert from '../../error_alert/error_alert';
import GridLabel from '../../grid_label/grid_label';
import HelpTooltip from '../../help_tooltip/help_tooltip';
import BugEditDialog from '../bug_edit_dialog/bug_edit_dialog';

const createIssueServiceRequest = (rule: Rule): GetIssueRequest => {
    const parts = rule.bug.id.split('/');
    const monorailProject = parts[0];
    const bugId = parts[1];
    const issueId = `projects/${monorailProject}/issues/${bugId}`;
    return {
        name: issueId
    };
};

const bugStatusColor = (status: string): MuiDefaultColor => {
    // In monorail, bug statuses are configurable per system. Right now,
    // we don't have a configurable mapping from status to semantic in
    // Weetbix. We will try to recognise common terminology and fall
    // back to "other" status otherwise.
    status = status.toLowerCase();
    const unassigned = ['new', 'untriaged', 'available'];
    const assigned = ['accepted', 'assigned', 'started', 'externaldependency'];
    const fixed = ['fixed', 'verified'];
    if (unassigned.indexOf(status) >= 0) {
        return 'error';
    } else if (assigned.indexOf(status) >= 0) {
        return 'primary';
    } else if (fixed.indexOf(status) >= 0) {
        return 'success';
    } else {
        // E.g. Won't fix, duplicate, archived.
        return 'info';
    }
};

const bugUpdatesHelpText = 'Whether the priority and verified status of the associated bug should be'
    + ' automatically updated based on cluster impact. Only one rule may be set to'
    + ' update a given bug at any one time.';

interface Props {
    rule: Rule;
}

const BugInfo = ({
    rule
}: Props) => {

    const issueService = getIssuesService();

    const [editDialogOpen, setEditDialogOpen] = useState(false);

    const fetchBugRequest = createIssueServiceRequest(rule);
    const { isLoading, isError, data: issue, error } = useQuery(['bug', fetchBugRequest.name],
        async () => await issueService.getIssue(fetchBugRequest)
    );

    const mutateRule = useMutateRule();

    if (isLoading) {
        return <LinearProgress />;
    }

    if (isError) {
        return (
            <Paper elevation={3} sx={{ py: 2, mt: 1 }}>
                <Container>
                    <ErrorAlert
                        showError={true}
                        errorTitle='Failed to load bug details.'
                        errorText={`Failed to load bug details due to: ${error}`}
                    />
                </Container>
            </Paper>
        );
    }

    const handleToggleUpdateBug = () => {
        const request: UpdateRuleRequest = {
            rule: {
                name: rule.name,
                isManagingBug: !rule.isManagingBug,
            },
            updateMask: 'isManagingBug',
            etag: rule.etag,
        };
        mutateRule.mutate(request);
    };

    return (
        <>
            {
                issue && (
                    <Paper elevation={3} sx={{ pt: 2, mt: 1, flexGrow: 1 }}>
                        <Container maxWidth={false}>
                            <Typography sx={{
                                fontWeight: 600,
                                fontSize: 20
                            }}>
                                Bug details
                            </Typography>
                            <Grid container rowGap={2}>
                                <GridLabel text="Bug">
                                </GridLabel>
                                <Grid container item xs={10} alignItems="center" columnGap={1}>
                                    <Link target="_blank" href={rule.bug.url}>
                                        {rule.bug.linkText}
                                    </Link>
                                    <Chip label={issue.status.status} color={bugStatusColor(issue.status.status)} />
                                    <IconButton aria-label="edit" onClick={() => setEditDialogOpen(true)}>
                                        <Edit />
                                    </IconButton>
                                </Grid>
                                <Grid item xs={12}>
                                    <Divider />
                                </Grid>
                                <GridLabel text="Summary" xs={12} />
                                <Grid container item xs={12}>
                                    {issue.summary}
                                </Grid>
                                <Grid item xs={12}>
                                    <Divider />
                                </Grid>
                                <GridLabel text="Update bug">
                                    <HelpTooltip text={bugUpdatesHelpText} />
                                </GridLabel>
                                <Grid container item xs={10} alignItems="center">
                                    {mutateRule.isLoading && (<CircularProgress size="1rem" />)}
                                    <Switch
                                        aria-label="receive bug status"
                                        checked={rule.isManagingBug}
                                        onChange={handleToggleUpdateBug}
                                        disabled={mutateRule.isLoading}
                                    />
                                </Grid>
                            </Grid>
                        </Container>
                        <BugEditDialog
                            open={editDialogOpen}
                            setOpen={setEditDialogOpen}
                        />
                    </Paper>
                )
            }
        </>
    );
};

export default BugInfo;