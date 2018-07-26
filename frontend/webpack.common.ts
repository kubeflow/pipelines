import * as CleanWebpackPlugin from 'clean-webpack-plugin';
import * as CopyWebpackPlugin from 'copy-webpack-plugin';
import * as ExtractTextPlugin from 'extract-text-webpack-plugin';
import * as HtmlWebpackPlugin from 'html-webpack-plugin';
import * as path from 'path';

// tslint:disable-next-line:no-default-export
export default {
  entry: {
    index: path.resolve(__dirname, 'src/index.ts'),
  },
  module: {
    rules: [
      // Intentionally leaving out .js files, **typescriptify all the things!**
      {
        loader: 'polymer-webpack-loader',
        options: {
          processStyleLinks: true,
        },
        test: /\.html$/,
      },
      // This is for web component styles, which is needed by the polymer
      // loader above, in order to inline styles in their element templates.
      {
        include: [path.resolve(__dirname, 'src/components')],
        test: /\.css$/,
        use: 'css-loader',
      },
      // This is for all other style files, which are not web components.
      {
        exclude: [path.resolve(__dirname, 'src/components')],
        test: /\.css$/,
        use: ExtractTextPlugin.extract({
          use: [{
            loader: 'css-loader',
            options: {
              minimize: true,
              sourceMap: true,
            },
          }],
        }),
      },
      {
        enforce: 'pre',
        exclude: /bower_components/,
        loader: 'tslint-loader',
        options: {
          emitErrors: true,
        },
        test: /\.ts$/,
      },
      {
        loader: 'ts-loader',
        test: /\.ts$/,
      },
    ]
  },
  output: {
    filename: 'app.js',
    path: path.resolve(__dirname, 'dist')
  },
  plugins: [
    CopyWebpackPlugin([{
      from: path.resolve(__dirname, 'bower_components/webcomponentsjs/*.js'),
      to: 'bower_components/webcomponentsjs/[name].[ext]'
    }, {
      from: path.resolve(__dirname, 'index.html'),
      to: 'index.html',
    }]),
    new CleanWebpackPlugin(['dist']),
    new ExtractTextPlugin('styles.css'),
  ],
  resolve: {
    extensions: ['.ts', '.tsx', '.js', '.jsx', '.html'],
    modules: [
      path.resolve(__dirname, 'bower_components'),
      path.resolve(__dirname, 'node_modules'),
    ],
  },
};
