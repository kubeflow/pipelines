import * as fs from 'fs';
import * as _path from 'path';
import proxyMiddleware from '../server/proxy-middleware';

const prefix = __dirname + '/pipeline-data';

const fixedData = require('./fixed-data').data;

const rocMetadataJsonPath = './eval-output/metadata.json';
const rocDataPath = './eval-output/roc.csv';

const confusionMatrixMetadataJsonPath = './model-output/metadata.json';
const confusionMatrixPath = './model-output/confusion_matrix.csv';

const apisPrefix = '/apis/v1alpha1';

let tensorboardPod = '';

let apiServerReady = false;

// Simulate API server not ready for 5 seconds
setTimeout(() => {
  apiServerReady = true;
}, 5000);

export default (app) => {

  app.set('json spaces', 2);

  app.get(apisPrefix + '/healthz', (req, res) => {
    if (apiServerReady) {
      res.send({ apiServerReady });
    } else {
      res.status(404).send();
    }
  });

  app.get(apisPrefix + '/pipelines', (req, res) => {
    if (!apiServerReady) {
      res.status(404).send();
      return;
    }

    res.header('Content-Type', 'application/json');
    if (req.query && req.query.filter) {
      // NOTE: We do not mock fuzzy matching. E.g. 'ee' doesn't match 'Pipeline'
      // This may need to be updated when the backend implements filtering.
      res.json(fixedData.pipelines.filter((p) => p.name.toLocaleLowerCase().match(
          decodeURIComponent(req.query.filter).toLocaleLowerCase())));
    } else {
      res.json(fixedData.pipelines);
    }
  });

  app.all(apisPrefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    switch (req.method) {
      case 'DELETE':
        const i = fixedData.pipelines.findIndex((p) => p.id === pid);
        if (fixedData.pipelines[i].name.startsWith('Cannot be deleted')) {
          res.status(502).send(`Deletion failed for Pipeline: '${fixedData.pipelines[i].name}'`);
        } else {
          // Delete the Pipeline from fixedData.
          fixedData.pipelines.splice(i, 1);
          res.send('ok');
        }
        break;
      case 'GET':
        const p = fixedData.pipelines.filter((p) => p.id === pid);
        res.json(p[0]);
        break;
      default:
        res.status(405).send('Unsupported request type: ' + res.method);
    }
  });

  app.get(apisPrefix + '/pipelines/:pid/jobs', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    let jobs = fixedData.pipelines.filter((p) => p.id === pid)[0].jobs;
    if (req.query && req.query.filter) {
      // NOTE: We do not mock fuzzy matching. E.g. 'ee' doesn't match 'Pipeline'
      // This may need to be updated when the backend implements filtering.
      jobs = jobs.filter((j) => j.metadata.name.toLocaleLowerCase().match(
          decodeURIComponent(req.query.filter).toLocaleLowerCase()));
    }
    res.json(jobs.map((j) => j.metadata));
  });

  app.post(apisPrefix + '/pipelines/:pid/enable', (req, res) => {
    setTimeout(() => {
      const pid = Number.parseInt(req.params.pid);
      const pipeline = fixedData.pipelines.find((p) => p.id === pid);
      pipeline.enabled = true;
      res.send('ok');
    }, 1000);
  });

  app.post(apisPrefix + '/pipelines/:pid/disable', (req, res) => {
    setTimeout(() => {
      const pid = Number.parseInt(req.params.pid);
      const pipeline = fixedData.pipelines.find((p) => p.id === pid);
      pipeline.enabled = false;
      res.send('ok');
    }, 1000);
  });

  app.get(apisPrefix + '/pipelines/:pid/jobs/:jname', (req, res) => {
    const pid = Number.parseInt(req.params.pid);
    const jname = req.params.jname;
    const pipeline = fixedData.pipelines.find((p) => p.id === pid);
    const job = pipeline.jobs.find((j) => j.metadata.name === jname);
    if (!job) {
      res.status(404).send('Cannot find a job with name: ' + jname);
      return;
    }
    res.json(job.jobDetail);
  });

  app.get(apisPrefix + '/packages', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages);
  });

  app.get(apisPrefix + '/packages/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    res.send(fs.readFileSync('./mock-backend/mock-template.yaml'));
  });

  app.post(apisPrefix + '/packages/upload', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages[0]);
  });

  app.get(apisPrefix + '/artifacts/list/:path', (req, res) => {

    const path = decodeURIComponent(req.params.path);

    res.header('Content-Type', 'application/json');
    res.json([
      path + '/file1',
      path + '/file2',
      path + (path.match('analysis$|model$') ? '/metadata.json' : '/file3'),
    ]);
  });

  app.get(apisPrefix + '/artifacts/get/:path', (req, res) => {
    res.header('Content-Type', 'application/json');
    const path = decodeURIComponent(req.params.path);
    if (path.endsWith('roc.csv')) {
      res.sendFile(_path.resolve(__dirname, rocDataPath));
    } else if (path.endsWith('confusion_matrix.csv')) {
      res.sendFile(_path.resolve(__dirname, confusionMatrixPath));
    } else if (path.endsWith('analysis/metadata.json')) {
      res.sendFile(_path.resolve(__dirname, confusionMatrixMetadataJsonPath));
    } else if (path.endsWith('model/metadata.json')) {
      res.sendFile(_path.resolve(__dirname, rocMetadataJsonPath));
    } else {
      res.send('dummy file');
    }
  });

  app.get(apisPrefix + '/apps/tensorboard', (req, res) => {
    res.send(tensorboardPod);
  });

  app.post(apisPrefix + '/apps/tensorboard', (req, res) => {
    tensorboardPod = 'http://tensorboardserver:port';
    setTimeout(() => {
      res.send('ok');
    }, 1000);
  });

  app.get('/_componentTests*', (req, res) => {
    res.sendFile(_path.resolve('test', 'components', 'index.html'));
  });

  app.all(apisPrefix + '*', (req, res) => {
    res.status(404).send('Bad request endpoint.');
  });

  proxyMiddleware(app, apisPrefix);

};
