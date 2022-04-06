// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';

import Grid from '@mui/material/Grid';

interface Props {
    text?: string;
    children?: React.ReactNode,
    xs?: number;
}

const GridLabel = ({
    text,
    children,
    xs = 2,
}: Props) => {
    return (
        <Grid container item xs={xs} alignItems="center">
            {text}
            {children}
        </Grid>
    );
};

export default GridLabel;