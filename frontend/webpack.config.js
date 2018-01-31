const CopyWebpackPlugin = require('copy-webpack-plugin');
const CleanWebpackPlugin = require('clean-webpack-plugin');
const jsonServer = require('json-server')
const path = require('path');

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
});

module.exports = {
  entry: path.resolve(__dirname, 'src/index.ts'),
  output: {
    filename: 'app.js',
    path: path.resolve(__dirname, 'dist')
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
      }
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
    }])
  ]
};
