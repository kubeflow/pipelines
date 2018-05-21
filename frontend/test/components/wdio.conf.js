const path = require('path');
const headless = !!process.env.HEADLESS_UI_TESTS;

exports.config = {
  baseUrl: 'http://localhost:3000',
  capabilities: [{
    browserName: 'chrome',
    chromeOptions: {
      args: headless ? ['--headless', '--disable-gpu', '--window-size=1024,800'] : [],
    },
  }],
  framework: 'jasmine',
  logLevel: 'silent',
  services: ['selenium-standalone'],
  specs: [
    './components/test.js',
  ],
  sync: true,
  waitforTimeout: 10000,
}
