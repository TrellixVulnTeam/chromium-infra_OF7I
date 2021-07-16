// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as React from 'react';
import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { useSelector } from 'react-redux';

import {
  selectCurrentPeriod,
  selectNumPeriods,
  selectVisibleDates,
  selectVisibleMetrics,
} from './metricsSlice';

interface Props {
  orderBy: number;
  order: string;
  filter: string;
}

/*
  Updates the URL with the current parameters, so that the current view is
  easily linkable by copying the current address in the URL bar.
*/
const MetricsTableParams: React.FunctionComponent<Props> = ({
  orderBy,
  order,
  filter,
}: Props) => {
  const [params, setParams] = React.useState<string>('');

  const dates = useSelector(selectVisibleDates);
  const metrics = useSelector(selectVisibleMetrics);
  const numPeriods = useSelector(selectNumPeriods);
  const period = useSelector(selectCurrentPeriod);

  const location = useLocation();

  const createURLParams = () => {
    let params = `?periods=${numPeriods}&period=${period}`;
    if (dates.length > 0) {
      params += `&date=${dates[dates.length - 1]}`;
    }
    if (filter !== '') {
      params += '&filter=' + encodeURIComponent(filter);
    }
    metrics.forEach((metric) => {
      params += '&metric=' + encodeURIComponent(metric);
    });
    params += `&orderBy=${orderBy}&order=${order}`;
    return params;
  };
  // This might be a bit of a hack.  History seems to cause re-renders
  // TODO(gatong) Investigate further if I should use history instead
  useEffect(() => {
    const timer = setTimeout(() => {
      setParams(createURLParams());
    }, 100);
    return () => clearTimeout(timer);
  });
  useEffect(() => {
    if (params !== '') {
      window.history.replaceState(
        null,
        window.document.title,
        location.pathname + params
      );
    }
  }, [params]);

  return <></>;
};

export default MetricsTableParams;
