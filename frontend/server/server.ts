import Storage = require('@google-cloud/storage');
import express = require('express');
import fs = require('fs');
import proxy = require('http-proxy-middleware');
import fetch from 'node-fetch';
import path = require('path');
import process = require('process');
import tmp = require('tmp');
import * as k8sHelper from './k8s-helper';
import proxyMiddleware from './proxy-middleware';
import * as Utils from './utils';

const app = express() as express.Application;

if (process.argv.length < 3) {
  console.error(`\
Usage: node server.js <static-dir> [port].
       You can specify the API server address using the
       ML_PIPELINE_SERVICE_HOST and ML_PIPELINE_SERVICE_PORT
       env vars.`);
  process.exit(1);
}

const currentDir = path.resolve(__dirname);
const buildDatePath = path.join(currentDir, 'BUILD_DATE');
const commitHashPath = path.join(currentDir, 'COMMIT_HASH');

const staticDir = path.resolve(process.argv[2]);
const buildDate =
    fs.existsSync(buildDatePath) ? fs.readFileSync(buildDatePath, 'utf-8').trim() : '';
const commitHash =
    fs.existsSync(commitHashPath) ? fs.readFileSync(commitHashPath, 'utf-8').trim() : '';
const port = process.argv[3] || 3000;
const apiServerHost = process.env.ML_PIPELINE_SERVICE_HOST || 'localhost';
const apiServerPort = process.env.ML_PIPELINE_SERVICE_PORT || '3001';
const apiServerAddress = `http://${apiServerHost}:${apiServerPort}`;

app.use(express.static(staticDir));

const v1alpha1Prefix = '/apis/v1alpha1';
const v1alpha2Prefix = '/apis/v1alpha2';

const healthzStats = {
  apiServerReady: false,
  buildDate,
  commitHash,
};

app.get(v1alpha1Prefix + '/healthz', (req, res) => {
  fetch(apiServerAddress + '/healthz', { timeout: 1000 })
    .then(() => healthzStats.apiServerReady = true)
    .catch(() => healthzStats.apiServerReady = false)
    .then(() => res.json(healthzStats));
});

app.get('/artifacts/list/*', async (req, res) => {
  const decodedPath = decodeURIComponent(req.params[0]);

  if (decodedPath.match('^gs://')) {
    const reqPath = decodedPath.substr('gs://'.length).split('/');
    const bucket = reqPath[0];
    const filepath = reqPath.slice(1).join('/');

    try {
      const storage = Storage();
      const results = await storage.bucket(bucket).getFiles({ prefix: filepath });
      res.send(results[0].map((f) => `gs://${bucket}/${f.name}`));
    } catch (err) {
      console.error('Error listing files:', err);
      res.status(500).send('Error: ' + err);
    }
  } else {
    res.status(404).send('Unsupported path.');
  }
});

app.get('/artifacts/get/*', async (req, res, next) => {
  const decodedPath = decodeURIComponent(req.params[0]);

  if (decodedPath.match('^gs://')) {
    const reqPath = decodedPath.substr('gs://'.length).split('/');
    const bucket = reqPath[0];
    const filename = reqPath.slice(1).join('/');
    const destFilename = tmp.tmpNameSync();

    try {
      const storage = Storage();
      await storage.bucket(bucket).file(filename).download({ destination: destFilename });
      console.log(`gs://${bucket}/${filename} downloaded to ${destFilename}.`);
      res.sendFile(destFilename, undefined, (err) => {
        if (err) {
          next(err);
        } else {
          fs.unlink(destFilename, (unlinkErr) => {
            if (unlinkErr) {
              console.error('Error deleting downloaded file: ' + unlinkErr);
            }
          });
        }
      });
    } catch (err) {
      console.error('Error getting file:', err);
      res.status(500).send('Failed to download file: ' + err);
    }
  } else {
    res.status(404).send('Unsupported path.');
  }

});

app.get('/apps/tensorboard', async (req, res) => {
  if (!k8sHelper.isInCluster) {
    res.status(500).send('Cannot talk to Kubernetes master');
    return;
  }
  const logdir = decodeURIComponent(req.query.logdir);
  if (!logdir) {
    res.status(404).send('logdir argument is required');
    return;
  }

  try {
    res.send(encodeURIComponent(await k8sHelper.getTensorboardAddress(logdir)));
  } catch (err) {
    res.status(500).send('Failed to list Tensorboard pods: ' + err);
  }
});

app.post('/apps/tensorboard', async (req, res) => {
  if (!k8sHelper.isInCluster) {
    res.status(500).send('Cannot talk to Kubernetes master');
    return;
  }
  const logdir = decodeURIComponent(req.query.logdir);
  if (!logdir) {
    res.status(404).send('logdir argument is required');
    return;
  }

  try {
    await k8sHelper.newTensorboardPod(logdir);
    const tensorboardAddress = await k8sHelper.waitForTensorboard(logdir, 60 * 1000);
    res.send(tensorboardAddress);
  } catch (err) {
    res.status(500).send('Failed to start Tensorboard app: ' + err);
  }

});

app.get('/k8s/pod/logs', async (req, res) => {
  if (!k8sHelper.isInCluster) {
    res.status(500).send('Cannot talk to Kubernetes master');
    return;
  }

  const podName = decodeURIComponent(req.query.podname);
  if (!podName) {
    res.status(404).send('podname argument is required');
    return;
  }

  try {
    res.send(await k8sHelper.getPodLogs(podName));
  } catch (err) {
    res.status(500).send('Could not get main container logs: ' + err);
  }
});

proxyMiddleware(app, v1alpha1Prefix);
proxyMiddleware(app, v1alpha2Prefix);

app.all(v1alpha1Prefix + '/*', proxy({
  changeOrigin: true,
  onProxyReq: (proxyReq, req, res) => {
    console.log('Proxied request: ', proxyReq.path);
  },
  target: apiServerAddress,
}));

app.all(v1alpha2Prefix + '/*', proxy({
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
