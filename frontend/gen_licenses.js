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
const fs = require('fs');
const licenseChecker = require('license-checker');
const path = require('path');

const start = path.resolve(process.cwd(), process.argv[2]);
let licenseMissing = 0;
const whitelist = ['@kubernetes/client-node'];

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
      if (whitelist.indexOf(k) && fs.existsSync('third_party/' + k)) {
        licenseFile = path.resolve(__dirname, 'third_party/' + k + '/LICENSE');
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
