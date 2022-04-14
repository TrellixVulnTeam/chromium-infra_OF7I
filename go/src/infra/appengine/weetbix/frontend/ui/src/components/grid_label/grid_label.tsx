// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';

import Grid from '@mui/material/Grid';
import Box from '@mui/material/Box';

interface Props {
    text?: string;
    children?: React.ReactNode,
    xs?: number;
    lg?: number;
}

const GridLabel = ({
    text,
    children,
    xs = 2,
    lg = xs,
}: Props) => {
    return (
        <Grid item xs={xs} lg={lg}>
            <Box
                sx={{
                    display: 'inline-block',
                    wordBreak:'break-all',
                    overflowWrap: 'break-word'
                }}
                paddingTop={1}>
                {text}
            </Box>
            {children}
        </Grid>
    );
};

export default GridLabel;