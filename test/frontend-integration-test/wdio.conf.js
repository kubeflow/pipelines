// Copyright 2018 The Kubeflow Authors
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

const debug = process.env.DEBUG == '1' || process.env.DEBUG == 'true';

exports.config = {
  runner: 'local',
  host: '127.0.0.1',
  port: 4444,
  maxInstances: 1,
  baseUrl: 'http://localhost:3000',
  capabilities: [
    {
      maxInstances: 1,
      browserName: 'chrome',
      'goog:chromeOptions': {
        args: ['--headless', '--disable-gpu'],
      },
    },
  ],
  coloredLogs: true,
  connectionRetryCount: 3,
  connectionRetryTimeout: 90000,
  deprecationWarnings: false,
  framework: 'mocha',
  mochaOpts: {
    ui: 'bdd',
    // units: ms
    timeout: 1200000,
  },
  logLevel: debug ? 'info' : 'silent',
  reporters: [
    'spec',
    [
      'junit',
      {
        outputDir: './',
        outputFileFormat: function (options) {
          return 'junit_FrontendIntegrationTestOutput.xml';
        },
      },
    ],
  ],
  services: debug ? [['selenium-standalone', { drivers: { chrome: 'latest' } }]] : [],
  specs: ['./helloworld.spec.js'],
  sync: true,
  waitforTimeout: 10000,
};
