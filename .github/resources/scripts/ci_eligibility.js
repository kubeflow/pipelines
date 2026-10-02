// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

'use strict';

const fs = require('node:fs');

function loadMembers(path = process.env.KUBEFLOW_MEMBERS_FILE) {
  if (!path) {
    throw new Error('Kubeflow ACL membership file is missing. Run the trusted membership lookup and set KUBEFLOW_MEMBERS_FILE.');
  }
  let members;
  try {
    members = JSON.parse(fs.readFileSync(path, 'utf8'));
  } catch (error) {
    throw new Error('Could not load Kubeflow ACL membership file. Check the trusted membership lookup and rerun the workflow.', {cause: error});
  }
  if (!Array.isArray(members) || members.length === 0 ||
      members.some(login => typeof login !== 'string' || !/^[A-Za-z0-9][A-Za-z0-9-]{0,38}$/.test(login))) {
    throw new Error('Invalid Kubeflow ACL membership file. Expected a nonempty array of GitHub usernames; rerun the trusted membership lookup.');
  }
  return new Set(members.map(login => login.toLowerCase()));
}

function eligible(pr, members = loadMembers()) {
  const labels = new Set(pr.labels.map(label => label.name));
  return !labels.has('needs-ok-to-test') && (labels.has('ok-to-test') ||
    pr.user.login === 'dependabot[bot]' ||
    members.has(pr.user.login.toLowerCase()));
}

module.exports = {loadMembers, eligible};
