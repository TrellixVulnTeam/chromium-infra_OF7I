// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { Config } from 'karma';

import webpackConfig from './webpack.config';

module.exports = (config: Config) => {
    const isDebug = process.argv.some((arg) => arg === '--debug');
    config.set({
        // base path that will be used to resolve all patterns (eg. files, exclude)
        basePath: '',

        // frameworks to use
        // available frameworks: https://npmjs.org/browse/keyword/karma-adapter
        frameworks: ['webpack', 'mocha', 'chai'],

        // list of files / patterns to load in the browser
        files: [{ pattern: 'index_test.ts', watched: false }],

        // list of files / patterns to exclude
        exclude: [],

        // preprocess matching files before serving them to the browser
        // available preprocessors:
        // https://npmjs.org/browse/keyword/karma-preprocessor
        preprocessors: {
            'index_test.ts': ['webpack', 'sourcemap'],
        },

        plugins: [
            'karma-chrome-launcher',
            'karma-webpack',
            'karma-sourcemap-loader',
            'karma-mocha',
            'karma-mocha-reporter',
            'karma-chai',
        ],

        webpack: {
            // webpack configuration
            devtool: 'inline-source-map',
            mode: 'development',
            module: webpackConfig.module,
            resolve: webpackConfig.resolve,
            externals: webpackConfig.externals,
            output: {
                // Use relative file path for cleaner stack trace with navigable source
                // location in the terminal.
                devtoolModuleFilenameTemplate: '[resource-path]',
            },
            optimization: {
                // Disable splitChunks. Otherwise sourcemap won't work in the terminal.
                // https://github.com/ryanclark/karma-webpack/issues/493
                splitChunks: false,
            },
        },

        webpackMiddleware: {
            stats: 'errors-only',
        },

        // test results reporter to use
        // possible values: 'dots', 'progress'
        // available reporters: https://npmjs.org/browse/keyword/karma-reporter
        reporters: ['mocha'],

        // web server port
        port: 9876,

        // enable / disable colors in the output (reporters and logs)
        colors: true,

        // level of logging
        // possible values: config.LOG_DISABLE || config.LOG_ERROR ||
        // config.LOG_WARN || config.LOG_INFO || config.LOG_DEBUG
        logLevel: config.LOG_INFO,

        // enable / disable watching file and executing tests whenever any file
        // changes
        autoWatch: true,

        // start these browsers
        // available browser launchers:
        // https://npmjs.org/browse/keyword/karma-launcher
        browsers: isDebug ? ['Chrome'] : ['ChromeHeadless'],

        // Continuous Integration mode
        // if true, Karma captures browsers, runs the tests and exits
        singleRun: false,

        // Concurrency level
        // how many browser should be started simultaneous
        concurrency: Infinity,
    });
};
