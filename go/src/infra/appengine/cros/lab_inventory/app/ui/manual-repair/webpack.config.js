// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

const path = require('path');
const {merge} = require('webpack-merge');
const {CleanWebpackPlugin} = require('clean-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');

const commonConfig = {
  entry: {
    polyfills: '@babel/polyfill',
    app: path.resolve(__dirname, 'src/main.ts'),
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: 'ts-loader',
        exclude: /node_modules/,
      },
      {
        test: /\.css$/,
        use: [
          'style-loader',
          'css-loader',
        ],
      },
      {
        test: /\.s[a|c]ss$/,
        use: [
          'style-loader',
          'css-loader',
          'sass-loader',
        ],
      },
    ],
  },
  plugins: [
    new CleanWebpackPlugin(),
    new CopyWebpackPlugin({
      patterns: [
        {
          from: path.resolve(
              __dirname, 'node_modules/@webcomponents/webcomponentsjs/*.js'),
          to: path.resolve(__dirname, 'dist/scripts/wc'),
          flatten: true,
        },
      ],
    }),
    new HtmlWebpackPlugin({
      title: 'Manual Repair Records',
      template: 'index.html',
    }),
  ],
  resolve: {
    extensions: ['.tsx', '.ts', '.js'],
  },
  output: {
    filename: '[name].js',
    path: path.resolve(__dirname, 'dist'),
  },
};

const devConfig = {
  mode: 'development',
  devtool: 'inline-source-map',
};

const stageConfig = {
  mode: 'development',
  devtool: 'source-map',
};

const prodConfig = {
  mode: 'production',
  devtool: 'source-map',
};

module.exports = env => {
  switch (env.NODE_ENV) {
    case 'staging':
      return merge(commonConfig, stageConfig);
    case 'production':
      return merge(commonConfig, prodConfig);
    default:
      return merge(commonConfig, devConfig);
  }
};
