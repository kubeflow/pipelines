const common = require('./webpack.common.js');
const indexServer = require('./mock-backend/index-server');
const merge = require('webpack-merge');
const path = require('path');

module.exports = merge(common, {
  devtool: 'inline-source-map',
  devServer: {
    before: indexServer,
    contentBase: path.join(__dirname),
    compress: true,
    overlay: true,
    port: 3000,
    proxy: {
      '/_api': {
        target: 'http://localhost:3001',
        pathRewrite: {'^/_api': ''},
      },
    },
    // Serve index.html for any unrecognized path
    historyApiFallback: true,
  },
});
