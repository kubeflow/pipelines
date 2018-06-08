const path = require('path');

exports.config = {
  baseUrl: 'http://localhost:3000',
  capabilities: [{
    browserName: 'chrome',
    chromeOptions: {
      args: ['--headless', '--disable-gpu', '--window-size=1024,800'],
    },
  }],
  framework: 'jasmine',
  jasmineNodeOpts: {
    defaultTimeoutInterval: 25000,
  },
  logLevel: 'silent',
  services: ['selenium-standalone'],
  specs: [
    './components/test.js',
  ],
  sync: true,
  waitforTimeout: 10000,
}
