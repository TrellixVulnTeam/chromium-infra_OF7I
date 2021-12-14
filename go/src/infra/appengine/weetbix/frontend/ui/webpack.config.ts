// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { Configuration } from 'webpack';

const config: Configuration = {
    entry: './src/index.ts',
    mode: 'development',
    module: {
        rules: [
            { test: /\.css$/, use: ['style-loader', 'css-loader'] },
            { test: /\.ts$/, use: 'ts-loader' },
        ],
    },
};

export default config;