const path = require('path');
const express = require('express');
const app = express();

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

app.get('*', (req, res) => {
  res.sendFile(path.resolve(staticDir, 'index.html'));
});

app.listen(port, () => {
  console.log('Server listening at http://localhost:' + port);
});
