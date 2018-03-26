import Storage = require('@google-cloud/storage');
import express = require('express');
import fs = require('fs');
import proxy = require('http-proxy-middleware');
import path = require('path');
import process = require('process');
import tmp = require('tmp');

const app = express() as express.Application;

if (process.argv.length < 3) {
  console.error(`\
Usage: node server.js <static-dir> [port].
       You can specify the API server address using the
       ML_PIPELINE_MANAGER_SERVICE_HOST and ML_PIPELINE_MANAGER_SERVICE_PORT
       env vars.`);
  process.exit(1);
}

const staticDir = path.resolve(process.argv[2]);
const port = process.argv[3] || 3000;
const apiServerHost = process.env.ML_PIPELINE_MANAGER_SERVICE_HOST || 'localhost';
const apiServerPort = process.env.ML_PIPELINE_MANAGER_SERVICE_PORT || '3001';
const apiServerAddress = `http://${apiServerHost}:${apiServerPort}`;

app.use(express.static(staticDir));

const apisPrefix = '/apis/v1alpha1';

app.get(apisPrefix + '/artifact/list/*', (req, res) => {
  if (!req.params) {
    res.status(404).send('Error: No path provided.');
    return;
  }

  const storage = Storage();

  const decodedPath = decodeURIComponent(req.params[0]);

  if (decodedPath.match('^gs://')) {
    const reqPath = decodedPath.substr('gs://'.length).split('/');
    const bucket = reqPath[0];
    const filepath = reqPath.slice(1).join('/');

    storage
      .bucket(bucket)
      .getFiles({prefix: filepath})
      .then((results) => res.send(results[0].map((f) => f.name)))
      .catch((err) => {
        console.error('Error listing files:', err);
        res.status(500).send('Error: ' + err);
      });
  } else {
    res.status(404).send('Error: Unsupported path.');
  }
});

app.get(apisPrefix + '/artifact/get/*', (req, res, next) => {
  if (!req.params) {
    res.status(404).send('Error: No path provided.');
    return;
  }

  const storage = Storage();

  const decodedPath = decodeURIComponent(req.params[0]);

  if (decodedPath.match('^gs://')) {
    const reqPath = decodedPath.substr('gs://'.length).split('/');
    const bucket = reqPath[0];
    const filename = reqPath.slice(1).join('/');
    const destFilename = tmp.tmpNameSync();

    storage
      .bucket(bucket)
      .file(filename)
      .download({ destination: destFilename })
      .then(() => {
        console.log(`gs://${bucket}/${filename} downloaded to ${destFilename}.`);
        res.sendFile(destFilename, undefined, (err) => {
          if (err) {
            next(err);
          } else {
            fs.unlink(destFilename, null);
          }
        });
      })
      .catch((err) => {
        console.error('Error getting file:', err);
        res.status(500).send('Error: ' + err);
      });
  } else {
    res.status(404).send('Error: Unsupported path.');
  }

});

app.all(apisPrefix + '/*', proxy({
  changeOrigin: true,
  onProxyReq: (proxyReq, req, res) => {
    console.log('Proxied request: ', proxyReq.path);
  },
  target: apiServerAddress,
}));

app.get('*', (req, res) => {
  // TODO: look into caching this file to speed up multiple requests.
  res.sendFile(path.resolve(staticDir, 'index.html'));
});

app.listen(port, () => {
  console.log('Server listening at http://localhost:' + port);
});
