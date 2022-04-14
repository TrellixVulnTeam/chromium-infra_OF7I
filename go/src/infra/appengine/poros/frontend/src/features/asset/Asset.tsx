// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';

import RefreshIcon from '@mui/icons-material/Refresh';
import DeleteIcon from '@mui/icons-material/Delete';
import { Typography } from '@mui/material';
import Grid from '@mui/material/Grid';
import Box from '@mui/material/Box';
import TextField from '@mui/material/TextField';
import {
  clearSelectedRecord,
  createAssetAsync,
  setDescription,
  setName,
} from './assetSlice';
import { useAppSelector, useAppDispatch } from '../../app/hooks';

export const Asset = () => {
  const name: string = useAppSelector((state) => state.asset.record.name);
  const description: string = useAppSelector(
    (state) => state.asset.record.description
  );
  const assetId: string = useAppSelector((state) => state.asset.record.assetId);
  const dispatch = useAppDispatch();

  const handleSaveClick = (
    name: string,
    description: string,
    assetId: string
  ) => {
    if (assetId === '') {
      dispatch(createAssetAsync({ name, description }));
    }
  };

  const handleCancelClick = () => {
    dispatch(clearSelectedRecord());
  };

  return (
    <Box
      sx={{
        width: 500,
        maxWidth: '100%',
        padding: 1,
      }}
    >
      <Grid container spacing={2} padding={1}>
        <Grid
          item
          style={{
            display: 'flex',
            justifyContent: 'flex-start',
            alignItems: 'center',
          }}
          xs={8}
        >
          <Typography variant="h5">Asset</Typography>
        </Grid>
      </Grid>
      <Grid container spacing={2} padding={1}>
        <Grid item xs={12}>
          <TextField
            label="Name"
            id="name"
            value={name}
            onChange={(e) => dispatch(setName(e.target.value))}
            fullWidth
            InputProps={{ fullWidth: true }}
            variant="standard"
          />
        </Grid>
      </Grid>
      <Grid container spacing={2} padding={1}>
        <Grid item xs={12}>
          <TextField
            id="description"
            label="Description"
            multiline
            rows={4}
            variant="standard"
            onChange={(e) => dispatch(setDescription(e.target.value))}
            value={description}
            fullWidth
            InputProps={{ fullWidth: true }}
          />
        </Grid>
      </Grid>
      <Grid container spacing={2} padding={1}>
        <Grid item xs={12}>
          <TextField
            disabled
            label="Id"
            id="assetid"
            variant="standard"
            value={assetId}
            fullWidth
            InputProps={{ fullWidth: true }}
          />
        </Grid>
      </Grid>
      <Grid container spacing={2} padding={1}>
        <Grid
          item
          style={{
            display: 'flex',
            justifyContent: 'flex-start',
            alignItems: 'center',
          }}
          xs={8}
        >
          <Typography variant="h6">Associated Machines</Typography>
        </Grid>
      </Grid>

      <Grid container spacing={2} padding={1}>
        <Grid
          item
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'right',
          }}
          xs={12}
        >
          <Stack direction="row" spacing={2}>
            <Button
              variant="outlined"
              onClick={handleCancelClick}
              startIcon={<RefreshIcon />}
            >
              Cancel
            </Button>
            <Button
              variant="contained"
              onClick={() => handleSaveClick(name, description, assetId)}
              endIcon={<DeleteIcon />}
            >
              Save
            </Button>
          </Stack>
        </Grid>
      </Grid>
    </Box>
  );
};
