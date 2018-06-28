This directory contains tests for the ML Pipelines frontend project.

You can run tests separately by doing `npm run unit-tests`,
`npm run visual-regression-tests`, and `npm run mock-e2e-tests`.
You can run all tests using `npm test`.

## Unit Tests
These are written in Typescript in order to reference tested code directly in
node.js with no browser involved to keep them very fast. They test standalone
library code under /frontend/src/lib.

## UI Tests
These are written in Javascript to skip the extra transpile step, since we
don't see much value for types within these tests. They use the
[webdriver.io](http://webdriver.io) framework to automate browser functions.

UI tests are run in headless Chrome by default if they're run using `npm test`.
In order to see the browser, you can do `npm run visual-regression-tests`,
which reads two environment variables as parameters:
- HEADLESS_UI_TESTS: by default false, but is set to true when using `npm
test`.
- SINGLE_SUITE: can be used to run a single UI suite. For example:
`export SINGLE_SUITE=ui/visual-regression/pipeline-details.spec.js && npm run visual-regression-tests`.

**It is recommended to run the tests with HEADLESS_UI_TESTS=1 unless debugging to
reduce flakiness.**

### Visual Regression Tests
These tests make use of webdriver.io's visual regression service to compare
screenshots taken during tests against golden shots under
frontend/test/ui/visual-regression/screenshots/reference.

These tests can be run via:
`npm run visual-regression-tests`

### Mock End-to-End Tests
Unlike the visual regression tests, these tests actually modify the backend state
(mocked) and do not focus on screenshots.

These tests can be run via:
`npm run mock-e2e-tests`
