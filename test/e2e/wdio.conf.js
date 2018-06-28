const path = require('path');

if (!process.env.ML_PIPELINE_UI_SERVICE_HOST || !process.env.ML_PIPELINE_UI_SERVICE_PORT) {
  console.error('This test must run in the ML Pipelines cluster');
  process.exit(1);
}

const frontendAddress =
    `http://${process.env.ML_PIPELINE_UI_SERVICE_HOST}:${process.env.ML_PIPELINE_UI_SERVICE_PORT}`;

exports.config = {
  maxInstances: 1,
  baseUrl: frontendAddress,
  capabilities: [{
    maxInstances: 1,
    browserName: 'chrome',
    chromeOptions: {
      args: ['--headless', '--disable-gpu', '--window-size=1024,768'],
    },
  }],
  coloredLogs: true,
  connectionRetryCount: 3,
  connectionRetryTimeout: 90000,
  deprecationWarnings: false,
  framework: 'jasmine',
  host: '127.0.01',
  port: 4444,
  jasmineNodeOpts: {
    defaultTimeoutInterval: 10000,
  },
  logLevel: 'silent',
  plugins: {
    'wdio-webcomponents': {},
  },
  reporters: ['dot'],
  specs: [
    './e2e.spec.js',
  ],
  sync: true,
  waitforTimeout: 10000,
}
