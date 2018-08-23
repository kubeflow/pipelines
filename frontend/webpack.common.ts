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

import * as CleanWebpackPlugin from 'clean-webpack-plugin';
import * as CopyWebpackPlugin from 'copy-webpack-plugin';
import * as MiniCssExtractPlugin from 'mini-css-extract-plugin';
import * as path from 'path';
import * as webpack from 'webpack';

const config: webpack.Configuration = {
  entry: {
    index: path.resolve(__dirname, 'src/index.ts'),
  },
  module: {
    rules: [
      // Intentionally leaving out .js files, **typescriptify all the things!**
      {
        loader: 'polymer-webpack-loader',
        options: {
          processStyleLinks: true,
        },
        test: /\.html$/,
      },
      // This is for web component styles, which is needed by the polymer
      // loader above, in order to inline styles in their element templates.
      {
        include: [path.resolve(__dirname, 'src/components')],
        test: /\.css$/,
        use: 'css-loader',
      },
      // This is for all other style files, which are not web components.
      {
        exclude: [path.resolve(__dirname, 'src/components')],
        test: /\.css$/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader as any,
            options: {
              minimize: true,
              sourceMap: true,
            },
          },
          'css-loader',
        ],
      },
      {
        enforce: 'pre',
        exclude: path.resolve(__dirname, 'bower_components'),
        loader: 'tslint-loader',
        options: {
          emitErrors: true,
        },
        test: /\.ts$/,
      },
      {
        include: path.resolve(__dirname),
        loader: 'ts-loader',
        options: {
          configFile: path.resolve(__dirname, 'tsconfig.json'),
          instance: 'src',
        },
        test: /\.ts$/,
      },
    ]
  },
  output: {
    filename: 'app.js',
    path: path.resolve(__dirname, 'dist')
  },
  plugins: [
    new CopyWebpackPlugin([{
      from: path.resolve(__dirname, 'bower_components/webcomponentsjs/*.js'),
      to: 'bower_components/webcomponentsjs/[name].[ext]'
    }, {
      from: path.resolve(__dirname, 'index.html'),
      to: 'index.html',
    }]),
    new CleanWebpackPlugin(['dist']),
    new MiniCssExtractPlugin({filename: 'styles.css'}),
  ],
  resolve: {
    extensions: ['.ts', '.tsx', '.js', '.jsx', '.html'],
    modules: [
      path.resolve(__dirname, 'bower_components'),
      path.resolve(__dirname, 'node_modules'),
    ],
  },
};

// tslint:disable-next-line:no-default-export
export default config;
