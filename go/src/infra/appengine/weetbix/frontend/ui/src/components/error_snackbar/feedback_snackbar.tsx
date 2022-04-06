// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { useContext } from 'react';

import Alert from '@mui/material/Alert';
import Snackbar from '@mui/material/Snackbar';

import {
    SnackbarContext,
    snackContextDefaultState
} from '../../context/snackbar_context';

const FeedbackSnackbar = () => {

    const { snack, setSnack } = useContext(SnackbarContext);

    const handleClose = () => {
        setSnack(snackContextDefaultState);
    };

    return (
        <Snackbar
            data-testid="snackbar"
            open={snack.open}
            autoHideDuration={6000}
            anchorOrigin={{ horizontal: 'center', vertical: 'bottom' }}
            onClose={handleClose}>
            <Alert onClose={handleClose} severity={snack.severity} sx={{ width: '100%' }}>
                {snack.message}
            </Alert>
        </Snackbar>
    );
};

export default FeedbackSnackbar;