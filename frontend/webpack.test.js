const Clean = require('clean-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const ExtractTextPlugin = require('extract-text-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const path = require('path');
const setupServerMockup = require('./mock-backend/server');

const modulesToTest = require('./test/modules');
const entries = {};
const plugins = [
  new CopyWebpackPlugin([{
    from: path.resolve(__dirname, 'bower_components/webcomponentsjs/*.js'),
    to: 'bower_components/webcomponentsjs/[name].[ext]'
  }]),
  new CopyWebpackPlugin([{
    from: path.resolve(__dirname, 'bower_components/web-component-tester/*.js'),
    to: 'bower_components/web-component-tester/[name].[ext]'
  }]),
  new Clean(['test/dist']),
];

modulesToTest.forEach((e) => {
  entries[e.module] = path.resolve(__dirname, e.dir, e.module + '-test');
  plugins.push(new HtmlWebpackPlugin({
    inject: true,
    template: path.resolve(__dirname, 'test/test.ejs'),
    chunks: [e.module],
    filename: e.module + '-test.html'
  }));
});

module.exports = {
  entry: entries,
  output: {
    filename: '[name]_test.js',
    path: path.resolve(__dirname, 'test/dist')
  },
  resolve: {
    modules: [
      path.resolve(__dirname, 'bower_components'),
      path.resolve(__dirname, 'node_modules'),

    ],
    extensions: ['.ts', '.tsx', '.js', '.jsx', '.html']
  },
  externals: {
    'sinon': 'sinon',
  },
  module: {
    rules: [
      {
        test: /\.html$/,
        loader: 'polymer-webpack-loader',
        options: {
          processStyleLinks: true,
        },
      },
      // This is for web component styles, which is needed by the polymer
      // loader above, in order to inline styles in their element templates.
      {
        test: /\.css$/,
        include: [path.resolve(__dirname, 'src/components')],
        use: 'css-loader',
      },
      // This is for all other style files, which are not web components.
      {
        test: /\.css$/,
        exclude: [path.resolve(__dirname, 'src/components')],
        use: ExtractTextPlugin.extract({
          use: [{
            loader: 'css-loader',
            options: {
              sourceMap: true,
              minimize: true
            },
          }],
        }),
      },
      {
        test: /\.ts$/,
        use: 'ts-loader',
      },
    ]
  },
  plugins,
  devServer: {
    contentBase: path.join(__dirname),
    compress: true,
    overlay: true,
    port: 3000,
    before: setupServerMockup,
    // Serve index.html for any unrecognized path
    historyApiFallback: true,
  },
};
