const express = require('express');
const jsonServer = require('json-server');

const app = express();

const defaultPage = 'templates';

// Redirect '/' to 'templates'
app.get('/', (_, res) => {
  console.log('hi');
  res.redirect(`/${defaultPage}`);
});

// Static resources
[
  '/styles.css',
  '/bower_components/webcomponentsjs/webcomponents-loader.js',
  '/bower_components/webcomponentsjs/webcomponents-hi-sd-ce.js',
  '/app.js'
].forEach(resource => {
  app.get(resource, (_, res) => {
    res.sendFile(resource, options);
  })
});

[
  `/${defaultPage}*`
].forEach(path => {
  app.get(path, (req, res) => {
    res.sendFile('index.html', options, err => {
      console.log(err ? err : 'requested: ', req.url);
    });
  });
})

const options = {
  root: __dirname + '/dist/',
  dotfiles: 'deny',
  headers: {
      'x-timestamp': Date.now(),
      'x-sent': true
  }
};

// Add mock backend middleware
const router = jsonServer.router('mock-backend/mock-db.json');
const staticsMiddleware = jsonServer.defaults();

app.use(staticsMiddleware);
app.use(router);

// app.listen(3000, () => console.log('listening on :3000'));

module.exports = app;
