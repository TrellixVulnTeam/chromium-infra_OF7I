// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, {
    Dispatch,
    SetStateAction,
    useEffect,
    useState
} from 'react';
import { useParams } from 'react-router-dom';

import LoadingButton from '@mui/lab/LoadingButton';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import LinearProgress from '@mui/material/LinearProgress';

import useFetchRule from '../../../hooks/useFetchRule';
import { useMutateRule } from '../../../hooks/useMutateRule';
import { UpdateRuleRequest } from '../../../services/rules';
import BugPicker from '../../bug_picker/bug_picker';
import ErrorAlert from '../../error_alert/error_alert';

interface Props {
    open: boolean;
    setOpen: Dispatch<SetStateAction<boolean>>;
}


const BugEditDialog = ({
    open,
    setOpen,
}: Props) => {

    const { project, id: ruleId } = useParams();

    const { isLoading, isError, data: rule, error } = useFetchRule(ruleId, project);

    const [bugSystem, setBugSystem] = useState('');
    const [bugId, setBugId] = useState('');

    const mutateRule = useMutateRule(() => {
        setOpen(false);
    });

    useEffect(() => {
        if(rule) {
            setBugId(rule.bug.id);
            setBugSystem(rule.bug.system);
        }
    }, [rule]);

    if(!ruleId || !project) {
        return <ErrorAlert
            errorText={'Project and/or rule are not defined in the URL'}
            errorTitle="Project and/or rule are undefined"
            showError
        />;
    }

    if(isError || !rule) {
        return <ErrorAlert
            errorText={`An erro occured while fetching the rule: ${error}`}
            errorTitle="Failed to load rule"
            showError
        />;
    }

    const handleBugSystemChanged = (bugSystem: string) => {
        setBugSystem(bugSystem);
    };

    const handleBugIdChanged = (bugId: string) => {
        setBugId(bugId);
    };

    const handleClose = () => {
        setBugSystem(rule.bug.system);
        setBugId(rule.bug.id);
        setOpen(false);
    };

    const handleSave = () => {
        const request: UpdateRuleRequest = {
            rule: {
                name: rule.name,
                bug: {
                    system: bugSystem,
                    id: bugId,
                },
            },
            updateMask: 'bug',
            etag: rule.etag,
        };
        mutateRule.mutate(request);
    };


    if (isLoading) {
        return <LinearProgress />;
    }

    return (
        <>
            <Dialog open={open} fullWidth>
                <DialogTitle>Change associated bug</DialogTitle>
                <DialogContent sx={{ mt: 1 }}>
                    <BugPicker
                        bugSystem={bugSystem}
                        bugId={bugId}
                        handleBugSystemChanged={handleBugSystemChanged}
                        handleBugIdChanged={handleBugIdChanged}
                    />
                </DialogContent>
                <DialogActions>
                    <Button
                        variant="outlined"
                        onClick={handleClose}>
                        Cancel
                    </Button>
                    <LoadingButton
                        variant="contained"
                        onClick={handleSave}
                        loading={mutateRule.isLoading}
                    >
                        Save
                    </LoadingButton>
                </DialogActions>
            </Dialog>
        </>
    );
};

export default BugEditDialog;