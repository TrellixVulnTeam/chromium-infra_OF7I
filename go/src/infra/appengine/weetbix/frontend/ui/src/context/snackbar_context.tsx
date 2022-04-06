// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, {
    createContext,
    Dispatch,
    SetStateAction,
    useState
} from 'react';

import { AlertColor } from '@mui/material/Alert';

export interface Snack {
    open?: boolean;
    message?: string;
    severity?: AlertColor;
}

export interface SnackbarContextData {
    snack: Snack;
    setSnack: Dispatch<SetStateAction<Snack>>;
}

export const snackContextDefaultState: Snack = {
    open: false,
    message: '',
    severity: 'success',
};


export const SnackbarContext = createContext<SnackbarContextData>({
    snack: snackContextDefaultState,
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    setSnack: () => {}
});

interface Props {
    children: React.ReactNode;
};

export const SnackbarContextWrapper = ({ children }: Props) => {
    const [snack, setSnack] = useState<Snack>(snackContextDefaultState);
    return (
        <SnackbarContext.Provider value={{ snack, setSnack }}>
            {children}
        </SnackbarContext.Provider>
    );
};