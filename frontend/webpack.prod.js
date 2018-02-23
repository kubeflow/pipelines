const common = require('./webpack.common.js');
const merge = require('webpack-merge');
const MinifyPlugin = require('babel-minify-webpack-plugin');
const webpack = require('webpack');

module.exports = merge(common, {
  plugins: [
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify('prod')
    }),
    new MinifyPlugin(),
    new webpack.optimize.AggressiveMergingPlugin(),
  ],
});
