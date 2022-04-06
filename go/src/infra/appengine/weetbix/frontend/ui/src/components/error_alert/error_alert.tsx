// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';

import Close from '@mui/icons-material/Close';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Collapse from '@mui/material/Collapse';
import IconButton from '@mui/material/IconButton';

interface Props {
    errorTitle: string;
    errorText: string;
    showError: boolean;
    onErrorClose?: () => void;
}

const ErrorAlert = ({
    errorTitle,
    errorText,
    showError,
    onErrorClose,
}: Props) => (
    <Collapse in={showError}>
        <Alert
            severity="error"
            action={<IconButton
                aria-label="close"
                color="inherit"
                size="small"
                onClick={onErrorClose}
            >
                <Close fontSize="inherit" />
            </IconButton>}
            sx={{ mb: 2 }}
        >
            <AlertTitle>{errorTitle}</AlertTitle>
            {errorText}
        </Alert>
    </Collapse>
);

export default ErrorAlert;