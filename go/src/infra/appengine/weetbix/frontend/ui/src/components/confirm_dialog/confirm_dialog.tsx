// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import Typography from '@mui/material/Typography';
import DialogActions from '@mui/material/DialogActions';
import Button from '@mui/material/Button';

type HandleFunction = () => void;

interface Props {
    message?: string;
    open: boolean;
    onConfirm: HandleFunction;
    onCancel: HandleFunction;
}

const ConfirmDialog = ({
    message = '',
    open,
    onConfirm,
    onCancel
}: Props) => {
    return (
        <Dialog open={open} maxWidth="xs" fullWidth>
            <DialogTitle>Are you sure?</DialogTitle>
            {message && (
                <DialogContent>
                    <Typography>{message}</Typography>
                </DialogContent>
            )
            }
            <DialogActions>
                <Button variant="outlined" color="error" onClick={onCancel}>Cancel</Button>
                <Button variant="outlined" color="success" onClick={onConfirm}>Save</Button>
            </DialogActions>
        </Dialog>
    );
};

export default ConfirmDialog;