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

import { Runner, Test } from 'mocha';

const failures: any[] = [];
/**
 * An extension of HTML reporter to collect test failures into an array, while
 * at the same time rendering the HTML test status.
 */
class CustomReporter extends Mocha.reporters.HTML {
  constructor(runner: Runner) {
    super(runner);
    runner.on('fail', (test: Test, err: Error) => {
      failures.push({ title: test.fullTitle(), error: err.message });
    });
  }
}

const mochaOptions: MochaSetupOptions = {
  reporter: CustomReporter,
  ui: 'bdd',
};

mocha.setup(mochaOptions);

import './item-list-test';
import './job-details-test';
import './job-list-test';
import './job-new-test';
import './job-schedule-test';
import './popup-dialog-test';
import './run-details-tests';
import './table-viewer-test';

mocha.run(() => {
  // This div can be waited on by the browser automation script to identify the
  // end of testing and get the list of failures.
  const failuresElement = document.querySelector('#testFailures');
  if (failuresElement) {
    failuresElement.innerHTML = JSON.stringify(failures);
  } else {
    // tslint:disable-next-line:no-console
    console.error('Could not find HTML element with ID: "testFailures"');
  }
});
