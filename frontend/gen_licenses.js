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

/*
 * Usage: node gen_licenses <path_to_package_root>
 * Generates a dependency-licenses.txt file containing all packages, with a
 * separate entry for each containing its name and the text content of its
 * LICENSE file from its distributed package. If the file does not exist, the
 * script will exit with an error (equal to # of such packages). Packages can
 * be whitelisted by listing them in the `whitelist` array, and adding an entry
 * for them under ./third_party/<pkg_name>/LICENSE.
 */
// tslint:disable:no-console
const fs = require('fs');
const licenseChecker = require('license-checker');
const path = require('path');

const start = path.resolve(process.cwd(), process.argv[2]);
let licenseMissing = 0;
const whitelist = new Map([
  ['@jest/environment', 'third_party/jest/LICENSE'],
  ['@jest/reporters', 'third_party/jest/LICENSE'],
  ['@jest/test-sequencer', 'third_party/jest/LICENSE'],
  ['@jest/transform', 'third_party/jest/LICENSE'],
  ['eslint-module-utils', 'third_party/eslint/LICENSE'],
  ['@kubernetes/client-node', 'third_party/@kubernetes/client-node/LICENSE'],
  ['expect', 'third_party/jest/LICENSE'],
  ['jest-config', 'third_party/jest/LICENSE'],
  ['jest-diff', 'third_party/jest/LICENSE'],
  ['jest-environment-jsdom', 'third_party/jest/LICENSE'],
  ['jest-environment-node', 'third_party/jest/LICENSE'],
  ['jest-get-type', 'third_party/jest/LICENSE'],
  ['jest-haste-map', 'third_party/jest/LICENSE'],
  ['jest-jasmine2', 'third_party/jest/LICENSE'],
  ['jest-matcher-utils', 'third_party/jest/LICENSE'],
  ['jest-message-util', 'third_party/jest/LICENSE'],
  ['jest-regex-util', 'third_party/jest/LICENSE'],
  ['jest-resolve-dependencies', 'third_party/jest/LICENSE'],
  ['jest-resolve', 'third_party/jest/LICENSE'],
  ['jest-runner', 'third_party/jest/LICENSE'],
  ['jest-runtime', 'third_party/jest/LICENSE'],
  ['jest-snapshot', 'third_party/jest/LICENSE'],
  ['jest-util', 'third_party/jest/LICENSE'],
  ['mamacro', 'third_party/mamacro/LICENSE'],
]);

licenseChecker.init({
  production: true,
  start: path.resolve(__dirname, start),
}, (err, json) => {
  if (err) {
    throw new Error(err);
  } else {
    let output = '';
    Object.keys(json).forEach(k => {
      let licenseFile = json[k].licenseFile;
      const packageName = k.substr(0, k.indexOf('@', 1));
      const whitelistEntry = whitelist.get(packageName);
      if (!!whitelistEntry && fs.existsSync(whitelistEntry)) {
        licenseFile = path.resolve(__dirname, whitelistEntry);
      }
      if (!licenseFile) {
        console.error('License file not found for package: ', k);
        licenseMissing++;
        return;
      }
      const licenseText = fs.readFileSync(licenseFile, 'utf-8');
      output += 'Package: ' + k + '\n';
      output += licenseText;
      output += '\n-----------------------------------------\n\n';
    });
    const outFile = path.resolve(__dirname, start, 'dependency-licenses.txt');
    fs.writeFileSync(outFile, output, 'utf-8');
    console.log('Scanned ' + Object.keys(json).length + ' packages successfully.');
  }

  process.exit(licenseMissing);
});
