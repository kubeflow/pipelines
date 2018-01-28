const jsonServer = require('json-server')
const path = require('path');
const webpack = require('webpack');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const CleanWebpackPlugin = require('clean-webpack-plugin');

// Run mock backend server
const server = jsonServer.create();
const router = jsonServer.router('mock-backend/mock-db.json');
const middlewares = jsonServer.defaults({
  static: './dist',
});

server.use(middlewares);
server.use(router);
server.listen(3000, () => {
  console.log('Mock backend server is running at :3000')
})

module.exports = {
  entry: {
    app: './src/index'
  },
  devtool: 'inline-source-map',
  devServer: {
    contentBase: path.resolve(__dirname, 'dist'),
  },
  output: {
    filename: '[name].js',
    path: path.resolve(__dirname, 'dist')
  },
  module: {
    rules: [
      {
        test: /\.(html)$/,
        use: {
          loader: 'html-loader'
        }
      },
      {
        test: /\.ts?$/,
        use: 'ts-loader',
        exclude: /node_modules/
      }
    ]
  },
  resolve: {
    extensions: ['.ts', '.js']
  },
  plugins: [
    new CleanWebpackPlugin(['dist'], { verbose: true, root: path.resolve(__dirname) }),
    new HtmlWebpackPlugin({
      template: './index.html'
    }),
  ]
};