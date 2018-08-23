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

const chai = require('chai');

/**
 * This file navigates to the component tests page, and waits until the test
 * status is printed into a special div on the document.
 */
describe('component tests', () => {
  it('starts component tests', () => {
    browser.url('/_componentTests');
    browser.waitForText('#testFailures', 60000);

    const failures = browser.getText('#testFailures');
    const failuresText =
        JSON.parse(failures).map(f => '- ' + f.title + ':\n  ' + f.error + '\n').join('\n');
    chai.assert(failures === '[]',
        'component tests failed with the following errors:\n' + failuresText);
  });
});
