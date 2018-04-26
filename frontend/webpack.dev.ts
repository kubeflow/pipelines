import * as path from 'path';
import * as webpack from 'webpack';
import * as merge from 'webpack-merge';
import mockApiMiddleware from './mock-backend/mock-api-middleware';
import common from './webpack.common';

export default merge(common, {
  devServer: {
    before: mockApiMiddleware,
    compress: true,
    contentBase: path.join(__dirname),
    // Serve index.html for any unrecognized path
    historyApiFallback: true,
    overlay: true,
    port: 3000,
  },
  devtool: 'inline-source-map',
  plugins: [
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify('dev'),
    }),
  ],
});
