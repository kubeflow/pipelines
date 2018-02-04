const fs = require('fs');

// Need the full path because this is used in the setup only, which
// will be called from webpack.config.js at root level.
const mockDb = fs.readFileSync('mock-backend/mock-db.json');
const mockJson = JSON.parse(mockDb);

function setup(app) {
  app.get('/_templates', (_, res) => {
    res.json(mockJson._templates);
  });

  app.get('/_instances', (_, res) => {
    res.json(mockJson._templates);
  });
}

module.exports = setup;
