import proxyMiddleware from './proxy-middleware';
import Storage = require('@google-cloud/storage');
import express = require('express');
import fs = require('fs');
import proxy = require('http-proxy-middleware');
import path = require('path');
import process = require('process');
import tmp = require('tmp');
import * as Utils from './utils';
const k8s = require('@kubernetes/client-node');

const app = express() as express.Application;

// Get the current pod's namespace, which is written by k8s at this path:
const namespaceFilePath = '/var/run/secrets/kubernetes.io/serviceaccount/namespace';
let namespace = null;
let k8sV1Client = null;
let k8sV1Beta1Client = null;

if (fs.existsSync(namespaceFilePath)) {
  namespace = fs.readFileSync(namespaceFilePath);

  // Get the k8s client object. We need both the v1 API client, as well as the
  // v1beta1 client. The first is needed for all base operations, the second to
  // provide extensions, such as creating deployments.
  // TODO: extract this into a separate k8s APIs wrapper.
  k8sV1Client = k8s.Config.defaultClient();
  const k8sHost = process.env.KUBERNETES_SERVICE_HOST;
  const k8sPort = process.env.KUBERNETES_SERVICE_PORT;
  const caCert = fs.readFileSync(k8s.Config.SERVICEACCOUNT_CA_PATH);
  const token = fs.readFileSync(k8s.Config.SERVICEACCOUNT_TOKEN_PATH);
  k8sV1Beta1Client = new k8s.Extensions_v1beta1Api(`https://${k8sHost}:${k8sPort}`);
  k8sV1Beta1Client.setDefaultAuthentication({
    'applyToRequest': (opts) => {
      opts.ca = caCert;
      opts.headers['Authorization'] = 'Bearer ' + token;
    }
  });
}

if (process.argv.length < 3) {
  console.error(`\
Usage: node server.js <static-dir> [port].
       You can specify the API server address using the
       ML_PIPELINE_SERVICE_HOST and ML_PIPELINE_SERVICE_PORT
       env vars.`);
  process.exit(1);
}

const staticDir = path.resolve(process.argv[2]);
const port = process.argv[3] || 3000;
const apiServerHost = process.env.ML_PIPELINE_SERVICE_HOST || 'localhost';
const apiServerPort = process.env.ML_PIPELINE_SERVICE_PORT || '3001';
const apiServerAddress = `http://${apiServerHost}:${apiServerPort}`;

app.use(express.static(staticDir));

const apisPrefix = '/apis/v1alpha1';

app.get(apisPrefix + '/artifacts/list/*', async (req, res) => {
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

    try {
      const results = await storage.bucket(bucket).getFiles({ prefix: filepath });
      res.send(results[0].map((f) => `gs://${bucket}/${f.name}`));
    } catch (err) {
      console.error('Error listing files:', err);
      res.status(500).send('Error: ' + err);
    }
  } else {
    res.status(404).send('Error: Unsupported path.');
  }
});

app.get(apisPrefix + '/artifacts/get/*', async (req, res, next) => {
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

    try {
      await storage.bucket(bucket).file(filename).download({ destination: destFilename });
      console.log(`gs://${bucket}/${filename} downloaded to ${destFilename}.`);
      res.sendFile(destFilename, undefined, (err) => {
        if (err) {
          next(err);
        } else {
          fs.unlink(destFilename, (unlinkErr) => {
            console.error('Error deleting downloaded file: ' + unlinkErr);
          });
        }
      });
    } catch (err) {
      console.error('Error getting file:', err);
      res.status(500).send('Error: ' + err);
    };
  } else {
    res.status(404).send('Error: Unsupported path.');
  }

});

app.get(apisPrefix + '/apps/tensorboard', async (req, res) => {
  if (!k8sV1Client) {
    res.status(500).send('Cannot talk to Kubernetes master');
    return;
  }
  const logdir = decodeURIComponent(req.query.logdir);
  if (!logdir) {
    res.status(404).send('logdir argument is required');
    return;
  }

  try {
    const response = await k8sV1Client.listNamespacedPod(namespace);
    const items = response.body.items;
    const args = ['tensorboard', '--logdir', logdir];
    const tensorboard = items.find(i =>
      i.spec.containers.find(c => Utils.equalArrays(c.args, args)));
    res.send(tensorboard && tensorboard.status ?
      encodeURIComponent(`http://${tensorboard.status.podIP}:6006`) :
      ''
    );
  } catch (err) {
    res.status(404).send(err);
  }
});

app.post(apisPrefix + '/apps/tensorboard', async (req, res) => {
  if (!k8sV1Beta1Client) {
    res.status(500).send('Cannot talk to Kubernetes master');
    return;
  }
  const logdir = decodeURIComponent(req.query.logdir);
  if (!logdir) {
    res.status(404).send('logdir argument is required');
    return;
  }

  const randomSuffix = Utils.generateRandomString(15);
  const deploymentName = 'tensorboard-' + randomSuffix;
  const serviceName = 'tensorboard-service-' + randomSuffix;

  // TODO: take the configurations below to a separate file
  const tbDeployment = {
    kind: 'Deployment',
    metadata: {
      name: deploymentName,
    },
    spec: {
      selector: {
        matchLabels: {
          app: deploymentName,
        },
      },
      template: {
        metadata: {
          labels: {
            app: deploymentName,
          },
        },
        spec: {
          containers: [{
            args: [
              'tensorboard',
              '--logdir',
              logdir,
            ],
            name: 'tensorflow',
            image: 'tensorflow/tensorflow',
            ports: [{
              containerPort: 6006,
            }],
          }],
        },
      },
    },
  };
  const tbService = {
    kind: 'Service',
    metadata: {
      name: serviceName,
    },
    spec: {
      selector: {
        app: deploymentName,
      },
      ports: [{
        protocol: 'TCP',
        port: 6006,
        targetPort: 6006,
      }],
    },
    type: 'ClusterIP',
  };

  try {
    await k8sV1Beta1Client.createNamespacedDeployment(namespace, tbDeployment);
    await k8sV1Client.createNamespacedService(namespace, tbService);
    res.send('Started deployment and service');
  } catch (err) {
    res.status(500).send('Error starting app: ' + err);
  }

});

proxyMiddleware(app, apisPrefix);

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
