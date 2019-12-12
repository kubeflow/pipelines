/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// tslint:disable-next-line:no-var-requires
const backstopjs = require('backstopjs');
const url = 'http://localhost:3000';

const config = {
  asyncCaptureLimit: 10,
  asyncCompareLimit: 50,
  debug: false,
  debugWindow: false,
  engine: 'puppeteer',
  engineOptions: {
    args: ['--no-sandbox'],
  },
  id: 'pipelines',
  onReadyScript: 'steps.js',
  paths: {
    bitmaps_reference: 'backstop_data/bitmaps_reference',
    bitmaps_test: 'backstop_data/bitmaps_test',
    ci_report: 'backstop_data/ci_report',
    engine_scripts: 'backstop_data/engine_scripts',
    html_report: 'backstop_data/html_report',
  },
  report: ['browser'],
  scenarios: [
    {
      label: 'initial state',
      url,
    },
    {
      label: 'hover on first row',
      steps: [{ action: 'hover', selector: '.tableRow' }],
      url,
    },
    {
      label: 'select one row',
      steps: [{ action: 'click', selector: '.tableRow' }],
      url,
    },
    {
      label: 'select multiple rows',
      steps: [
        { action: 'click', selector: '.tableRow' },
        { action: 'click', selector: `.tableRow:nth-of-type(2)` },
        { action: 'click', selector: `.tableRow:nth-of-type(5)` },
      ],
      url,
    },
    {
      label: 'open upload dialog',
      steps: [{ action: 'click', selector: '#uploadBtn' }, { action: 'pause' }],
      url,
    },
  ],
  viewports: [{ width: 1024, height: 768 }],
};

backstopjs('test', { config });
