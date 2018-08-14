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
