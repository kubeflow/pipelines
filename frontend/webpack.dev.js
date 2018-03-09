const common = require('./webpack.common.js');
const mockApiMiddleware = require('./mock-backend/mock-api-middleware');
const merge = require('webpack-merge');
const path = require('path');
const webpack = require('webpack');

module.exports = merge(common, {
  devtool: 'inline-source-map',
  devServer: {
    before: mockApiMiddleware,
    contentBase: path.join(__dirname),
    compress: true,
    overlay: true,
    port: 3000,
    // Serve index.html for any unrecognized path
    historyApiFallback: true,
  },
  plugins: [
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify('dev')
    }),
  ],
});
