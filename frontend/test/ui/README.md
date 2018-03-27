This directory contains tests for the ML Pipelines frontend project.

You can run tests separately by doing `npm run unit-tests` or `npm run
ui-tests`. You can run all tests using `npm test`.

## Unit Tests
These are written in Typescript in order to reference tested code directly in
node.js, no browser involved, in order to keep them very fast. They test
standalone library code under /frontend/src/lib.

## UI Tests
These are written in Javascript to skip the extra transpile step, since we
don't see much value for types within these tests. They use the
[webdriver.io](http://webdriver.io) framework to automate browser functions,
making use of its visual regression service to compare screenshots taken
during tests against golden shots under
frontend/test/ui/screenshots/reference.
