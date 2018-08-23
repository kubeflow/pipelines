// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as CopyWebpackPlugin from 'copy-webpack-plugin';
import * as path from 'path';
import * as webpack from 'webpack';
import 'webpack-dev-server';
import * as merge from 'webpack-merge';
import mockApiMiddleware from './mock-backend/mock-api-middleware';
import common from './webpack.common';

const config: webpack.Configuration = merge(common, {
  devServer: {
    before: mockApiMiddleware,
    compress: true,
    contentBase: path.join(__dirname),
    // Serve index.html for any unrecognized path
    historyApiFallback: true,
    overlay: true,
    port: 3000,
    stats: {
      errorDetails: true,
      errors: true,
      warnings: true,
    } as any,
  },
  devtool: 'inline-source-map',
  entry: {
    index: path.resolve(__dirname, 'test/components/index.ts'),
  },
  mode: 'development',
  plugins: [
    new CopyWebpackPlugin([{
      from: path.resolve(__dirname, 'index.html'),
      to: 'index.html',
    }, {
      from: path.resolve(__dirname, 'test/node_modules/mocha/*'),
      to: 'node_modules/mocha/[name].[ext]'
    }]),
  ],
});

// tslint:disable-next-line:no-default-export
export default config;
