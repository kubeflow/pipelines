const fs = require('fs');

const prefix = __dirname + '/pipeline-data';

const fixedData = require('./fixed-data');
const templateYaml = fs.readFileSync('./mock-backend/template.yaml', { encoding: 'utf-8' });

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
    res.json(fixedData.jobs.filter(j => j._pipelineId === pid));
  });

  app.get('/_api/jobs/:jname', (req, res) => {
    res.header('Content-Type', 'application/json');
    const jname = req.params.jname;
    const j = fixedData.jobs.filter((j) => j.name === jname);
    res.json(j[0]);
  });

  app.get('/_api/jobs/:jid/outputPaths', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.send([
      { 'analyze': 'analysis' },
      { 'transform': 'transform' },
      { 'train': 'model' },
    ]);
  });

  app.get('/_api/packages', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages);
  });

  app.get('/_api/packages/:pid/template', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    res.send(templateYaml);
  });

  app.post('/_api/packages/upload', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages[0]);
  });

};
