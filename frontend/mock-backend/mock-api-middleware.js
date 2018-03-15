const fs = require('fs');
const _path = require('path');

const prefix = __dirname + '/pipeline-data';

const fixedData = require('./fixed-data');

const rocMetadataJsonPath = './roc/metadata.json';
const rocDataPath = './roc/roc.csv';

const confusionMatrixMetadataJsonPath = './confusionmatrix/metadata.json';
const confusionMatrixPath = './confusionmatrix/confusion_matrix.csv';

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

  app.set('json spaces', 2);

  app.get('/_api/pipelines', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.pipelines);
  });

  app.get('/_api/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    const p = fixedData.pipelines.filter((p) => p.id === pid);
    res.json(p[0]);
  });

  app.get('/_api/pipelines/:pid/jobs', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    const p = fixedData.pipelines.filter((p) => p.id === pid);
    res.json(p[0].jobs);
  });

  app.get('/_api/pipelines/:pid/jobs/:jname', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    const p = fixedData.pipelines.filter((p) => p.id === pid);
    const jname = req.params.jname;
    const j = p[0].jobs.filter((j) => j.name === jname);
    res.json(j[0]);
  });

  app.get('/_api/pipelines/:pid/jobs/:jname/runtimeTemplates', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.send(fs.readFileSync('./mock-backend/mock-runtime-template.yaml'));
  });

  app.get('/_api/packages', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages);
  });

  app.get('/_api/packages/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    res.send(fs.readFileSync('./mock-backend/mock-template.yaml'));
  });

  app.post('/_api/packages/upload', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages[0]);
  });

  app.get('/_api/artifact/list/:path', (req, res) => {

    const path = decodeURIComponent(req.params.path);

    res.header('Content-Type', 'application/json');
      res.json([
        path + '/file1',
        path + '/file2',
        path + (path.match('analysis$|model$') ? '/metadata.json' : '/file3'),
      ]);
  });

  app.get('/_api/artifact/get/:path', (req, res) => {
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
