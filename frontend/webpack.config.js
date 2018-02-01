const CopyWebpackPlugin = require('copy-webpack-plugin');
const CleanWebpackPlugin = require('clean-webpack-plugin');
const ExtractTextPlugin = require('extract-text-webpack-plugin');
const jsonServer = require('json-server')
const path = require('path');
const webpack = require('webpack');

// Run mock backend server
const server = jsonServer.create();
const router = jsonServer.router('mock-backend/mock-db.json');
const staticsMiddleware = jsonServer.defaults({
  static: './dist',
});

server.use(staticsMiddleware);
server.use(router);
server.listen(3000, () => {
  console.log('Mock backend server is running at :3000')
});

module.exports = {
  entry: {
    app: path.resolve(__dirname, 'src/index.ts'),
    polymer: ["polymer"],
  },
  output: {
    filename: '[name].js',
    path: path.resolve(__dirname, 'dist'),
    publicPath: '/',
  },
  devtool: 'inline-source-map',
  resolve: {
    extensions: ['.ts', '.js'],
    modules: [
      path.resolve(__dirname, 'node_modules'),
      path.resolve(__dirname, 'bower_components')
    ]
  },
  module: {
    loaders: [
      {
        test: /\.ts$/,
        loader: 'ts-loader',
        exclude: /node_modules/
      },
      {
        test: /\.html$/,
        use: [
          { loader: 'wc-loader', options: { root: '/' } },
        ]
      },
      {
        test: /\.css$/,
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
    ],
  },
  plugins: [
    new CleanWebpackPlugin(['dist'], { verbose: true, root: path.resolve(__dirname) }),
    new CopyWebpackPlugin([{
      from: path.resolve(__dirname, 'bower_components/webcomponentsjs/*.js'),
      to: 'bower_components/webcomponentsjs/[name].[ext]',
    }, {
      from: path.resolve(__dirname, 'index.html'),
      to: 'index.html',
    }]),
    new webpack.optimize.CommonsChunkPlugin({
      name: 'polymer'
    }),
    new ExtractTextPlugin('styles.css'),
  ]
};
