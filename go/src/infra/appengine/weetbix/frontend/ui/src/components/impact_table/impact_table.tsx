// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';

import Box from '@mui/material/Box';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';

import { Cluster, Counts } from '../../services/cluster';
import HelpTooltip from '../help_tooltip/help_tooltip';


const userClsFailedPresubmitTooltipText = 'The number of distinct developer changelists that failed at least one CQ run because of failure(s) in this cluster.';
const testRunsFailedTooltipText = 'The number of invocations (e.g. swarming tasks used to run tests) failed because of failures in this cluster. Invocations are usually expensive to setup and retry, so this is a measure of machine time wasted.';
const unexpectedFailuresTooltipText = 'The total number of test results in this cluster.';

interface Props {
    cluster: Cluster;
}

const ImpactTable = ({ cluster }: Props) => {
    const metric = (counts: Counts): number => {
        return counts.preWeetbix;
    };

    return (
        <TableContainer component={Box}>
            <Table data-testid="impact-table" size="small" sx={{ maxWidth: 500 }}>
                <TableHead>
                    <TableRow>
                        <TableCell></TableCell>
                        <TableCell align="right">1 day</TableCell>
                        <TableCell align="right">3 days</TableCell>
                        <TableCell align="right">7 days</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    <TableRow>
                        <TableCell>User Cls Failed Presubmit <HelpTooltip text={userClsFailedPresubmitTooltipText} /></TableCell>
                        <TableCell align="right">{metric(cluster.presubmitRejects1d)}</TableCell>
                        <TableCell align="right">{metric(cluster.presubmitRejects3d)}</TableCell>
                        <TableCell align="right">{metric(cluster.presubmitRejects7d)}</TableCell>
                    </TableRow>
                    <TableRow>
                        <TableCell>Test Runs Failed <HelpTooltip text={testRunsFailedTooltipText} /></TableCell>
                        <TableCell align="right">{metric(cluster.testRunFailures1d)}</TableCell>
                        <TableCell align="right">{metric(cluster.testRunFailures3d)}</TableCell>
                        <TableCell align="right">{metric(cluster.testRunFailures7d)}</TableCell>
                    </TableRow>
                    <TableRow>
                        <TableCell>Unexpected Failures <HelpTooltip text={unexpectedFailuresTooltipText} /></TableCell>
                        <TableCell align="right">{metric(cluster.failures1d)}</TableCell>
                        <TableCell align="right">{metric(cluster.failures3d)}</TableCell>
                        <TableCell align="right">{metric(cluster.failures7d)}</TableCell>
                    </TableRow>
                </TableBody>
            </Table>
        </TableContainer>
    );
};

export default ImpactTable;