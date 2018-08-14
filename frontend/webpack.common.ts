import * as CleanWebpackPlugin from 'clean-webpack-plugin';
import * as CopyWebpackPlugin from 'copy-webpack-plugin';
import * as HtmlWebpackPlugin from 'html-webpack-plugin';
import * as MiniCssExtractPlugin from 'mini-css-extract-plugin';
import * as path from 'path';
import * as webpack from 'webpack';

const config: webpack.Configuration = {
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
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
            options: {
              minimize: true,
              sourceMap: true,
            },
          },
          'css-loader',
        ],
      },
      {
        enforce: 'pre',
        exclude: path.resolve(__dirname, 'bower_components'),
        loader: 'tslint-loader',
        options: {
          emitErrors: true,
        },
        test: /\.ts$/,
      },
      {
        include: path.resolve(__dirname),
        loader: 'ts-loader',
        options: {
          configFile: path.resolve(__dirname, 'tsconfig.json'),
          instance: 'src',
        },
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
    new MiniCssExtractPlugin({filename: 'styles.css'}),
  ],
  resolve: {
    extensions: ['.ts', '.tsx', '.js', '.jsx', '.html'],
    modules: [
      path.resolve(__dirname, 'bower_components'),
      path.resolve(__dirname, 'node_modules'),
    ],
  },
};

// tslint:disable-next-line:no-default-export
export default config;
