// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
import { Client as MinioClient } from 'minio';
import * as tar from 'tar';

// The minio endpoint, port, access and secret keys are hardcoded to the same
// values used in the deployment.
const minioClient = new MinioClient({
  accessKey: 'minio',
  endPoint: 'minio-service.default',
  port: 9000,
  secretKey: 'minio123',
  useSSL: false,
} as any);

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

const v1alpha2Prefix = '/apis/v1alpha2';

const healthzStats = {
  apiServerReady: false,
  buildDate,
  commitHash,
};

app.get(v1alpha2Prefix + '/healthz', (req, res) => {
  fetch(apiServerAddress + '/healthz', { timeout: 1000 })
    .then(() => healthzStats.apiServerReady = true)
    .catch(() => healthzStats.apiServerReady = false)
    .then(() => res.json(healthzStats));
});

app.get('/artifacts/get', async (req, res) => {
  const [source, bucket, encodedKey] = [req.query.source, req.query.bucket, req.query.key];
  if (!source) {
    res.status(500).send('Storage source is missing from artifact request');
    return;
  }
  if (!bucket) {
    res.status(500).send('Storage bucket is missing from artifact request');
    return;
  }
  if (!encodedKey) {
    res.status(500).send('Storage key is missing from artifact request');
    return;
  }
  const key = decodeURIComponent(encodedKey);
  console.log(`Getting storage artifact at: ${source}: ${bucket}/${key}`);
  switch (source) {
    case 'gcs':
      try {
        const storage = new Storage();
        const destFilename = tmp.tmpNameSync();
        await storage.bucket(bucket).file(key).download({ destination: destFilename });
        console.log(`gs://${bucket}/${key} downloaded to ${destFilename}.`);
        res.sendFile(destFilename, undefined, (err) => {
          if (err) {
            console.log('Error sending file: ' + err);
          } else {
            fs.unlink(destFilename, (unlinkErr) => {
              if (unlinkErr) {
                console.error('Error deleting downloaded file: ' + unlinkErr);
              }
            });
          }
        });
      } catch (err) {
        res.status(500).send('Failed to download GCS file. Error: ' + err);
      }
      break;
    case 'minio':
      minioClient.getObject(bucket, key, (err, stream) => {
        if (err) {
          res.status(500).send(`Failed to get object in bucket ${bucket} at path ${key}: ${err}`);
          return;
        }

        try {
          let contents = '';
          stream.pipe(new tar.Parse()).on('entry', (entry) => {
            entry.on('data', (buffer) => contents += buffer.toString());
          });

          stream.on('end', () => {
            res.send(contents);
          });
        } catch (err) {
          res.status(500).send(`Failed to get object in bucket ${bucket} at path ${key}: ${err}`);
        }
      });
      break;
    default:
      res.status(500).send('Unknown storage source: ' + source);
      return;
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

proxyMiddleware(app, v1alpha2Prefix);

app.all(v1alpha2Prefix + '/*', proxy({
  changeOrigin: true,
  onProxyReq: (proxyReq) => {
    console.log('Proxied request: ', (proxyReq as any).path);
  },
  target: apiServerAddress,
}));

app.all(v1alpha2Prefix + '/*', proxy({
  changeOrigin: true,
  onProxyReq: (proxyReq) => {
    console.log('Proxied request: ', (proxyReq as any).path);
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
