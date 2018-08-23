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

exports.config = {
  maxInstances: 1,
  baseUrl: 'http://localhost:3000',
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
  framework: 'mocha',
  host: '127.0.01',
  port: 4444,
  mochaOpts: {
    timeout: 100000,
  },
  logLevel: 'silent',
  plugins: {
    'wdio-webcomponents': {},
  },
  reporters: ['dot', 'junit'],
  reporterOptions: {
    junit: {
      outputDir: './',
      outputFileFormat: () => 'junit_E2eTestOutput.xml',
    },
  },
  specs: [
    './e2e.spec.js',
  ],
  sync: true,
  waitforTimeout: 10000,
}
