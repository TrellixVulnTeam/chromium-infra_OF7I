// Copyright 202 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { useState } from 'react';

import { useAppSelector, useAppDispatch } from '../../app/hooks';
import { fetchLabAsync, selectLab } from './labSlice'
import styles from './Lab.module.css';

export function Lab() {
  const lab = useAppSelector(selectLab);
  const dispatch = useAppDispatch();

  return (
    <div>
      <div className={styles.row}>
        <button
          className={styles.asyncButton}
          onClick={() => dispatch(fetchLabAsync('test'))}
        >
          Update
        </button>
        <span className={styles.value}>{lab.name}</span>
      </div>
    </div>
  )
}
