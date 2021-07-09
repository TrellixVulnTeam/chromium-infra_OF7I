// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { Redirect, RouteComponentProps } from 'react-router-dom';
import Container from '@material-ui/core/Container';
import Grid from '@material-ui/core/Grid';

import { useDispatch, useSelector } from 'react-redux';
import {
  selectAvailable,
  setCurrent,
} from '../features/dataSources/dataSourcesSlice';

interface MatchParams {
  dataSource?: string;
}

type Props = RouteComponentProps<MatchParams>;

const MetricsPage: React.FunctionComponent<Props> = ({
  match,
  location,
}: Props) => {
  const available = useSelector(selectAvailable);

  const dispatch = useDispatch();

  if (
    match.params.dataSource === undefined ||
    !available.some((ds) => ds.name === match.params.dataSource)
  ) {
    return <Redirect to={`/${available[0].name}`} />;
  }
  const params = new URLSearchParams(location.search);
  dispatch(setCurrent(match.params.dataSource, params));

  return (
    <Container maxWidth={false}>
      <Grid container spacing={3}>
        <Grid item xs={12}></Grid>
      </Grid>
    </Container>
  );
};

export default MetricsPage;
