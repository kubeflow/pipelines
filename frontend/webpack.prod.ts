// tslint:disable-next-line:no-var-requires
const minifyPlugin = require('babel-minify-webpack-plugin');
import * as webpack from 'webpack';
import * as merge from 'webpack-merge';
import common from './webpack.common';

const config: webpack.Configuration = merge(common, {
  mode: 'production',
  plugins: [
    new minifyPlugin(),
    new webpack.optimize.AggressiveMergingPlugin(),
  ],
});

// tslint:disable-next-line:no-default-export
export default config;
