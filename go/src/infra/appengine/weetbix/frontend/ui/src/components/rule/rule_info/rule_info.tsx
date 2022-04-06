// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';

import Archive from '@mui/icons-material/Archive';
import Edit from '@mui/icons-material/Edit';
import Unarchive from '@mui/icons-material/Unarchive';
import LoadingButton from '@mui/lab/LoadingButton';
import Container from '@mui/material/Container';
import Divider from '@mui/material/Divider';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';

import { useMutateRule } from '../../../hooks/useMutateRule';
import {
    Rule,
    UpdateRuleRequest
} from '../../../services/rules';
import { linkToCluster } from '../../../tools/urlHandling/links';
import CodeBlock from '../../codeblock/codeblock';
import ConfirmDialog from '../../confirm_dialog/confirm_dialog';
import GridLabel from '../../grid_label/grid_label';
import HelpTooltip from '../../help_tooltip/help_tooltip';
import RuleEditDialog from '../rule_edit_dialog/rule_edit_dialog';

const archivedTooltipText = 'Archived failure association rules do not match failures. If a rule is no longer needed, it should be archived.';
const sourceClusterTooltipText = 'The cluster this rule was originally created from.';
interface Props {
    project: string;
    rule: Rule;
}

const RuleInfo = ({ project, rule }: Props) => {

    const [editDialogOpen, setEditDialogOpen] = useState(false);
    const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);

    const mutateRule = useMutateRule();

    const toggleArchived = () => {
        const request: UpdateRuleRequest = {
            rule: {
                name: rule.name,
                isActive: !rule.isActive,
            },
            updateMask: 'isActive',
            etag: rule.etag,
        };
        mutateRule.mutate(request);
    };

    const onArchiveConfirm = () => {
        setConfirmDialogOpen(false);
        toggleArchived();
    };

    const onArchiveCancel = () => {
        setConfirmDialogOpen(false);
    };

    return (
        <Paper elevation={3} sx={{ pt: 2, pb: 2, mt: 1 }} >
            <Container maxWidth={false}>
                <Typography sx={{
                    fontWeight: 600,
                    fontSize: 20
                }}>
                    Details
                </Typography>
                <Grid container rowGap={2}>
                    <GridLabel text="Rule definition" />
                    <Grid container item xs={10} alignItems="center">
                        <IconButton onClick={() => setEditDialogOpen(true)} aria-label="edit">
                            <Edit />
                        </IconButton>
                    </Grid>
                    <Grid item xs={12} zeroMinWidth>
                        <CodeBlock code={rule.ruleDefinition} />
                    </Grid>
                    <Grid item xs={12}>
                        <Divider />
                    </Grid>
                    <GridLabel text="Source cluster">
                        <HelpTooltip text={sourceClusterTooltipText} />
                    </GridLabel>
                    <Grid container item xs={10} alignItems="center">
                        {
                            rule.sourceCluster.algorithm && rule.sourceCluster.id ? (
                                <Link aria-label='source cluster link' component={RouterLink} to={linkToCluster(project, rule.sourceCluster)}>
                                    {rule.sourceCluster.algorithm}/{rule.sourceCluster.id}
                                </Link>
                            ) : (
                                <Typography>None</Typography>
                            )
                        }
                    </Grid>
                    <Grid item xs={12}>
                        <Divider />
                    </Grid>
                    <GridLabel text="Archived">
                        <HelpTooltip text={archivedTooltipText} />
                    </GridLabel>
                    <Grid container item xs={10} alignItems="center" columnGap={1}>
                        <Typography>{rule.isActive ? 'No' : 'Yes'}</Typography>
                        <LoadingButton
                            loading={mutateRule.isLoading}
                            variant="outlined"
                            startIcon={rule.isActive ? (<Archive />) : (<Unarchive />)}
                            onClick={() => setConfirmDialogOpen(true)}
                        >
                            {rule.isActive ? 'Archive' : 'Restore'}
                        </LoadingButton>
                    </Grid>
                </Grid>
            </Container>
            <ConfirmDialog
                open={confirmDialogOpen}
                message={`Are you sure you want to ${rule.isActive ? 'archive' : 'restore'} this rule?`}
                onConfirm={onArchiveConfirm}
                onCancel={onArchiveCancel}
            />
            <RuleEditDialog
                open={editDialogOpen}
                setOpen={setEditDialogOpen}
                rule={rule}
            />
        </Paper>
    );
};

export default RuleInfo;