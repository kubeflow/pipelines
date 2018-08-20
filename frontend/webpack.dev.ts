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
