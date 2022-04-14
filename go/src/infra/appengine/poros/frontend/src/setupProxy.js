// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/* eslint-disable */
const { createProxyMiddleware } = require('http-proxy-middleware');

// Proxy all /api methods
module.exports = function (app) {
  app.use(
    '/prpc',
    createProxyMiddleware({
      target: 'http://localhost:8800',
      changeOrigin: true,
    })
  );
};
