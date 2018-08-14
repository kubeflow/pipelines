import * as MinifyPlugin from 'babel-minify-webpack-plugin';
import * as webpack from 'webpack';
import * as merge from 'webpack-merge';
import common from './webpack.common';

const config: webpack.Configuration = merge(common, {
  mode: 'production',
  plugins: [
    new MinifyPlugin(),
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify('prod')
    }),
    new webpack.optimize.AggressiveMergingPlugin(),
  ],
});

// tslint:disable-next-line:no-default-export
export default config;
