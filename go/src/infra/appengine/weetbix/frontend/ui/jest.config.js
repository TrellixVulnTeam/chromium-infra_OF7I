// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
/** @type {import('ts-jest/dist/types').InitialOptionsTsJest} */
// eslint-disable-next-line no-undef
module.exports = {
    preset: 'ts-jest/presets/js-with-ts',
    testEnvironment: 'jsdom',
    testMatch: [
        '**/__tests__/**/*.[jt]s?(x)',
        '**/?(*.)+(test).[jt]s?(x)'
    ],
    /**
   * The reason we need to set this is because we are importing node_modules which are using
   * `es6` modules and that is not compatible with jest, so we need to transform them.
   */
    transformIgnorePatterns: ['/node_modules/(?!(lit-element|lit-html|@material|lit|@lit|node-fetch|data-uri-to-buffer|fetch-blob|formdata-polyfill)/)'],
    moduleNameMapper: {
        '\\.(css|less)$': 'identity-obj-proxy'
    },
    setupFiles: [
        './src/testing_tools/setUpEnv.ts'
    ]
};