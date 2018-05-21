import '../../src/index';

/**
 * This is a wrapper of the main index.ts entrypoint that loads the components
 * test runner iff the current page is _componentTests. This is separated into
 * its own file so that navigating to _componentTests in production can never
 * get to the tests, and so that the production bundle does not contain the
 * tests in the first place.
 */

if (location.pathname === '/_componentTests') {
  import('./test-runner');
}
