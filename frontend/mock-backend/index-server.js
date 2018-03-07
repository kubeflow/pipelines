const fs = require('fs');
const proxy = require('http-proxy-middleware');

const prefix = __dirname + '/pipeline-data';

module.exports = (app) => {

  app.use((req, res, next) => {
    if (req.url.startsWith('/_ftp/')) {
      const path = prefix + req.url.substr('/_ftp'.length);
      if (fs.lstatSync(path).isDirectory()) {
        const files = fs.readdirSync(path).map((f) => ({
          isDirectory: fs.lstatSync(path + '/' + f).isDirectory(),
          name: f,
        }));
        res.json(files);
      } else {
        res.send(fs.readFileSync(path, 'utf8'));
        return;
      }
    }
    next();
  });

  app.all('/_api/*', proxy({
    changeOrigin: true,
    pathRewrite: { '^/_api/': '/' },
    target: 'http://localhost:3001',
  }));

};
