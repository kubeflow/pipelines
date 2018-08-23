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
