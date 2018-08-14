const path = require('path');

exports.config = {
  baseUrl: 'http://localhost:3000',
  capabilities: [{
    browserName: 'chrome',
    chromeOptions: {
      args: ['--headless', '--disable-gpu', '--window-size=1024,800'],
    },
  }],
  framework: 'mocha',
  logLevel: 'silent',
  mochaOpts: {
    timeout: 60000,
  },
  services: ['selenium-standalone'],
  specs: [
    './components/test.js',
  ],
  sync: true,
  waitforTimeout: 10000,
}
