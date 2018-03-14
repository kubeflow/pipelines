const fs = require('fs');

const prefix = __dirname + '/pipeline-data';

const fixedData = require('./fixed-data');

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

  app.get('/_api/pipelines/:pid/job/:jname', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pid = Number.parseInt(req.params.pid);
    const p = fixedData.pipelines.filter((p) => p.id === pid);
    const jname = req.params.jname;
    const j = p[0].jobs.filter((j) => j.name === jname);
    res.json(j[0]);
  });

  app.get('/_api/packages', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages);
  });

  app.get('/_api/packages/:pid/template', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    res.send(fs.readFileSync('./mock-backend/mock-template.yaml'));
  });

  app.post('/_api/packages/upload', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json(fixedData.packages[0]);
  });

  app.get('/_api/artifact/list/:path', (req, res) => {
    res.header('Content-Type', 'application/json');
    res.json([
      req.params.path + '/file1',
      req.params.path + '/file2',
      req.params.path + '/file3',
      req.params.path + '/file4',
    ]);
  });

  app.get('/_api/artifact/get/:path', (req, res) => {
    res.send('This is a text artifact file.');
  });
};
