import * as fs from 'fs';
import * as _path from 'path';
import proxyMiddleware from '../server/proxy-middleware';

const prefix = __dirname + '/pipeline-data';

const fixedData = require('./fixed-data');

const rocMetadataJsonPath = './eval-output/metadata.json';
const rocDataPath = './eval-output/roc.csv';

const confusionMatrixMetadataJsonPath = './model-output/metadata.json';
const confusionMatrixPath = './model-output/confusion_matrix.csv';

const apisPrefix = '/apis/v1alpha1';

let tensorboardPod = '';

export default (app) => {

  app.set('json spaces', 2);

  app.get(apisPrefix + '/healthz', (req, res) => {
    res.json({
      buildDate: 'now',
      commitHash: 'no_commit_hash',
      version: 'local build',
    });
  });

  app.get(apisPrefix + '/pipelines', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.pipelines);
  });

  app.get(apisPrefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    const p = fixedData.pipelines.filter((p) => p.id === pid);
    res.json(p[0]);
  });

  app.get(apisPrefix + '/pipelines/:pid/jobs', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    const p = fixedData.pipelines.filter((p) => p.id === pid);
    // TODO: Is it guaranteed that j.metadata.name and j.workflow.metadata.name
    // are the same?
    const jobs = p[0].jobs.map((j) => j.metadata);
    res.json(jobs);
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
    // This simply allows us to have multiple mocked graphs.
    const mockJobFileName = pid === 1 ?
      'mock-coinflip-job-runtime.json' : 'mock-xgboost-job-runtime.json';
    res.json(JSON.parse(fs.readFileSync(`./mock-backend/${mockJobFileName}`, 'utf-8')));
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

  proxyMiddleware(app, apisPrefix);

};
