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
const headless = !(process.env.HEADLESS == '0' || process.env.HEADLESS == 'false');

const seleniumHost = process.env.SELENIUM_HOST || '127.0.0.1';
const seleniumPort = Number(process.env.SELENIUM_PORT || 4444);
const baseUrl = process.env.KFP_BASE_URL || 'http://localhost:3000';
const junitOutputDir = process.env.WDIO_JUNIT_OUTPUT_DIR || './';
const specsFromEnv = process.env.WDIO_SPECS
  ? process.env.WDIO_SPECS.split(',')
      .map(spec => spec.trim())
      .filter(Boolean)
  : [];
const specs = specsFromEnv.length
  ? specsFromEnv
  : ['./helloworld.spec.js', './tensorboard-example.spec.js'];
const chromeArgs = ['--disable-gpu', '--no-sandbox', '--disable-dev-shm-usage'];
if (headless) {
  chromeArgs.unshift('--headless');
}

exports.config = {
  runner: 'local',
  hostname: seleniumHost,
  port: seleniumPort,
  maxInstances: 1,
  baseUrl,
  capabilities: [
    {
      maxInstances: 1,
      browserName: 'chrome',
      'goog:chromeOptions': {
        args: chromeArgs,
      },
    },
  ],
  connectionRetryCount: 3,
  connectionRetryTimeout: 90000,
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
        outputDir: junitOutputDir,
        outputFileFormat: function (options) {
          return 'junit_FrontendIntegrationTestOutput.xml';
        },
      },
    ],
  ],
  services: [],
  specs,
  waitforTimeout: 10000,
};
