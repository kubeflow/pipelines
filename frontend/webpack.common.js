const CleanWebpackPlugin = require('clean-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const ExtractTextPlugin = require('extract-text-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const path = require('path');

module.exports = {
  entry: {
    index: path.resolve(__dirname, 'src/index.ts'),
  },
  output: {
    filename: 'app.js',
    path: path.resolve(__dirname, 'dist')
  },
  resolve: {
    modules: [
      path.resolve(__dirname, 'bower_components'),
      path.resolve(__dirname, 'node_modules'),
    ],
    extensions: ['.ts', '.tsx', '.js', '.jsx', '.html']
  },
  module: {
    rules: [
      // Intentionally leaving out .js files, **typescriptify all the things!**
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
        enforce: 'pre',
        loader: 'tslint-loader',
        options: {
          configFile: 'tslint.json',
        },
        exclude: /bower_components/,
      },
      {
        test: /\.ts$/,
        loader: 'ts-loader',
      },
    ]
  },
  plugins: [
    new CopyWebpackPlugin([{
      from: path.resolve(__dirname, 'bower_components/webcomponentsjs/*.js'),
      to: 'bower_components/webcomponentsjs/[name].[ext]'
    }, {
      from: path.resolve(__dirname, 'index.html'),
      to: 'index.html',
    }]),
    new CleanWebpackPlugin(['dist']),
    new ExtractTextPlugin('styles.css'),
  ]
};
