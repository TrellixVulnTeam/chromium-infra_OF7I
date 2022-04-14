// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React, { useState } from 'react';

import { useAppSelector, useAppDispatch } from '../../app/hooks';
import { fetchAssetAsync, selectAsset } from './assetSlice'
import styles from './Asset.module.css';

export function Asset() {
  const asset = useAppSelector(selectAsset);
  const dispatch = useAppDispatch();

  return (
    <div>
      <div className={styles.row}>
        <button
          className={styles.asyncButton}
          onClick={() => dispatch(fetchAssetAsync('test'))}
        >
          Update
        </button>
        <span className={styles.value}>{asset.name}</span>
      </div>
    </div>
  )
}
