// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {
  DataGrid,
  GridRowsProp,
  GridColDef,
  GridToolbarContainer,
  GridToolbarColumnsButton,
  GridToolbarDensitySelector,
  GridCellParams,
  MuiEvent,
  GridRenderCellParams,
} from '@mui/x-data-grid';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';

import RefreshIcon from '@mui/icons-material/Refresh';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import { Typography } from '@mui/material';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import {
  clearSelectedRecord,
  onSelectRecord,
  queryAssetAsync,
} from './assetSlice';
import { useAppSelector, useAppDispatch } from '../../app/hooks';
import { Asset } from './Asset';

export function AssetList() {
  const dispatch = useAppDispatch();

  const rows: GridRowsProp = useAppSelector((state) => state.asset.assets);
  const columns: GridColDef[] = [
    { field: 'assetId', headerName: 'Id', width: 150 },
    { field: 'name', headerName: 'Name', width: 150 },
    { field: 'description', headerName: 'Description', width: 150 },
    { field: 'createdBy', headerName: 'Created By', width: 150 },
    { field: 'createdAt', headerName: 'Created At', width: 150 },
    {
      field: 'Edit',
      renderCell: (cellValues) => {
        return (
          <IconButton
            aria-label="delete"
            size="small"
            onClick={() => {
              handleEditClick(cellValues);
            }}
          >
            <EditIcon fontSize="inherit" />
          </IconButton>
        );
      },
    },
  ];
  const handleEditClick = (cellValues: GridRenderCellParams) => {
    const selectedRow = cellValues.row;
    dispatch(onSelectRecord({ assetId: selectedRow.assetId }));
    console.log(cellValues);
  };

  const handleCreateClick = () => {
    dispatch(clearSelectedRecord());
  };

  const handleRefreshClick = () => {
    dispatch(queryAssetAsync({ pageSize: 100, pageToken: '' }));
  };

  function CustomToolbar() {
    return (
      <GridToolbarContainer>
        <GridToolbarColumnsButton />
        <GridToolbarDensitySelector />
      </GridToolbarContainer>
    );
  }

  return (
    <div>
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
          <Typography variant="h6">Assets</Typography>
        </Grid>
        <Grid
          item
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
          }}
          xs={4}
        >
          <Stack direction="row" spacing={2}>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={handleRefreshClick}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              onClick={handleCreateClick}
              endIcon={<DeleteIcon />}
            >
              Create
            </Button>
          </Stack>
        </Grid>
      </Grid>

      <div style={{ width: '100%' }}>
        <DataGrid
          autoHeight
          getRowId={(r) => r.assetId}
          rows={rows}
          columns={columns}
          components={{
            Toolbar: CustomToolbar,
          }}
          onCellClick={(
            params: GridCellParams,
            event: MuiEvent<React.MouseEvent>
          ) => {
            event.defaultMuiPrevented = true;
          }}
        />
      </div>
      <Asset />
    </div>
  );
}
