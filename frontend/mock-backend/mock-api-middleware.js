const fs = require('fs');
const _path = require('path');

const prefix = __dirname + '/pipeline-data';

const fixedData = require('./fixed-data');

const rocMetadataJsonPath = './roc/metadata.json';
const rocDataPath = './roc/roc.csv';

const confusionMatrixMetadataJsonPath = './confusionmatrix/metadata.json';
const confusionMatrixPath = './confusionmatrix/confusion_matrix.csv';

const apisPrefix = '/apis/v1alpha1';

module.exports = (app) => {

  app.set('json spaces', 2);

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
    const jobs = p[0].jobs.map((j) => ({'name': j.metadata.name, scheduledAt: 0}));
    res.json(jobs);
  });

  app.get(apisPrefix + '/pipelines/:pid/jobs/:jname', (req, res) => {
    res.json({
      job: JSON.parse(fs.readFileSync('./mock-backend/mock-job-runtime.json', 'utf-8')),
    });
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

};
