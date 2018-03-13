import express = require('express');
import fs = require('fs');
import os = require('os');
import path = require('path');
import process = require('process');
import proxy = require('http-proxy-middleware');
import { Storage } from '@google-cloud/storage';

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

app.get('/_config/apiServerAddress', (req, res) => {
  res.send(apiServerAddress);
});

app.get('/_api/artifact/:path', (req, res) => {
  const storage = new Storage();

  if (!req.params.path) {
    console.error('No path provided. Aborting..');
    return;
  }

  if (req.params.path.startsWith('gs://')) {
    const reqPath = req.params.path.substr('gs://'.length).split('/');
    const bucket = reqPath[0];
    const filename = path.join(...reqPath.slice(1));
    const destFilename = path.join(os.tmpdir(), Math.floor(Math.random() * 1000000).toString());

    // Downloads the file
    storage
      .bucket(bucket)
      .file(filename)
      .download({ destination: destFilename })
      .then(() => {
        console.log(`gs://${bucket}/${filename} downloaded to ${destFilename}.`);
        const contents = fs.readFileSync(destFilename, { encoding: 'utf-8', flag: 'r' });
        res.sendFile(destFilename);
      })
      .catch(err => {
        console.error('Error getting file:', err);
        res.status(500).send('Error: ' + err);
      });
  }

});

app.all('/_api/*', proxy({
  changeOrigin: true,
  onProxyReq: (proxyReq, req, res) => {
    console.log('Proxied request: ', proxyReq.path);
  },
  pathRewrite: { '^/_api/': '/apis/v1alpha1/' },
  target: apiServerAddress,
}));

app.get('*', (req, res) => {
  res.sendFile(path.resolve(staticDir, 'index.html'));
});

app.listen(port, () => {
  console.log('Server listening at http://localhost:' + port);
});
