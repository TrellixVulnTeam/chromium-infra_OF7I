// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, {
    ChangeEvent,
    Dispatch,
    SetStateAction,
    useState
} from 'react';

import LoadingButton from '@mui/lab/LoadingButton';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import TextField from '@mui/material/TextField';

import { useMutateRule } from '../../../hooks/useMutateRule';
import {
    Rule,
    UpdateRuleRequest
} from '../../../services/rules';

interface Props {
    open: boolean;
    setOpen: Dispatch<SetStateAction<boolean>>;
    rule: Rule;
}


const RuleEditDialog = ({
    open = false,
    setOpen,
    rule,
}: Props) => {

    const [currentRuleDefinition, setCurrentRuleDefinition] = useState(rule.ruleDefinition);

    const mutateRule = useMutateRule(() => {
        setOpen(false);
    });
    const handleDefinitionChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
        setCurrentRuleDefinition(e.target.value);
    };

    const handleClose = () => {
        setCurrentRuleDefinition(() => rule.ruleDefinition);
        setOpen(() => false);
    };

    const handleSave = () => {
        const request: UpdateRuleRequest = {
            rule: {
                name: rule.name,
                ruleDefinition: currentRuleDefinition,
            },
            updateMask: 'ruleDefinition',
            etag: rule.etag,
        };
        mutateRule.mutate(request);
    };

    return (
        <Dialog
            open={open}
            fullWidth
        >
            <DialogTitle>Edit rule definition</DialogTitle>
            <DialogContent>
                <TextField
                    id="rule-definition-input"
                    label="Definition"
                    multiline
                    margin="dense"
                    rows={4}
                    value={currentRuleDefinition}
                    onChange={handleDefinitionChange}
                    fullWidth
                    variant="filled"
                    autoFocus
                    inputProps={{ 'data-testid':'rule-input' }}
                />
                <small>
                    Supported is AND, OR, =,{'<>'}, NOT, IN, LIKE, parentheses and <a href="https://cloud.google.com/bigquery/docs/reference/standard-sql/functions-and-operators#regexp_contains">REGEXP_CONTAINS</a>.
                    Valid identifiers are <em>test</em> and <em>reason</em>.
                </small>
            </DialogContent>
            <DialogActions>
                <Button
                    color="error"
                    variant="outlined"
                    onClick={handleClose}>
                    Cancel
                </Button>
                <LoadingButton
                    variant="outlined"
                    color="success"
                    onClick={handleSave}
                    loading={mutateRule.isLoading}
                >
                    Save
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
};

export default RuleEditDialog;