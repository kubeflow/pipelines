import { } from 'mocha';

const failures = [];
/**
 * An extension of HTML reporter to collect test failures into an array, while
 * at the same time rendering the HTML test status.
 */
class CustomReporter extends Mocha.reporters.HTML {
  constructor(runner: any) {
    super(runner);
    runner.on('fail', (test, err) => {
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
import './pipeline-new-test';

mocha.run(() => {
  // This div can be waited on by the browser automation script to identify the
  // end of testing and get the list of failures.
  document.querySelector('#testFailures').innerHTML = JSON.stringify(failures);
});
