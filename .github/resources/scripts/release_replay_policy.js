// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
'use strict';

// Read immutable release-base workflow definitions as data. Only run this
// adapter and its policy dependencies from the trusted publisher checkout.
const fs = require('node:fs');
const path = require('node:path');
const {execFileSync} = require('node:child_process');
const {applicable, loadBaseInventory} = require('./ci_expected_workflows');
const {missingReleasePreCommit} = require('./merge_queue_policy');

async function main() {
  const {branch, parent, paths} = JSON.parse(fs.readFileSync(0, 'utf8'));
  const root = path.resolve(__dirname, '../../..');
  const github = {graphql(query, variables) {
    const args = ['api', 'graphql', '--input', '-'];
    const response = JSON.parse(execFileSync('gh', args, {
      input: JSON.stringify({query, variables}), maxBuffer: 16 * 1024 * 1024,
    }));
    if (response.errors) throw new Error('Incomplete release workflow inventory');
    return response.data;
  }};
  const {inventory} = await loadBaseInventory({
    github, owner: 'kubeflow', repo: 'pipelines', baseSha: parent, root,
    pullRequest: {base: {ref: branch, sha: parent, repo: {full_name: 'kubeflow/pipelines'}}},
  });
  const expected = inventory.workflows.filter(workflow => workflow.pull_request !== null &&
    applicable(workflow.pull_request, branch, paths) &&
    !(workflow.path === '.github/workflows/upgrade-test.yml' && workflow.disabled_for_migration === true));
  const workflows = expected.map(workflow => workflow.path);
  if (!workflows.length || missingReleasePreCommit(branch, workflows)) {
    throw new Error('Incomplete release PR workflow coverage');
  }
  process.stdout.write(JSON.stringify(workflows));
}

main().catch(error => { console.error(error.message); process.exitCode = 1; });
