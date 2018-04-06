import common from './webpack.common';
import mockApiMiddleware from './mock-backend/mock-api-middleware';
import * as merge from 'webpack-merge';
import * as path from 'path';
import * as webpack from 'webpack';

export default merge(common, {
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
