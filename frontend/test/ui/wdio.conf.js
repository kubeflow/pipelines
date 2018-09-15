// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const path = require('path');
const VisualRegressionCompare = require('wdio-visual-regression-service/compare');

const headless = process.env.HEADLESS_UI_TESTS == '1' || process.env.HEADLESS_UI_TESTS == 'true';
// If you want to run a single suite, using the following:
// SINGLE_SUITE=./ui/visual-regression/[some-suite].spec.js ./node_modules/.bin/wdio ui/wdio.conf.js
const singleSuite = process.env.SINGLE_SUITE;

function getScreenshotName(basePath) {
  return function (context) {
    const testName = context.test.title.replace(/ /g, '-');
    const suiteName = path.parse(context.test.file).name;
    const browserViewport = context.meta.viewport;
    const browserWidth = browserViewport.width;
    const browserHeight = browserViewport.height;

    return path.join(basePath, `${suiteName}/${testName}_${browserWidth}x${browserHeight}.png`);
  };
}

exports.config = {
  maxInstances: 10,
  baseUrl: 'http://localhost:3000',
  capabilities: [{
    maxInstances: 5,
    browserName: 'chrome',
    chromeOptions: {
      args: headless ? ['--headless', '--disable-gpu'] : [],
    },
  }],
  coloredLogs: true,
  connectionRetryCount: 3,
  connectionRetryTimeout: 90000,
  deprecationWarnings: false,
  framework: 'mocha',
  mochaOpts: {
    timeout: 10000,
  },
  logLevel: 'silent',
  plugins: {
    'wdio-webcomponents': {},
  },
  reporters: ['dot'],
  screenshotPath: './ui/visual-regression/errorShots/',
  services: ['selenium-standalone', 'visual-regression'],
  specs: [
    singleSuite || './ui/visual-regression/*.spec.js',
  ],
  suites: {
    // These suites are separated to avoid race conditions between the e2e tests (which can modify
    // backend state) and the visual regression tests. Because of this, only the visual regression
    // tests will be run if `wdio ui/wdio.conf.js` is called. To run mock e2e tests, use:
    // `wdio ui/wdio.conf.js --suite mockEndToEnd`.
    mockEndToEnd: [
      './ui/end-to-end/*.spec.js',
    ],
    visualRegression: [
      './ui/visual-regression/*.spec.js',
    ],
  },
  sync: true,
  visualRegression: {
    compare: new VisualRegressionCompare.LocalCompare({
      referenceName: getScreenshotName(path.join(process.cwd(), 'ui/visual-regression/screenshots/reference')),
      screenshotName: getScreenshotName(path.join(process.cwd(), 'ui/visual-regression/screenshots/screen')),
      diffName: getScreenshotName(path.join(process.cwd(), 'ui/visual-regression/screenshots/diff')),
      misMatchTolerance: 0,
    }),
    viewports: [{ width: 1024, height: 768 }],
  },
  waitforTimeout: 10000,
}
