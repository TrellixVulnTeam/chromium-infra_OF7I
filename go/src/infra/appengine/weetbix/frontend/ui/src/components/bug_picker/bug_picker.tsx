// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, {
    ChangeEvent
} from 'react';
import { useQuery } from 'react-query';
import { useParams } from 'react-router-dom';

import CircularProgress from '@mui/material/CircularProgress';
import FormControl from '@mui/material/FormControl';
import Grid from '@mui/material/Grid';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import TextField from '@mui/material/TextField';

import { readProjectConfig } from '../../services/config';
import ErrorAlert from '../error_alert/error_alert';

interface Props {
    bugSystem: string;
    bugId: string;
    handleBugSystemChanged: (bugSystem: string) => void;
    handleBugIdChanged: (bugId: string) => void;
}

const getMonorailSystem = (bugId: string): string | null => {
    if (bugId.indexOf('/') >= 0) {
        const parts = bugId.split('/');
        return parts[0];
    } else {
        return null;
    }
};

const getBugNumber = (bugId: string): string => {
    if (bugId.indexOf('/') >= 0) {
        const parts = bugId.split('/');
        return parts[1];
    } else {
        return bugId;
    }
};

/**
 * An enum representing the supported bug systems.
 *
 * This is needed because mui's <Select> doesn't compare string correctly in typescript.
 */
enum BugSystems {
    MONORAIL = 'monorail',
    BUGANIZER = 'buganizer',
}


/**
 * This method works around the fact that Select
 * components compare strings for reference.
 *
 * @param {string} bugSystem The bug system to find in the enum.
 * @return {string} A static enum value equal to the string
 *          provided and used in the Select component.
 */
const getStaticBugSystem = (bugSystem: string): string => {
    switch (bugSystem) {
        case 'monorail': {
            return BugSystems.MONORAIL;
        }
        case 'buganizer': {
            return BugSystems.BUGANIZER;
        }
        default: {
            throw new Error('Unnkown bug system.');
        }
    }
};

const BugPicker = ({
    bugSystem,
    bugId,
    handleBugSystemChanged,
    handleBugIdChanged,
}: Props) => {
    const { project } = useParams();

    if(!project) {
        return (
            <ErrorAlert
                showError
                errorTitle="Project not defined"
                errorText={'No project param detected.}'}
            />
        );
    }

    const selectedBugSystem = getStaticBugSystem(bugSystem);

    const {
        isLoading,
        isError,
        data: projectConfig,
        error
    } = useQuery(['project', project], async () => {
        return await readProjectConfig(project);
    });

    if (isLoading) {
        return (
            <Grid container justifyContent="center">
                <CircularProgress data-testid="circle-loading" />
            </Grid>
        );
    }

    if (isError || !projectConfig) {
        return <ErrorAlert
            showError
            errorTitle="Failed to load project config"
            errorText={`An error occured while fetching the project config: ${error}`}
        />;
    }

    const monorailSystem = getMonorailSystem(bugId);

    const onBugSystemChange = (e: SelectChangeEvent<typeof bugSystem>) => {
        handleBugSystemChanged(e.target.value);

        // When the bug system changes, we also need to update the Bug ID.
        if (e.target.value == 'monorail') {
            handleBugIdChanged(`${projectConfig.monorail.project}/${getBugNumber(bugId)}`);
        } else if (e.target.value == 'buganizer') {
            handleBugIdChanged(getBugNumber(bugId));
        }
    };

    const onBugNumberChange = (e: ChangeEvent<HTMLInputElement>) => {
        const enteredBugId = e.target.value;

        if (monorailSystem != null) {
            handleBugIdChanged(`${monorailSystem}/${enteredBugId}`);
        } else {
            handleBugIdChanged(enteredBugId);
        }
    };

    return (
        <Grid container item columnSpacing={1} sx={{ mt: 1 }}>
            <Grid item xs={6}>
                <FormControl variant="standard" fullWidth>
                    <InputLabel id="bug-picker_select-bug-tracker-label">Bug tracker</InputLabel>
                    <Select
                        labelId="bug-picker_select-bug-tracker-label"
                        id="bug-picker_select-bug-tracker"
                        value={selectedBugSystem}
                        onChange={onBugSystemChange}
                        variant="standard"
                        inputProps={{ 'data-testid': 'bug-system' }}
                    >
                        <MenuItem value={getStaticBugSystem('monorail')}>
                            {projectConfig.monorail.displayPrefix}
                        </MenuItem>
                        <MenuItem value={getStaticBugSystem('buganizer')}>
                            Buganizer
                        </MenuItem>
                    </Select>
                </FormControl>
            </Grid>
            <Grid item xs={6}>
                <TextField
                    label="Bug number"
                    variant="standard"
                    inputProps={{ 'data-testid': 'bug-number' }}
                    value={getBugNumber(bugId)}
                    onChange={onBugNumberChange}
                />
            </Grid>
        </Grid>
    );
};

export default BugPicker;