const path = require('path');
const express = require('express');
const app = express();

if (process.argv.length < 3) {
  console.error(`\
Usage: node server.js <static-dir> [port].
       You can specify the API server address using the API_SERVER_ADDRESS env var.`);
  process.exit(1);
}

const staticDir = path.resolve(process.argv[2]);
const port = process.argv[3] || 3000;

const apiServerAddress = process.env.API_SERVER_ADDRESS || 'http://localhost:3001';

app.use(express.static(staticDir));

app.get('/_config/apiServerAddress', (req, res) => {
  res.send(apiServerAddress);
});

app.get('*', (req, res) => {
  res.sendFile(path.resolve(staticDir, 'index.html'));
});

app.listen(3000, () => {
  console.log('Server listening at http://localhost:' + port);
});
