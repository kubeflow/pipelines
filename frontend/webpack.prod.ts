import common from './webpack.common';
import * as merge from 'webpack-merge';
import * as MinifyPlugin from 'babel-minify-webpack-plugin';
import * as webpack from 'webpack';

export default merge(common, {
  plugins: [
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify('prod')
    }),
    new MinifyPlugin(),
    new webpack.optimize.AggressiveMergingPlugin(),
  ],
});
